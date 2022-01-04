package auth

import (
	"context"
	"fmt"
	"log"
	netmail "net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	"github.com/sschwartz96/syncapod-backend/internal/mail"
	"golang.org/x/crypto/bcrypt"
)

type Auth interface {
	// Syncapod
	Login(ctx context.Context, username, password, agent string) (*db.UserRow, *db.SessionRow, error)
	Authorize(ctx context.Context, sessionID uuid.UUID) (*db.UserRow, error)
	Logout(ctx context.Context, sessionID uuid.UUID) error
	CreateUser(ctx context.Context, email, username, pwd string, dob time.Time) (*db.UserRow, error)
	ActivateUser(ctx context.Context, token uuid.UUID) (*db.ActivationRow, error)
	ResetPassword(ctx context.Context, emailOrUsername string) error
	ValidatePasswordResetToken(ctx context.Context, token uuid.UUID) (*db.PasswordResetRow, error)

	// OAuth
	CreateAuthCode(ctx context.Context, userID uuid.UUID, clientID string) (*db.AuthCodeRow, error)
	CreateAccessToken(ctx context.Context, authCode *db.AuthCodeRow) (*db.AccessTokenRow, error)
	ValidateAuthCode(ctx context.Context, code string) (*db.AuthCodeRow, error)
	ValidateAccessToken(ctx context.Context, token string) (*db.UserRow, error)
	ValidateRefreshToken(ctx context.Context, token string) (*db.AccessTokenRow, error)
}

type AuthController struct {
	authStore  db.AuthStore
	oauthStore db.OAuthStore
	mailer     mail.MailQueuer
}

func NewAuthController(aStore db.AuthStore, oStore db.OAuthStore, mailer *mail.Mailer) *AuthController {
	return &AuthController{authStore: aStore, oauthStore: oStore, mailer: mailer}
}

// Login queries db for user and validates password.
// On success, it creates session and inserts into db
// returns error if user not found or password is invalid
func (a *AuthController) Login(ctx context.Context, username, password, agent string) (*db.UserRow, *db.SessionRow, error) {
	user, err := a.findUserByEmailOrUsername(ctx, username)
	if err != nil {
		return nil, nil, fmt.Errorf("AuthController.Login() error finding user: %v", err)
	}
	if !compare(user.PasswordHash, password) {
		return nil, nil, fmt.Errorf("AuthController.Login() error incorrect password")
	}
	user.PasswordHash = []byte{}
	session, err := createSession(user.ID, agent)
	if err != nil {
		return nil, nil, fmt.Errorf("AuthController.Login() error creating new session: %v", err)
	}

	err = a.authStore.InsertSession(context.Background(), session)
	if err != nil {
		return nil, nil, fmt.Errorf("AuthController.Login() error inserting new session: %v", err)
	}
	return user, session, nil
}

// Authorize queries db for session via id, validates and returns user info.
// returns error if the session is not found or invalid
func (a *AuthController) Authorize(ctx context.Context, sessionID uuid.UUID) (*db.UserRow, error) {
	session, user, err := a.authStore.GetSessionAndUser(ctx, sessionID)
	now := time.Now()
	if err != nil {
		return nil, fmt.Errorf("AuthController.Authorize() error finding session: %v", err)
	}
	if session.Expires.Before(now) {
		go func() {
			err := a.authStore.DeleteSession(context.Background(), sessionID)
			if err != nil {
				log.Printf("AuthController.Authorize() error deleting session: %v\n", err)
			}
		}()
		return nil, fmt.Errorf("AuthController.Authorize() error: session expired")
	}
	session.LastSeenTime = now
	session.Expires = now.Add(time.Hour * 168)
	go func() {
		err := a.authStore.UpdateSession(context.Background(), session)
		if err != nil {
			log.Printf("AuthController.Authorize() error updating session: %v\n", err)
		}
	}()
	user.PasswordHash = []byte{}
	return user, nil
}

func (a *AuthController) Logout(ctx context.Context, sessionID uuid.UUID) error {
	err := a.authStore.DeleteSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("AuthController.Logout() error deleting session: %v", err)
	}
	return nil
}

func (a *AuthController) CreateUser(ctx context.Context, email, username, pwd string, dob time.Time) (*db.UserRow, error) {
	// ensure user is older than 18
	if (time.Now().Unix() - dob.Unix()) < 567648000 {
		return nil, fmt.Errorf("user must be at least 18 years old")
	}

	// ensure password >= 15 characters
	if len(pwd) < 15 {
		return nil, fmt.Errorf("password has to be at least than 15 characters")
	}

	// validate email address proper email
	address, err := netmail.ParseAddress(email)
	if err != nil || len(address.Name) > 0 {
		return nil, fmt.Errorf("invalid email")
	}

	pwdHash, err := hash(pwd)
	if err != nil {
		return nil, fmt.Errorf("AuthController.CreateUser() error hashing password: %v", err)
	}

	newUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("AuthController.CreateUser() error genearting new UUID: %v", err)
	}

	newUser := &db.UserRow{
		ID:           newUUID,
		Email:        email,
		Username:     username,
		Birthdate:    dob,
		PasswordHash: pwdHash,
		Created:      time.Now(),
		LastSeen:     time.Now(),
		Activated:    false,
	}
	err = a.authStore.InsertUser(ctx, newUser)
	if err != nil {
		if err.Error() == "duplicate key value violates unique constraint \"users_email_key\"" {
			return nil, fmt.Errorf("email taken")
		}
		if err.Error() == "duplicate key value violates unique constraint \"users_username_key\"" {
			return nil, fmt.Errorf("username taken")
		}
		return nil, fmt.Errorf("AuthController.CreateUser() error inserting user into db: %w", err)
	}

	activationToken, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("AuthController.CreateUser() error generating UUID: %w", err)
	}
	activationRow := &db.ActivationRow{Token: activationToken, UserID: newUser.ID, Expires: time.Now().Add(time.Hour * 24)}
	err = a.authStore.InsertActivation(ctx, activationRow)
	if err != nil {
		// TODO: remove user from database???
		// this is a very unexpected edge case
		return nil, fmt.Errorf("AuthController.CreateUser() error inserting activation code: %w", err)
	}

	a.mailer.Queue(newUser.Email, "Please Activate Your syncapod.com Account", "Token: "+activationToken.String()) // TODO: create html email template

	return newUser, nil
}

// ActivateUser finds the activation token, if valid, updates the user's activated field
func (a *AuthController) ActivateUser(ctx context.Context, token uuid.UUID) (*db.ActivationRow, error) {
	activationRow, err := a.authStore.FindActivation(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("AuthController.ActivateUser() error finding activation row: %w", err)
	}

	// check if expired
	if time.Now().After(activationRow.Expires) {
		return nil, fmt.Errorf("AuthController.ActivateUser() error: activation token expired")
	}

	err = a.authStore.UpdateUserActivated(ctx, activationRow.UserID)
	if err != nil {
		return nil, fmt.Errorf("AuthController.ActivateUser() error activating user: %w", err)
	}

	err = a.authStore.DeleteActivation(ctx, activationRow.Token)
	if err != nil {
		return nil, fmt.Errorf("AuthController.ActivateUser() error deleting activation row: %w", err)
	}
	return activationRow, nil
}

func (a *AuthController) ResetPassword(ctx context.Context, email string) error {
	user, err := a.authStore.FindUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("AuthController.ResetPassword() error finding user by email: %w", err)
	}
	passwordResetToken, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("AuthController.ResetPassword() error generating token: %w", err)
	}

	passwordResetRow := &db.PasswordResetRow{Token: passwordResetToken, UserID: user.ID, Expires: time.Now().Add(time.Hour * 2)}
	err = a.authStore.InsertPasswordReset(ctx, passwordResetRow)
	if err != nil {
		return fmt.Errorf("AuthController.ResetPassword() error inserting password reset row: %w", err)
	}

	// TODO: template email
	a.mailer.Queue(user.Email, "Reset Password", "Click this link to reset your syncapod.com password\nToken: "+passwordResetToken.String())

	return nil
}

// ValidatePasswordResetToken just proxies to the authStore.FindPasswordReset
func (a *AuthController) ValidatePasswordResetToken(ctx context.Context, token uuid.UUID) (*db.PasswordResetRow, error) {
	return a.authStore.FindPasswordReset(ctx, token)
}

// findUserByEmailOrUsername is a helper method for login
// takes in string u which could either be an email address or username
// returns UserRow upon success
func (a *AuthController) findUserByEmailOrUsername(ctx context.Context, u string) (*db.UserRow, error) {
	var user *db.UserRow
	var err error
	if strings.Contains(u, "@") {
		user, err = a.authStore.FindUserByEmail(ctx, u)
	} else {
		user, err = a.authStore.FindUserByUsername(ctx, u)
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

// Hash takes pwd string and returns hash type string
func hash(password string) ([]byte, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		fmt.Printf("Hash(), error hashing password: %v", err)
		return nil, err
	}
	return hash, nil
}

// Compare takes a password and hash compares and returns true for match
func compare(hash []byte, password string) bool {
	return bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil
}

// createSession creates a session
func createSession(userID uuid.UUID, agent string) (*db.SessionRow, error) {
	now := time.Now()
	newUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	return &db.SessionRow{
		ID:           newUUID,
		UserID:       userID,
		Expires:      now.Add(time.Hour * 168),
		LastSeenTime: now,
		LoginTime:    now,
		UserAgent:    agent,
	}, nil
}
