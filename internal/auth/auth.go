package auth

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sschwartz96/syncapod-backend/internal/db_new"
	"github.com/sschwartz96/syncapod-backend/internal/util"
	"golang.org/x/crypto/bcrypt"
)

type AuthController struct {
	queries *db_new.Queries
	log     *slog.Logger
}

func NewAuthController(queries *db_new.Queries, log *slog.Logger) *AuthController {
	return &AuthController{queries: queries, log: log}
}

// Login queries db for user and validates password.
// On success, it creates session and inserts into db
// returns error if user not found or password is invalid
func (a *AuthController) Login(ctx context.Context, username, password, agent string) (*db_new.User, *db_new.Session, error) {
	user, err := a.findUserByEmailOrUsername(ctx, username)
	if err != nil {
		return nil, nil, fmt.Errorf("AuthController.Login() error finding user: %v", err)
	}
	if !compare(user.PasswordHash, password) {
		return nil, nil, fmt.Errorf("AuthController.Login() error incorrect password")
	}

	sessionInsertParams, err := createInsertSessionParams(user.ID, agent)

	if err != nil {
		return nil, nil, fmt.Errorf("AuthController.Login() error creating session %v", err)
	}
	session, err := a.queries.InsertSession(ctx, *sessionInsertParams)
	if err != nil {
		return nil, nil, fmt.Errorf("AuthController.Login() error inserting new session: %v", err)
	}

	// remove user's password hash
	user.PasswordHash = []byte{}
	return user, &session, nil
}

// Authorize queries db for session via id, validates and returns user info.
// returns error if the session is not found or invalid
func (a *AuthController) Authorize(ctx context.Context, sessionID uuid.UUID) (*db_new.User, error) {
	userSession, err := a.queries.GetSessionAndUser(ctx, util.PGUUID(sessionID))
	if err != nil {
		return nil, fmt.Errorf("AuthController.Authorize() error finding session: %v", err)
	}

	now := time.Now()
	if userSession.Session.Expires.Time.Before(now) {
		go func() {
			err := a.queries.DeleteSession(context.Background(), userSession.Session.ID)
			if err != nil {
				a.log.Warn("error deleting sesion", util.Err(err))
			}
		}()
		return nil, fmt.Errorf("AuthController.Authorize() error: session expired")
	}
	userSession.Session.LastSeenTime = util.PGFromTime(now)
	userSession.Session.Expires = util.PGFromTime(now.Add(time.Hour * 168))
	go func() {
		err := a.queries.UpdateSession(context.Background(), db_new.UpdateSessionParams(userSession.Session))
		if err != nil {
			a.log.Warn("error updating session", util.Err(err))
		}
	}()
	userSession.User.PasswordHash = []byte{}
	return &userSession.User, nil
}

func (a *AuthController) Logout(ctx context.Context, sessionID uuid.UUID) error {
	err := a.queries.DeleteSession(ctx, util.PGUUID(sessionID))
	if err != nil {
		return fmt.Errorf("AuthController.Logout() error deleting session: %v", err)
	}
	return nil
}
func (a *AuthController) CreateUser(ctx context.Context, email, username, pwd string, dob time.Time) (*db_new.User, error) {
	pwdHash, err := hash(pwd)
	if err != nil {
		return nil, fmt.Errorf("AuthController.CreateUser() error hashing password: %v", err)
	}

	newUserParams := db_new.InsertUserParams{
		Email:        email,
		Username:     username,
		Birthdate:    util.PGDateFromTime(dob),
		PasswordHash: pwdHash,
		Created:      util.PGNow(),
		LastSeen:     util.PGNow(),
	}

	newUser, err := a.queries.InsertUser(ctx, newUserParams)
	if err != nil {
		return nil, fmt.Errorf("AuthController.CreateUser() error inserting user into db: %v", err)
	}
	return &newUser, nil
}

// findUserByEmailOrUsername is a helper method for login
// takes in string u which could either be an email address or username
// returns UserRow upon success
func (a *AuthController) findUserByEmailOrUsername(ctx context.Context, u string) (*db_new.User, error) {
	var user db_new.User
	var err error
	if strings.Contains(u, "@") {
		user, err = a.queries.GetUserByEmail(ctx, u)
	} else {
		user, err = a.queries.GetUserByUsername(ctx, u)
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Hash takes pwd string and returns hash type string
func hash(password string) ([]byte, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return nil, err
	}
	return hash, nil
}

// Compare takes a password and hash compares and returns true for match
func compare(hash []byte, password string) bool {
	return bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil
}

// createInsertSessionParams creates a InsertSessionParams object ready to be inserted
func createInsertSessionParams(userID pgtype.UUID, agent string) (*db_new.InsertSessionParams, error) {
	now := time.Now()
	nowPG := util.PGFromTime(now)
	expiresPG := util.PGFromTime(now.Add(time.Hour * 168))

	return &db_new.InsertSessionParams{
		UserID:       userID,
		Expires:      expiresPG,
		LastSeenTime: nowPG,
		LoginTime:    nowPG,
		UserAgent:    agent,
	}, nil
}
