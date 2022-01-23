// Package TestMain() located in auth_test.go
package twirp

import (
	"context"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	protos "github.com/sschwartz96/syncapod-backend/internal/gen"
	"github.com/stretchr/testify/require"
	"github.com/twitchtv/twirp"
	"golang.org/x/crypto/bcrypt"
)

var (
	goTimeRSSURL = "https://changelog.com/gotime/feed"
	testAdminPwd = "AdminPasswordEasy"

	testAdminUser = &db.UserRow{
		ID:           uuid.New(),
		Email:        "admin@twirp.test",
		Username:     "admin_twirp_test",
		Birthdate:    time.Unix(0, 0).UTC(),
		PasswordHash: genPassOrFail(testAdminPwd),
		Created:      time.Unix(0, 0),
		LastSeen:     time.Unix(0, 0),
		Activated:    false,
		IsAdmin:      true,
	}
)

func genPassOrFail(pwd string) []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte(testAdminPwd), bcrypt.MinCost)
	if err != nil {
		log.Fatalln("could not generate hash from password:", pwd)
	}
	return hash
}

func setupAdmin() error {
	// var err error

	// insert user session to mimic user already authenticated
	// authStore := db.NewAuthStorePG(dbpg)
	// if err = authStore.InsertSession(context.Background(), testAdminSesh); err != nil {
	// 	return fmt.Errorf("failed to insert user session: %v", err)
	// }
	insertUser(testAdminUser)
	insertSession(testAdminSesh)
	return nil
}

func Test_AdminTwirp(t *testing.T) {
	// add metadata for authorization
	header := make(http.Header)
	header.Set(authTokenKey, testAdminSesh.ID.String())

	ctx, err := twirp.WithHTTPRequestHeaders(context.Background(), header)
	if err != nil {
		t.Fatalf("Twirp could not add add headers: %v", err)
	}

	client := protos.NewAdminJSONClient("http://localhost:8081", http.DefaultClient, twirp.WithClientPathPrefix(prefix))

	// AddPodcast
	addPodRes, err := client.AddPodcast(ctx, &protos.AddPodReq{Url: goTimeRSSURL})
	require.Nilf(t, err, "message: %w", err)
	require.Equal(t, "Go Time: Golang, Software Engineering", addPodRes.Podcast.Title)

	// RefreshPodcast
	_, err = client.RefreshPodcast(ctx, &protos.RefPodReq{})
	require.Nil(t, err, "error RefreshPodcast()")

	// // SearchPodscast
	// pods, err := client.SearchPodcasts(ctx, &protos.SearchPodReq{Text: "go time"})
	// require.Nil(t, err)
	// require.Equal(t, 1, len(pods.Podcasts))
}
