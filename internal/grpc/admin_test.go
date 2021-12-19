// Package TestMain() located in auth_test.go
package grpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	"github.com/sschwartz96/syncapod-backend/internal/protos"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	goTimeRSSURL = "https://changelog.com/gotime/feed"

	testSeshAdmin = &db.SessionRow{ID: uuid.New(), UserID: testUser.ID, LoginTime: time.Now(), LastSeenTime: time.Now(), Expires: time.Now().Add(time.Hour), UserAgent: "testUserAgent"}
)

func setupAdmin() error {
	var err error

	// insert user session to mimic user already authenticated
	authStore := db.NewAuthStorePG(dbpg)
	if err = authStore.InsertSession(context.Background(), testSeshAdmin); err != nil {
		return fmt.Errorf("failed to insert user session: %v", err)
	}
	return nil

}

func Test_AdminGRPC(t *testing.T) {
	// add metadata for authorization
	ctx := metadata.AppendToOutgoingContext(context.Background(), "token", testSesh.ID.String())

	// setup pod client
	conn, err := grpc.DialContext(
		ctx, "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("failed to dial grpc bufnet: %v", err)
	}
	defer conn.Close()
	client := protos.NewAdminClient(conn)

	// AddPodcast
	addPodRes, err := client.AddPodcast(ctx, &protos.AddPodReq{Url: goTimeRSSURL})
	require.Nil(t, err, "error AddPodcast()")
	require.Equal(t, "Go Time: Golang, Software Engineering", addPodRes.Podcast.Title)

	// RefreshPodcast
	_, err = client.RefreshPodcast(ctx, &protos.RefPodReq{})
	require.Nil(t, err, "error RefreshPodcast()")
}
