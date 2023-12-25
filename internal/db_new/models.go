// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.24.0

package db_new

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type Accesstoken struct {
	Token        []byte
	AuthCode     []byte
	RefreshToken []byte
	UserID       pgtype.UUID
	Created      pgtype.Timestamptz
	Expires      int32
}

type Authcode struct {
	Code     []byte
	ClientID string
	UserID   pgtype.UUID
	Scope    string
	Expires  pgtype.Timestamptz
}

type Category struct {
	ID       int32
	Name     string
	ParentID int32
}

type Episode struct {
	ID              pgtype.UUID
	Title           string
	EnclosureUrl    string
	EnclosureLength int64
	EnclosureType   string
	PubDate         pgtype.Timestamptz
	Description     string
	Duration        int64
	LinkUrl         string
	ImageUrl        string
	ImageTitle      string
	Explicit        string
	Episode         int32
	Season          int32
	EpisodeType     string
	Subtitle        string
	Summary         string
	Encoded         string
	PodcastID       pgtype.UUID
}

type Podcast struct {
	ID          pgtype.UUID
	Title       string
	Description string
	ImageUrl    string
	Language    string
	Category    []int32
	Explicit    string
	Author      string
	LinkUrl     string
	OwnerName   string
	OwnerEmail  string
	Episodic    pgtype.Bool
	Copyright   string
	Block       pgtype.Bool
	Complete    pgtype.Bool
	PubDate     pgtype.Timestamptz
	Keywords    string
	Summary     string
	RssUrl      string
}

type PodcastsSearch struct {
	ID        int32
	PodcastID pgtype.UUID
	Search    interface{}
}

type Session struct {
	ID           pgtype.UUID
	UserID       pgtype.UUID
	LoginTime    pgtype.Timestamptz
	LastSeenTime pgtype.Timestamptz
	Expires      pgtype.Timestamptz
	UserAgent    string
}

type Subscription struct {
	UserID        pgtype.UUID
	PodcastID     pgtype.UUID
	CompletedIds  []pgtype.UUID
	InProgressIds []pgtype.UUID
}

type User struct {
	ID           pgtype.UUID
	Email        string
	Username     string
	Birthdate    pgtype.Date
	PasswordHash []byte
	Created      pgtype.Timestamptz
	LastSeen     pgtype.Timestamptz
}

type Userepisode struct {
	UserID       pgtype.UUID
	EpisodeID    pgtype.UUID
	OffsetMillis pgtype.Int8
	LastSeen     pgtype.Timestamptz
	Played       pgtype.Bool
}
