package podcast

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	testDB *pgxpool.Pool
)

func TestMain(m *testing.M) {
	// connect stop after 5 seconds
	start := time.Now()
	fiveSec := time.Second * 5
	err := errors.New("start loop")
	for err != nil {
		if time.Since(start) > fiveSec {
			log.Fatal(`Could not connect to postgres\n
				Took longer than 5 seconds, maybe download postgres image`)
		}
		testDB, err = pgxpool.Connect(context.Background(),
			fmt.Sprintf(
				"postgres://postgres:secret@localhost:5432/postgres?sslmode=disable",
			),
		)
		time.Sleep(time.Millisecond * 250)
	}

	// setup db
	setupPodcastDB()

	// run tests
	runCode := m.Run()

	testDB.Close()

	os.Exit(runCode)
}

func setupPodcastDB() {

}
