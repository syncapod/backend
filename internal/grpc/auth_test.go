package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sschwartz96/syncapod-backend/internal"
	"github.com/sschwartz96/syncapod-backend/internal/auth"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	"github.com/sschwartz96/syncapod-backend/internal/podcast"
	"github.com/sschwartz96/syncapod-backend/internal/protos"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var (
	lis      *bufconn.Listener
	dbpg     *pgxpool.Pool
	testUser = &db.UserRow{
		ID:    uuid.MustParse("b921c6e3-9cd0-4aed-9c4e-1d88ae20c777"),
		Email: "user@grpc.test", Username: "user_grpc_test",
		Birthdate:    time.Unix(0, 0).UTC(),
		PasswordHash: []byte("$2y$12$ndywn/c6wcB0oPv1ZRMLgeSQjTpXzOUCQy.5vdYvJxO9CS644i6Ce"),
		Created:      time.Unix(0, 0), LastSeen: time.Unix(0, 0),
	}
)

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

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
		log.Fatalf("grpc.TestMain() error setting up auth database: %v", err)
	}
	err = setupPodDB()
	if err != nil {
		log.Fatalf("grpc.TestMain() error setting up podcast database: %v", err)
	}
	err = setupAdmin()
	if err != nil {
		log.Fatalf("grpc.TestMain() error setting up db for admin: %v", err)
	}

	// setup grpc server
	lis = bufconn.Listen(bufSize)
	podCon, err := podcast.NewPodController(db.NewPodcastStore(dbpg))
	authCon := auth.NewAuthController(db.NewAuthStorePG(dbpg), db.NewOAuthStorePG(dbpg))
	rssCon := podcast.NewRSSController(podCon)
	if err != nil {
		log.Fatalf("grpc.TestMain() error setting up pod controller: %v", err)
	}
	grpcServer := NewServer(nil,
		authCon,
		NewAuthService(authCon),
		NewPodcastService(podCon),
		NewAdminService(podCon, rssCon),
	)
	//s := grpc.NewServer()
	go func() {
		if err := grpcServer.Start(lis); err != nil {
			log.Fatalf("gRPC test server exited with error: %v", err)
		}
	}()

	// run tests
	runCode := m.Run()

	// close pgx pool
	dbpg.Close()

	// cleanup docker container
	err = dockerCleanFunc()
	if err != nil {
		log.Fatalf("grpc.TestMain() error cleaning up docker container: %v", err)
	}

	os.Exit(runCode)
}

func setupAuthDB() error {
	authStore := db.NewAuthStorePG(dbpg)
	err := authStore.InsertUser(context.Background(), testUser)
	if err != nil {
		return fmt.Errorf("failed to insert user: %v", err)
	}
	return nil
}

func TestAuthGRPC(t *testing.T) {
	// setup auth client
	conn, err := grpc.DialContext(
		context.Background(), "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("failed to dial grpc bufnet: %v", err)
	}
	defer conn.Close()
	client := protos.NewAuthClient(conn)

	// Authenticate
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
	authoRes, err := client.Authorize(context.Background(),
		&protos.AuthorizeReq{SessionKey: seshKey},
	)
	if err != nil {
		t.Fatalf("Authorize failed: %v", err)
	}
	require.NotEmpty(t, authoRes.User)
	log.Println("authorized user:", authoRes.User)

	// Logout
	logoutRes, err := client.Logout(context.Background(),
		&protos.LogoutReq{SessionKey: seshKey},
	)
	if err != nil {
		t.Fatalf("Logout failed: %v", err)
	}
	require.Equal(t, true, logoutRes.Success)
}
