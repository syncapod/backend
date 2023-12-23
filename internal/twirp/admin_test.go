// Package TestMain() located in auth_test.go
package twirp

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sschwartz96/syncapod-backend/internal/db_new"
	protos "github.com/sschwartz96/syncapod-backend/internal/gen"
	"github.com/sschwartz96/syncapod-backend/internal/util"
	"github.com/stretchr/testify/require"
	"github.com/twitchtv/twirp"
)

var (
	goTimeRSSURL = "https://changelog.com/gotime/feed"

	testSeshAdmin = db_new.Session{ID: util.PGUUID(uuid.New()),
		UserID:       testUser.ID,
		LoginTime:    util.PGFromTime(time.Now()),
		LastSeenTime: util.PGFromTime(time.Now()),
		Expires:      util.PGFromTime(time.Now().Add(time.Hour)),
		UserAgent:    "testUserAgent",
	}
)

func setupAdmin() error {
	var err error

	// insert user session to mimic user already authenticated
	queries := db_new.New(dbpg)
	if _, err = queries.InsertSession(context.Background(), db_new.InsertSessionParams(testSeshAdmin)); err != nil {
		return fmt.Errorf("failed to insert user session: %v", err)
	}
	return nil
}

func Test_AdminGRPC(t *testing.T) {
	// add metadata for authorization
	header := make(http.Header)
	id, _ := util.StringFromPGUUID(testSesh.ID)
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
