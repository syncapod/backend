package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sschwartz96/syncapod-backend/internal/auth"
	"github.com/sschwartz96/syncapod-backend/internal/config"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	sGRPC "github.com/sschwartz96/syncapod-backend/internal/grpc"
	"github.com/sschwartz96/syncapod-backend/internal/handler"
	"github.com/sschwartz96/syncapod-backend/internal/podcast"
	"golang.org/x/crypto/acme/autocert"
)

func main() {
	log.Println("Running syncapod")

	// read config
	cfg, err := readConfig("config.json")
	if err != nil {
		log.Fatal("Main() error, could not read config: ", err)
	}

	// manage certificate
	certMan := createCertManager(cfg)

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

	// setup & start gRPC server
	grpcServer := sGRPC.NewServer(certMan,
		authController,
		sGRPC.NewAuthService(authController),
		sGRPC.NewPodcastService(podController),
	)
	go func() {
		// setup listener
		grpcListener, err := net.Listen("tcp", ":"+strconv.Itoa(cfg.GRPCPort))
		if err != nil {
			log.Fatalf("main.grpc could not listen on port %d, err: %v", cfg.GRPCPort, err)
		}
		// start server
		err = grpcServer.Start(grpcListener)
		if err != nil {
			log.Fatalf("main.grpc error starting server: %v", err)
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

	// start server
	log.Println("starting server")
	err = startServer(cfg, certMan, handler)
	if err != nil {
		log.Fatalf("server error: %v", err)
	}

}

func createCertManager(cfg *config.Config) *autocert.Manager {
	if cfg.Production {
		return &autocert.Manager{
			Cache:      autocert.DirCache(cfg.CertDir),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist("syncapod.com", "www.syncapod.com"),
		}
	}
	return nil
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

func startServer(cfg *config.Config, a *autocert.Manager, h *handler.Handler) error {
	// check if we are production
	if cfg.Production {
		// run http server to redirect traffic and handle cert renewal
		go func() {
			log.Fatal(http.ListenAndServe(":http", a.HTTPHandler(nil)))
		}()
		// create server
		s := &http.Server{
			Addr:      ":https",
			TLSConfig: a.TLSConfig(),
			Handler:   h,
		}
		return s.ListenAndServeTLS("", "")
	} else {
		return http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), h)
	}
}
