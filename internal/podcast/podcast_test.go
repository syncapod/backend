package podcast

import (
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sschwartz96/syncapod-backend/internal"
)

var (
	dbpg *pgxpool.Pool
)

func TestMain(m *testing.M) {
	// spin up docker container and return pgx pool
	var dockerCleanFunc func() error
	var err error
	dbpg, dockerCleanFunc, err = internal.StartDockerDB("db_auth")
	if err != nil {
		log.Fatalf("auth.TestMain() error setting up docker db: %v", err)
	}

	// setup db
	setupPodcastDB()

	// run tests
	runCode := m.Run()

	// close pgx pool
	dbpg.Close()

	// cleanup docker container
	err = dockerCleanFunc()
	if err != nil {
		log.Fatalf("podcast.TestMain() error cleaning up docker container: %v", err)
	}

	os.Exit(runCode)
}

func setupPodcastDB() {
}
