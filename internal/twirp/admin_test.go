// Package TestMain() located in auth_test.go
package twirp

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sschwartz96/syncapod-backend/internal/db_new"
	protos "github.com/sschwartz96/syncapod-backend/internal/gen"
	"github.com/sschwartz96/syncapod-backend/internal/util"
	"github.com/stretchr/testify/require"
	"github.com/twitchtv/twirp"
)

var (
	goTimeRSSURL    = "https://changelog.com/gotime/feed"
	testSeshAdminID pgtype.UUID
)

func setupAdmin() error {
	var err error

	// insert user session to mimic user already authenticated
	queries := db_new.New(dbpg)

	testSeshAdminParams := db_new.InsertSessionParams{
		UserID:       testUserID,
		LoginTime:    util.PGFromTime(time.Now()),
		LastSeenTime: util.PGFromTime(time.Now()),
		Expires:      util.PGFromTime(time.Now().Add(time.Hour)),
		UserAgent:    "testUserAgent",
	}
	testSeshAdmin, err := queries.InsertSession(context.Background(), testSeshAdminParams)
	if err != nil {
		return fmt.Errorf("failed to insert user session: %v", err)
	}
	testSeshAdminID = testSeshAdmin.ID
	return nil
}

func Test_AdminGRPC(t *testing.T) {
	// add metadata for authorization
	header := make(http.Header)
	id, _ := util.StringFromPGUUID(testSeshID)
	header.Set(authTokenKey, id)

	ctx, err := twirp.WithHTTPRequestHeaders(context.Background(), header)
	if err != nil {
		t.Fatalf("Twirp could not add add headers: %v", err)
	}

	client := protos.NewAdminJSONClient("http://localhost:8081", http.DefaultClient, twirp.WithClientPathPrefix("/rpc/admin"))

	// AddPodcast
	addPodRes, err := client.AddPodcast(ctx, &protos.AddPodReq{Url: goTimeRSSURL})
	require.Nil(t, err, "error AddPodcast()")
	require.Equal(t, "Go Time: Golang, Software Engineering", addPodRes.Podcast.Title)

	// RefreshPodcast
	_, err = client.RefreshPodcast(ctx, &protos.RefPodReq{})
	require.Nil(t, err, "error RefreshPodcast()")
}
