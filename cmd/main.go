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
	"github.com/google/uuid"
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
	"golang.org/x/crypto/bcrypt"
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
	logger.Info("starting syncapod", zap.Bool("production", cfg.Production))

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
	logger.Info("successfully connected to db, running migrations")

	// run migrations
	mig, err := migrate.New("file://"+cfg.MigrationsDir, pgURI)
	if err != nil {
		logger.Fatal("error creating migrations", zap.Error(err))
	}
	err = mig.Up()
	if err != nil && err.Error() != "no change" {
		logger.Fatal("error running startup db migration", zap.Error(err))
	}
	if err.Error() == "no change" {
		logger.Info("no change in migrations")
	} else {
		logger.Info("successfully ran migrations")
	}

	// setup mail client
	logger.Info("setting up mailer")
	var mailQueuer mail.MailQueuer
	if cfg.Production {
		mailer, err := mail.NewMailer(cfg, logger)
		if err != nil {
			logger.Fatal("could not setup mail client", zap.Error(err))
		}
		mailQueuer = mailer
		go func() {
			logger.Info("starting mail consumer")
			err := mailer.Start()
			if err != nil {
				logger.Error("mail consumer failed to start up", zap.Error(err))
			}
		}()
	} else {
		mailQueuer = &developmentMailer{logger: logger}
	}
	logger.Info("successfully set up mailer")
	mailQueuer.Queue("sam.schwartz96@gmail.com", "Syncapod Starting Up", "This message is to inform that the Syncapod application has started up at"+time.Now().String())

	// setup stores
	authStore := db.NewAuthStorePG(pgdb)
	oauthStore := db.NewOAuthStorePG(pgdb)
	podStore := db.NewPodcastStore(pgdb)

	// setup controllers
	authController := auth.NewAuthController(authStore, oauthStore, mailQueuer)
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
		hash, _ := bcrypt.GenerateFromPassword([]byte("EasyPasswordRemeber"), bcrypt.MinCost)
		authStore.InsertUser(context.Background(), &db.UserRow{
			ID:           uuid.New(),
			Email:        "testAdmin@syncapod.com",
			Username:     "admin",
			Birthdate:    time.Now().AddDate(-18, 0, 0),
			PasswordHash: hash,
			Created:      time.Now(),
			LastSeen:     time.Now(),
			Activated:    true,
			IsAdmin:      true,
		})
		_, err := authController.CreateUser(context.Background(), "testUser@syncapod.com", "testUser", "EasyPasswordRemember", time.Now().AddDate(-18, 0, 0))
		if err != nil {
			log.Printf("failed to create test user: %v\n", err)
		}
		url, _ := url.Parse("https://feeds.twit.tv/twit.xml")
		pod, err := rssController.AddPodcast(context.Background(), url)
		if err != nil {
			log.Printf("failed to add debug podcast: %v\n", err)
		} else {
			log.Println("podID:", pod.ID)
		}
	}

	// start server
	newRouter := registerRoutes(cfg, defaultHandler, twirpServer)
	logger.Info("starting server")
	err = startServer(cfg, logger, newRouter)
	if err != nil {
		logger.Fatal("startServer error", zap.Error(err))
	}
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
		router.GET("/", toHttpRouterHandle(httputil.NewSingleHostReverseProxy(svelteServerURL).ServeHTTP))
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

func startServer(cfg *config.Config, logger *zap.Logger, router *httprouter.Router) error {
	// create server
	s := &http.Server{
		Handler:      router,
		Addr:         ":" + fmt.Sprintf("%d", cfg.Port),
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
		IdleTimeout:  time.Second * 5,
		ErrorLog:     log.New(&httpErrorToZap{logger}, "", 0), // TODO: create struct that contains Write([]byte) function to "convert" to zap.logger
	}
	return s.ListenAndServe()
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
