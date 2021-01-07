package internal

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/ory/dockertest/v3"
)

// StartDockerDB is a helper method to automatically start a postgres docker instance via dockertest.
// returns a pgxpool, a docker purge func, and/or error
func StartDockerDB(name string) (*pgxpool.Pool, func() error, error) {
	var pgxPool *pgxpool.Pool
	var pgURI string

	// create dockertest pool
	dockerPool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, fmt.Errorf("StartDockerDB() error creating new pool: %v", err)
	}

	// start the postgres instance
	resource, err := dockerPool.Run("postgres", "latest", []string{"POSTGRES_PASSWORD=secret"})
	if err != nil {
		return nil, nil, fmt.Errorf("StartDockerDB() error creating resource: %v", err)
	}

	// 10 second wait time to connect to the db
	dockerPool.MaxWait = time.Second * 10
	err = dockerPool.Retry(func() error {
		pgURI = fmt.Sprintf(
			"postgres://postgres:secret@localhost:%s/postgres?sslmode=disable",
			resource.GetPort("5432/tcp"),
		)
		pgxPool, err = pgxpool.Connect(context.Background(), pgURI)
		if err != nil {
			return err
		}
		conn, err := pgxPool.Acquire(context.Background())
		if err != nil {
			return err
		}
		defer conn.Release()
		return conn.Conn().Ping(context.Background())
	})
	if err != nil {
		return nil, nil, fmt.Errorf("StartDockerDB() error connecting: %v", err)
	}
	// run migrations
	mig, err := migrate.New("file://../../migrations", pgURI)
	if err != nil {
		log.Fatalf("couldn't create migrate struct : %v", err)
	}
	err = mig.Up()
	if err != nil && err.Error() != "no change" {
		log.Fatalf("couldn't run migrations: %v", err)
	}
	return pgxPool, func() error { return dockerPool.Purge(resource) }, nil
}
