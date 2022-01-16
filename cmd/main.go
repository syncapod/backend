package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/julienschmidt/httprouter"
	"github.com/sschwartz96/syncapod-backend/internal/auth"
	"github.com/sschwartz96/syncapod-backend/internal/config"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	"github.com/sschwartz96/syncapod-backend/internal/twirp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/sschwartz96/syncapod-backend/internal/handler"
	"github.com/sschwartz96/syncapod-backend/internal/mail"
	"github.com/sschwartz96/syncapod-backend/internal/podcast"
	"golang.org/x/crypto/acme/autocert"
)

func main() {
	// read config
	cfg, err := readConfig("config.json")
	if err != nil {
		log.Fatalln("could not read config:", err)
	}

	// create logger
	var logger *zap.Logger
	if cfg.Debug {
		zCfg := zap.NewDevelopmentConfig()
		zCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		logger, err = zCfg.Build()
		// logger, err = zap.NewDevelopment(zCfg)
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		log.Fatalln("could not initiate logger:", err)
	}

	// manage certificate
	certMan := createCertManager(cfg)

	// setup context
	ctx, cncFn := context.WithTimeout(context.Background(), time.Second*5)
	defer cncFn()

	// connect to db
	logger.Info("connecting to db")

	pgURI := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DbUser, url.QueryEscape(cfg.DbPass), cfg.DbHost, cfg.DbPort, cfg.DbName)

	pgdb, err := pgxpool.Connect(ctx, pgURI)
	if err != nil {
		logger.Fatal("could not connect to db", zap.Error(err))
	}

	// run migrations
	mig, err := migrate.New("file://"+cfg.MigrationsDir, pgURI)
	if err != nil {
		logger.Fatal("error creating migrations", zap.Error(err))
	}
	err = mig.Up()
	if err != nil && err.Error() != "no change" {
		logger.Fatal("error running startup db migration", zap.Error(err))
	}

	// setup mail client
	var mailer mail.MailQueuer
	if cfg.Production {
		mailer, err := mail.NewMailer(cfg, logger)
		if err != nil {
			logger.Fatal("could not setup mail client", zap.Error(err))
		}
		go func() {
			logger.Info("starting mail consumer")
			err := mailer.Start()
			if err != nil {
				logger.Error("mail consumer failed to start up", zap.Error(err))
			}
		}()
	} else {
		mailer = &developmentMailer{logger: logger}
	}
	mailer.Queue("sam.schwartz96@gmail.com", "Syncapod Starting Up", "This message is to inform that the Syncapod application has started up at"+time.Now().String())

	// setup stores
	authStore := db.NewAuthStorePG(pgdb)
	oauthStore := db.NewOAuthStorePG(pgdb)
	podStore := db.NewPodcastStore(pgdb)

	// setup controllers
	authController := auth.NewAuthController(authStore, oauthStore, mailer)
	podController, err := podcast.NewPodController(podStore)
	if err != nil {
		logger.Fatal("error setting up pod controller", zap.Error(err))
	}
	rssController := podcast.NewRSSController(podController)

	// setup twirp services
	tAuthService := twirp.NewAuthService(authController)
	tPodService := twirp.NewPodcastService(podController)
	tAdminService := twirp.NewAdminService(podController, rssController)

	// setup & start gRPC server
	twirpServer := twirp.NewServer(
		authController,
		tAuthService,
		tPodService,
		tAdminService,
	)

	// 	go func() {
	// 		// start server
	// 		err = twirpServer.Start()
	// 		if err != nil {
	// 			logger.Fatal("error starting up twirp server", zap.Error(err))
	// 		}
	// 	}()

	// start updating podcasts
	go updatePodcasts(logger, rssController)

	logger.Info("setting up handlers")

	// setup defaultHandler
	defaultHandler, err := handler.CreateHandler(cfg, authController, podController)
	if err != nil {
		logger.Fatal("could not setup handlers", zap.Error(err))
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
		} else {
			log.Println("podID:", podID)
		}
	}

	// start server
	newRouter := registerRoutes(cfg, defaultHandler, twirpServer)
	logger.Info("starting server")
	err = startServer(cfg, logger, certMan, newRouter)
	if err != nil {
		logger.Fatal("startServer error", zap.Error(err))
	}

}

func createCertManager(cfg *config.Config) *autocert.Manager {
	if cfg.Production {
		return &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			Cache:      autocert.DirCache(cfg.CertDir),
			HostPolicy: autocert.HostWhitelist("syncapod.com", "mail.syncapod.com", "www.syncapod.com", "45.79.25.193", "mta-sts.syncapod.com"),
			Email:      "sam.schwartz96@gmail.com",
		}
	}
	return nil
}

func updatePodcasts(logger *zap.Logger, rssController *podcast.RSSController) {
	for {
		err := rssController.UpdatePodcasts()
		if err != nil {
			logger.Error("main/updatePodcasts() error:", zap.Error(err))
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

func toHttpRouterHandle(handlerFunc http.HandlerFunc) httprouter.Handle {
	return func(res http.ResponseWriter, req *http.Request, p httprouter.Params) {
		log.Println(p)
		handlerFunc(res, req)
	}
}

func registerRoutes(cfg *config.Config, h *handler.Handler, t *twirp.Server) *httprouter.Router {
	router := httprouter.New()

	// svelte website
	svelteServerURL, _ := url.Parse("http://localhost:3000")
	router.GET("/admin", toHttpRouterHandle(httputil.NewSingleHostReverseProxy(svelteServerURL).ServeHTTP))
	router.GET("/admin/*all", toHttpRouterHandle(httputil.NewSingleHostReverseProxy(svelteServerURL).ServeHTTP))
	if cfg.Production {
		router.GET("/_app/*all_match", toHttpRouterHandle(httputil.NewSingleHostReverseProxy(svelteServerURL).ServeHTTP))
	} else {
		router.GET("/", toHttpRouterHandle(httputil.NewSingleHostReverseProxy(svelteServerURL).ServeHTTP))
		router.GET("/.svelte-kit/*all_match", toHttpRouterHandle(httputil.NewSingleHostReverseProxy(svelteServerURL).ServeHTTP))
		router.GET("/.vite/*all_match", toHttpRouterHandle(httputil.NewSingleHostReverseProxy(svelteServerURL).ServeHTTP))
		router.GET("/@vite/*all_match", toHttpRouterHandle(httputil.NewSingleHostReverseProxy(svelteServerURL).ServeHTTP))
		router.GET("/node_modules/*all_match", toHttpRouterHandle(httputil.NewSingleHostReverseProxy(svelteServerURL).ServeHTTP))
		router.GET("/src/*all_match", toHttpRouterHandle(httputil.NewSingleHostReverseProxy(svelteServerURL).ServeHTTP))
		router.GET("/@fs/*all_match", toHttpRouterHandle(httputil.NewSingleHostReverseProxy(svelteServerURL).ServeHTTP))
		// for pnpm run preview
		router.GET("/_app/*all_match", toHttpRouterHandle(httputil.NewSingleHostReverseProxy(svelteServerURL).ServeHTTP))
	}
	// oauth
	router.GET("/oauth/login", toHttpRouterHandle(h.OAuthHandler.LoginGet))
	router.POST("/oauth/login", toHttpRouterHandle(h.OAuthHandler.LoginPost))
	router.GET("/oauth/authorize", toHttpRouterHandle(h.OAuthHandler.AuthorizeGet))
	router.POST("/oauth/authorize", toHttpRouterHandle(h.OAuthHandler.AuthorizePost))
	router.POST("/oauth/token", toHttpRouterHandle(h.OAuthHandler.Token))

	// TODO: refresh Alexa commands
	router.GET("/api/alex", toHttpRouterHandle(h.AlexaHandler.Alexa))
	router.POST("/api/alex", toHttpRouterHandle(h.AlexaHandler.Alexa))

	// mta-sts.syncapod.com validation
	router.GET("/.well-known/mta-sts.txt", toHttpRouterHandle(h.MtaSts))

	// twirp - /rpc/*
	router = t.RegisterRouter(router)
	return router
}

func startServer(cfg *config.Config, logger *zap.Logger, a *autocert.Manager, router *httprouter.Router) error {

	// check if we are production
	if cfg.Production {
		// run http server to redirect traffic and handle cert renewal
		go func() {
			err := http.ListenAndServe(":http", a.HTTPHandler(nil))
			logger.Fatal("error starting up http redirect listener", zap.Error(err))
		}()
		// create server
		s := &http.Server{
			Addr:         ":https",
			TLSConfig:    a.TLSConfig(),
			Handler:      router,
			ReadTimeout:  time.Second * 15,
			WriteTimeout: time.Second * 15,
			ErrorLog:     log.New(&httpErrorToZap{logger}, "", 0), // TODO: create struct that contains Write([]byte) function to "convert" to zap.logger
		}
		return s.ListenAndServeTLS("", "")
	} else {
		// just standup a default server
		return http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), router)
	}
}

type httpErrorToZap struct {
	logger *zap.Logger
}

func (t *httpErrorToZap) Write(p []byte) (int, error) {
	// t.logger.Error("http server error", zap.Error(errors.New(string(p))))
	t.logger.Info("http server error", zap.String("error", string(p)))
	return len(p), nil
}

type developmentMailer struct {
	logger *zap.Logger
}

func (d *developmentMailer) Queue(to, subject, body string) {
	d.logger.Debug("developmentMailer.Queue()", zap.String("to", to), zap.String("subject", subject), zap.String("body", body))
}
