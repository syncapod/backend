package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sschwartz96/syncapod-backend/internal/auth"
	"github.com/sschwartz96/syncapod-backend/internal/config"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	"github.com/sschwartz96/syncapod-backend/internal/twirp"

	"github.com/sschwartz96/syncapod-backend/internal/handler"
	"github.com/sschwartz96/syncapod-backend/internal/podcast"
)

func main() {
	log.Println("Running syncapod")

	// read config
	cfg, err := readConfig("config.json")
	if err != nil {
		log.Fatal("Main() error, could not read config: ", err)
	}

	// setup context
	ctx, cncFn := context.WithTimeout(context.Background(), time.Second*5)
	defer cncFn()

	// connect to db
	log.Println("connecting to db")

	pgURI := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DbUser, url.QueryEscape(cfg.DbPass), cfg.DbHost, cfg.DbPort, cfg.DbName)
	log.Println("pgURI:", pgURI)
	pgdb, err := pgxpool.Connect(ctx, pgURI)
	if err != nil {
		log.Fatalf("couldn't connect to db: %v", err)
	}

	// run migrations
	mig, err := migrate.New("file://"+cfg.MigrationsDir, pgURI)
	if err != nil {
		log.Fatalf("couldn't create migrate struct : %v", err)
	}
	err = mig.Up()
	if err != nil && err.Error() != "no change" {
		log.Fatalf("couldn't run migrations: %v", err)
	}

	// setup stores
	authStore := db.NewAuthStorePG(pgdb)
	oauthStore := db.NewOAuthStorePG(pgdb)
	podStore := db.NewPodcastStore(pgdb)

	// setup controllers
	authController := auth.NewAuthController(authStore, oauthStore)
	podController, err := podcast.NewPodController(podStore)
	if err != nil {
		log.Fatalf("main() error setting up pod controller: %v", err)
	}
	rssController := podcast.NewRSSController(podController)

	// setup twirp services
	gAuthService := twirp.NewAuthService(authController)
	gPodService := twirp.NewPodcastService(podController)
	gAdminService := twirp.NewAdminService(podController, rssController)

	// setup & start twirp server
	twirpServer := twirp.NewServer(
		authController,
		gAuthService,
		gPodService,
		gAdminService,
	)

	if err != nil {
		log.Fatalf("failed to create grpc auth handler endpoint\n%v\n", err)
	}

	go func() {
		// start server
		err = twirpServer.Start()
		if err != nil {
			log.Fatalf("main.twirp error starting server: %v", err)
		}
	}()

	// start updating podcasts
	go updatePodcasts(rssController)

	log.Println("setting up handlers")

	// setup handler
	handler, err := handler.CreateHandler(cfg, authController, podController)
	if err != nil {
		log.Fatal("could not setup handlers: ", err)
	}

	// debug TODO: remove
	if cfg.Debug || true {
		_, err := authController.CreateUser(context.Background(), "testUser@syncapod.com", "testUser", "testUser123!@#", time.Now())
		if err != nil {
			log.Printf("failed to create test user: %v\n", err)
		}
		r, err := podcast.DownloadRSS("https://feeds.twit.tv/twit.xml")
		if err != nil {
			log.Printf("failed to download debug podcast: %v\n", err)
		}
		podID, err := rssController.AddNewPodcast("https://feeds.twit.tv/twit.xml", r)
		if err != nil {
			log.Printf("failed to add debug podcast: %v\n", err)
		}
		log.Println("podID:", podID)
	}

	// start server
	log.Println("starting server")
	err = startServer(cfg, handler)
	if err != nil {
		log.Fatalf("server error: %v", err)
	}

}

func updatePodcasts(rssController *podcast.RSSController) {
	for {
		err := rssController.UpdatePodcasts()
		if err != nil {
			log.Println("main/updatePodcasts() error:", err)
		}
		time.Sleep(time.Minute * 15)
	}
}

func readConfig(path string) (*config.Config, error) {
	cfgFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("readConfig() error opening file: %v", err)
	}
	return config.ReadConfig(cfgFile)
}

func startServer(cfg *config.Config, h *handler.Handler) error {
	return http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), h)
}
