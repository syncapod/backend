package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sschwartz96/syncapod-backend/internal/auth"
	"github.com/sschwartz96/syncapod-backend/internal/config"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	"github.com/sschwartz96/syncapod-backend/internal/db_new"
	"github.com/sschwartz96/syncapod-backend/internal/twirp"
	"github.com/sschwartz96/syncapod-backend/internal/util"

	"github.com/sschwartz96/syncapod-backend/internal/handler"
	"github.com/sschwartz96/syncapod-backend/internal/podcast"
)

func main() {
	slog.Info("Running syncapod")

	// TODO: change to using environment variables
	// read config
	cfg, err := readConfig("config.json")
	if err != nil {
		slog.Error("main() error, could not read config: ", util.Err(err))
		os.Exit(1)
	}

	// set up logger
	var logHanlder slog.Handler
	if cfg.Production {
		logHanlder = slog.NewJSONHandler(os.Stdout, nil)
	} else {
		logHanlder = slog.NewTextHandler(os.Stdout, nil)
	}
	log := slog.New(logHanlder)

	// setup context
	ctx, cncFn := context.WithTimeout(context.Background(), time.Second*5)
	defer cncFn()

	// connect to db
	log.Info("connecting to db")

	pgURI := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DbUser, url.QueryEscape(cfg.DbPass), cfg.DbHost, cfg.DbPort, cfg.DbName)
	logSafeURI := strings.ReplaceAll(pgURI, fmt.Sprintf(":%s@", cfg.DbPass), ":<redacted@")
	log.Debug("postgresql uri setup", slog.String("pgURI", logSafeURI))
	pgdb, err := pgxpool.New(ctx, pgURI)
	if err != nil {
		log.Error("could not create new db connection with pgxpool", slog.String("pgURI", logSafeURI), util.Err(err))
		os.Exit(2)
	}

	queries := db_new.New(pgdb)

	// run migrations
	mig, err := migrate.New("file://"+cfg.MigrationsDir, pgURI)
	if err != nil {
		log.Error("could not create migrate struct", util.Err(err))
		os.Exit(3)
	}
	err = mig.Up()
	if err != nil && err.Error() != "no change" {
		log.Error("could not run migrations", util.Err(err))
	}

	// setup stores
	oauthStore := db.NewOAuthStorePG(pgdb)
	podStore := db.NewPodcastStore(pgdb)

	// setup controllers
	authController := auth.NewAuthController(oauthStore, queries)
	podController, err := podcast.NewPodController(podStore)
	if err != nil {
		log.Error("error setting up pod controller", util.Err(err))
	}
	rssController := podcast.NewRSSController(podController, log)

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
		log.Error("failed to create grpc auth handler endpoint", util.Err(err))
		os.Exit(4)
	}

	go func() {
		// start server
		err = twirpServer.Start()
		// TODO: send this error through a channel and handle it on the main thread
		if err != nil {
			log.Error("error starting twirp server", util.Err(err))
			os.Exit(5)
		}
	}()

	// start updating podcasts
	go func() {
		for {
			err := rssController.UpdatePodcasts()
			if err != nil {
				log.Error("main/updatePodcasts() error:", util.Err(err))
			}
			time.Sleep(time.Minute * 15)
		}
	}()

	log.Info("setting up handlers")

	// setup handler
	handler, err := handler.CreateHandler(cfg, authController, podController, log)
	if err != nil {
		log.Error("could not setup handlers", util.Err(err))
		os.Exit(6)
	}

	// debug TODO: remove
	if cfg.Debug || true {
		_, err := authController.CreateUser(context.Background(), "testUser@syncapod.com", "testUser", "testUser123!@#", time.Now())
		if err != nil {
			log.Error("failed to create test user", util.Err(err))
		}
		r, err := podcast.DownloadRSS("https://feeds.twit.tv/twit.xml")
		if err != nil {
			log.Error("failed to download debug podcast", util.Err(err))
		}
		pod, err := rssController.AddNewPodcast("https://feeds.twit.tv/twit.xml", r)
		if err != nil {
			log.Error("failed to add debug podcast", util.Err(err))
		}
		log.Info("finished adding podcast", slog.String("podID", pod.ID.String()))
	}

	// start server
	log.Info("starting server")
	err = startServer(cfg, handler)
	if err != nil {
		log.Error("server error", util.Err(err))
		os.Exit(7)
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
