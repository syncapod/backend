package twirp

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sschwartz96/syncapod-backend/internal"
	"github.com/sschwartz96/syncapod-backend/internal/auth"
	"github.com/sschwartz96/syncapod-backend/internal/db_new"
	protos "github.com/sschwartz96/syncapod-backend/internal/gen"
	"github.com/sschwartz96/syncapod-backend/internal/podcast"
	"github.com/sschwartz96/syncapod-backend/internal/util"
	"github.com/stretchr/testify/require"
	"github.com/twitchtv/twirp"
)

var (
	dbpg       *pgxpool.Pool
	testUserID pgtype.UUID
	testUser   = db_new.InsertUserParams{
		Email: "user@twirp.test", Username: "user_twirp_test",
		Birthdate:    util.PGDateFromTime(time.Unix(0, 0).UTC()),
		PasswordHash: []byte("$2y$12$ndywn/c6wcB0oPv1ZRMLgeSQjTpXzOUCQy.5vdYvJxO9CS644i6Ce"),
		Created:      util.PGFromTime(time.Unix(0, 0)),
		LastSeen:     util.PGFromTime(time.Unix(0, 0)),
	}
)

func TestMain(m *testing.M) {
	var dockerCleanFunc func() error
	var err error
	dbpg, dockerCleanFunc, err = internal.StartDockerDB("db_auth")
	if err != nil {
		log.Fatalf("auth.TestMain() error setting up docker db: %v", err)
	}

	// setup db
	err = setupAuthDB()
	if err != nil {
		log.Fatalf("twirp.TestMain() error setting up auth database: %v", err)
	}
	err = setupPodDB()
	if err != nil {
		log.Fatalf("twirp.TestMain() error setting up podcast database: %v", err)
	}
	err = setupAdmin()
	if err != nil {
		log.Fatalf("twirp.TestMain() error setting up db for admin: %v", err)
	}

	authController := auth.NewAuthController(db_new.New(dbpg), slog.Default())
	podController, err := podcast.NewPodController(db_new.New(dbpg))
	if err != nil {
		log.Fatalf("twirp.TestMain() error setting up PodController: %v", err)
	}
	rssController := podcast.NewRSSController(podController, slog.Default())

	twirpServer := NewServer(authController,
		NewAuthService(authController), NewPodcastService(podController),
		NewAdminService(podController, rssController),
	)

	go func() {
		err := twirpServer.Start()
		if err != nil {
			log.Fatalf("Twirp server failed to start: %v", err)
		}
	}()

	// run tests
	runCode := m.Run()

	// close pgx pool
	dbpg.Close()

	// cleanup docker container
	err = dockerCleanFunc()
	if err != nil {
		log.Fatalf("twirp.TestMain() error cleaning up docker container: %v", err)
	}

	os.Exit(runCode)
}

func setupAuthDB() error {
	queries := db_new.New(dbpg)
	user, err := queries.InsertUser(context.Background(), testUser)
	if err != nil {
		return fmt.Errorf("failed to insert user: %v", err)
	}
	testUserID = user.ID
	return nil
}

func TestAuthGRPC(t *testing.T) {
	// setup auth client
	client := protos.NewAuthProtobufClient(
		"http://localhost:8081",
		http.DefaultClient,
		twirp.WithClientPathPrefix("/rpc/auth"),
	)

	autheRes, err := client.Authenticate(context.Background(),
		&protos.AuthenticateReq{Username: testUser.Username, Password: "password"},
	)
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}
	require.NotEmpty(t, autheRes.SessionKey)
	seshKey := autheRes.SessionKey
	log.Println("got session key:", seshKey)

	// Authorization
	// authoRes, err := client.Authorize(context.Background(),
	// 	&protos.AuthorizeReq{SessionKey: seshKey},
	// )
	// if err != nil {
	// 	t.Fatalf("Authorize failed: %v", err)
	// }
	// require.NotEmpty(t, authoRes.User)
	// log.Println("authorized user:", authoRes.User)

	header := make(http.Header)
	header.Add(authTokenKey, seshKey)
	ctx, err := twirp.WithHTTPRequestHeaders(context.Background(), header)
	if err != nil {
		t.Fatalf("Failed to add header to context: %v", err)
	}

	// Logout
	logoutRes, err := client.Logout(ctx, &protos.LogoutReq{SessionKey: seshKey})
	if err != nil {
		t.Fatalf("Logout failed: %v", err)
	}
	require.Equal(t, true, logoutRes.Success)
}
