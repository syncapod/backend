package auth

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	"golang.org/x/crypto/bcrypt"
)

type Auth interface {
	// Syncapod
	Login(ctx context.Context, username, password, agent string) (*db.UserRow, *db.SessionRow, error)
	Authorize(ctx context.Context, sessionID uuid.UUID) (*db.UserRow, error)
	Logout(ctx context.Context, sessionID uuid.UUID) error
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
}

func NewAuthController(aStore db.AuthStore, oStore db.OAuthStore) *AuthController {
	return &AuthController{authStore: aStore, oauthStore: oStore}
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
	session := createSession(user.ID, agent)
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

// findUserByEmailOrUsername is a helper method for login
// takes in string u which could either be an email address or username
// returns UserRow upon success
func (a *AuthController) findUserByEmailOrUsername(ctx context.Context, u string) (*db.UserRow, error) {
	var user *db.UserRow
	var err error
	if strings.Contains(u, "@") {
		user, err = a.authStore.GetUserByEmail(ctx, u)
	} else {
		user, err = a.authStore.GetUserByUsername(ctx, u)
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
func createSession(userID uuid.UUID, agent string) *db.SessionRow {
	now := time.Now()
	return &db.SessionRow{
		ID:           uuid.New(),
		UserID:       userID,
		Expires:      now.Add(time.Hour * 168),
		LastSeenTime: now,
		LoginTime:    now,
		UserAgent:    agent,
	}
}
