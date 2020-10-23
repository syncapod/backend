package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	"go.mongodb.org/mongo-driver/x/mongo/driver/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthController struct {
	authStore  db.AuthStore
	oauthStore db.OAuthStore
}

type Auth interface {
	// Syncapod
	Login()
	Authorize()
	Logout()

	// OAuth
	OAuthRequest() // initial req, sends back "grant" auth code
	OAuthGrant()   // accept granted auth code and return access token
	OAuthToken()   // read token and send back resource
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

// CreateSession creates a session and stores it into database
func CreateSession(userID uuid.UUID, userAgent string, stayLoggedIn bool) (string, error) {
	// determine expires
	var expires time.Duration
	if stayLoggedIn {
		expires = time.Hour * 26280
	} else {
		expires = time.Hour
	}

	// Create key
	key, err := CreateKey(64)
	if err != nil {
		return "", err
	}

	if userAgent == "" {
		userAgent = "unknown"
	}

	// Create Session object
	session := &protos.Session{
		Id:           protos.NewObjectID(),
		UserID:       userID,
		SessionKey:   key,
		LoginTime:    ptypes.TimestampNow(),
		LastSeenTime: ptypes.TimestampNow(),
		Expires:      util.AddToTimestamp(ptypes.TimestampNow(), expires),
		UserAgent:    userAgent,
	}

	// Store session in database
	err = user.UpsertSession(dbClient, session)
	if err != nil {
		return "", err
	}

	return key, nil
}

// CreateKey takes in a key length and returns base64 encoding
func CreateKey(l int) (string, error) {
	key := make([]byte, l)
	_, err := rand.Read(key)
	if err != nil {
		return "", fmt.Errorf("Error creating pseudo-random key: %v", err)
	}
	return base64.URLEncoding.EncodeToString(key)[:l], nil
}

// ValidateSession looks up session key, check if its valid and returns a pointer to the user
// returns error if the key doesn't exist, or has expired
func ValidateSession(dbClient db.Database, key string) (*protos.User, error) {
	// Find the key
	sesh, err := user.FindSession(dbClient, key)
	if err != nil {
		return nil, fmt.Errorf("ValidateSession() error finding session: %v", err)
	}

	// Check if expired
	if sesh.Expires.AsTime().Before(time.Now()) {
		err := user.DeleteSession(dbClient, sesh.Id)
		if err != nil {
			return nil, fmt.Errorf("ValidateSession() (session expired) error deleting session: %v", err)
		}
		return nil, errors.New("ValidateSession() session expired")
	}

	// calculate time to add to expiration
	lastSeen, _ := ptypes.Timestamp(sesh.LastSeenTime)
	timeToAdd := time.Since(lastSeen)

	sesh.LastSeenTime = ptypes.TimestampNow()
	util.AddToTimestamp(sesh.Expires, timeToAdd)
	upsertErr := make(chan error)
	go func() {
		upsertErr <- user.UpsertSession(dbClient, sesh)
	}()

	// Find the user
	u, err := user.FindUserByID(dbClient, sesh.UserID)
	if err != nil {
		return nil, fmt.Errorf("ValidateSession() error finding user: %v", err)
	}

	// check the upsertErr
	err = <-upsertErr
	if err != nil {
		return nil, fmt.Errorf("ValidateSession() error upsert new session: %v", err)
	}

	return u, nil
}
