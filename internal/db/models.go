package db

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AuthStore interface {
	// User
	InsertUser(ctx context.Context, u *UserRow) error
	FindUserByID(ctx context.Context, id uuid.UUID) (*UserRow, error)
	FindUserByEmail(ctx context.Context, email string) (*UserRow, error)
	FindUserByUsername(ctx context.Context, username string) (*UserRow, error)
	// UpdateUser(ctx context.Context, u *UserRow) error
	UpdateUserPassword(ctx context.Context, id uuid.UUID, password_hash []byte) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	// Password Reset
	InsertPasswordReset(ctx context.Context, p *PasswordResetRow) error
	FindPasswordReset(ctx context.Context, token uuid.UUID) (*PasswordResetRow, error)
	DeletePasswordReset(ctx context.Context, token uuid.UUID) error
	// User Activation
	InsertActivation(ctx context.Context, p *ActivationRow) error
	FindActivation(ctx context.Context, token uuid.UUID) (*ActivationRow, error)
	DeleteActivation(ctx context.Context, token uuid.UUID) error

	// Session
	InsertSession(ctx context.Context, s *SessionRow) error
	GetSession(ctx context.Context, id uuid.UUID) (*SessionRow, error)
	UpdateSession(ctx context.Context, s *SessionRow) error
	DeleteSession(ctx context.Context, id uuid.UUID) error

	// Both
	GetSessionAndUser(ctx context.Context, sessionID uuid.UUID) (*SessionRow, *UserRow, error)
}

type OAuthStore interface {
	// Auth Code
	InsertAuthCode(ctx context.Context, a *AuthCodeRow) error
	GetAuthCode(ctx context.Context, code []byte) (*AuthCodeRow, error)
	// UpdateAuthCode(ctx context.Context, a *AuthCodeRow) error
	DeleteAuthCode(ctx context.Context, code []byte) error

	// Access Token
	InsertAccessToken(ctx context.Context, a *AccessTokenRow) error
	GetAccessTokenByRefresh(ctx context.Context, refreshToken []byte) (*AccessTokenRow, error)
	DeleteAccessToken(ctx context.Context, token []byte) error

	GetAccessTokenAndUser(ctx context.Context, token []byte) (*UserRow, *AccessTokenRow, error)
}

// UserRow contains all user specific information
type UserRow struct {
	ID           uuid.UUID
	Email        string
	Username     string
	Birthdate    time.Time
	PasswordHash []byte
	Created      time.Time
	LastSeen     time.Time
	Activated    bool
}

// SessionRow contains all session information
type SessionRow struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	LoginTime    time.Time
	LastSeenTime time.Time
	Expires      time.Time
	UserAgent    string
}

// AuthCode is the authorization code of oauth2.0
// code is the primary key
type AuthCodeRow struct {
	Code     []byte    `json:"code"`
	ClientID string    `json:"client_id"`
	UserID   uuid.UUID `json:"user_id"`
	Scope    Scope     `json:"scope"`
	Expires  time.Time `json:"expires"`
}

// AccessToken contains the information to provide user access within oAuth scope
type AccessTokenRow struct {
	Token        []byte    `json:"token"`
	AuthCode     []byte    `json:"auth_code"`
	RefreshToken []byte    `json:"refresh_token"`
	UserID       uuid.UUID `json:"user_id"`
	Created      time.Time `json:"created"`
	Expires      int       `json:"expires"`
}

// ActivationRow contains information for user account activation
type ActivationRow struct {
	Token   uuid.UUID `json:"token"`
	UserID  uuid.UUID `json:"user_id"`
	Expires time.Time `json:"expires"`
}

// PasswordResetRow contains information for user password reset
type PasswordResetRow struct {
	Token   uuid.UUID `json:"token"`
	UserID  uuid.UUID `json:"user_id"`
	Expires time.Time `json:"expires"`
}

// Scope contains identifiers to oAuth permissions
type Scope string

// Scopes of oauth2.0
const (
	Read       Scope = "Read"
	ReadChange Scope = "ReadChange"
)

// Podcast contains information and xml struct tags for podcast
type Podcast struct {
	// REQUIRED
	ID          uuid.UUID
	Title       string
	Description string
	ImageURL    string
	Language    string
	Category    []int
	Explicit    string
	// RECOMMENDED
	Author     string
	LinkURL    string
	OwnerName  string
	OwnerEmail string
	// SITUATIONAL
	Episodic  bool
	Copyright string
	Block     bool
	Complete  bool
	// RSS/OTHER
	PubDate  time.Time
	Keywords string
	Summary  string
	RSSURL   string
}

// Episode holds information about a single episode of a podcast within the rss feed
type Episode struct {
	// REQUIRED
	ID              uuid.UUID
	Title           string
	EnclosureURL    string
	EnclosureLength int64
	EnclosureType   string
	// RECOMMENDED
	PubDate     time.Time
	Description string
	Duration    int64
	LinkURL     string
	ImageURL    string
	ImageTitle  string
	Explicit    string
	// SITUATIONAL
	Episode     int
	Season      int
	EpisodeType string
	//Block       bool
	// OTHER
	Subtitle  string
	Summary   string
	Encoded   string
	PodcastID uuid.UUID
}

type Category struct {
	ID       int
	Name     string
	ParentID int
}

type Subscription struct {
	UserID        uuid.UUID
	PodcastID     uuid.UUID
	CompletedIDs  []uuid.UUID
	InProgressIDs []uuid.UUID
}

type UserEpisode struct {
	UserID       uuid.UUID
	EpisodeID    uuid.UUID
	OffsetMillis int64
	LastSeen     time.Time
	Played       bool
}
