package handler

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog/v2"
	"github.com/sschwartz96/syncapod-backend/internal/auth"
	"github.com/sschwartz96/syncapod-backend/internal/config"
	"github.com/sschwartz96/syncapod-backend/internal/podcast"
	"github.com/sschwartz96/syncapod-backend/internal/util"
)

// Handler is the main handler for syncapod, all routes go through it
type Handler struct {
	router *chi.Mux
	log    *slog.Logger
}

// CreateHandler sets up the main handler
func CreateHandler(cfg *config.Config, authC *auth.AuthController, podCon *podcast.PodController, log *slog.Logger) (*Handler, error) {

	// create chi routers (muxes)
	router := chi.NewRouter()         // created all domains
	syncapodRouter := chi.NewRouter() // created for the syncapod.com domain
	mtaSTSRouter := chi.NewRouter()   // created for the mta-sts.syncapod.com domain

	// create main handle
	handler := &Handler{
		router: router,
		log:    log,
	}

	// create oauth handler
	oauthHandler, err := CreateOauthHandler(
		cfg,
		authC,
		map[string]string{
			cfg.AlexaClientID:   cfg.AlexaSecret,
			cfg.ActionsClientID: cfg.ActionsSecret,
		},
		log,
	)
	if err != nil {
		return nil, fmt.Errorf("CreateHandler() error creating oauthHandler: %v", err)
	}

	// create alexa handler
	alexaHandler := CreateAlexaHandler(authC, podCon, log)

	// middleware
	httpLogger := httplog.NewLogger("syncapod")
	httpLogger.Logger = log
	router.Use(httplog.RequestLogger(httpLogger))

	// this handles routing to various hostnames
	hostRouter := NewHostRouter(syncapodRouter)
	hostRouter.SetHostRoute(fmt.Sprintf("mta-sts.%s", cfg.Host), mtaSTSRouter)

	router.Mount("/", hostRouter.Handler())

	// syncapod routes
	syncapodRouter.Get("/", func(res http.ResponseWriter, req *http.Request) {
		log.Info("req.Host", slog.Any("req.Host", req.Host))
		log.Info("req.Header.Get(\"Host\")", slog.Any("req.Header.Get(\"Host\")", req.Header.Get("Host")))
		res.Write([]byte("hello world"))
	})
	syncapodRouter.Mount("/oauth", oauthHandler.Routes())
	syncapodRouter.Mount("/api/alexa", alexaHandler.Routes())
	syncapodRouter.Mount("/api/actions", http.HandlerFunc(handler.actionsDebugHandler))

	// mta.sts domain routes
	mtaSTSRouter.Get("/.well-known/mta-sts.txt", mtaStsTxt)

	// print the routes

	return handler, nil
}

func (h *Handler) GetHandler() http.Handler {
	return h.router
}

func mtaStsTxt(res http.ResponseWriter, req *http.Request) {
	// MTA-STS doc: https://maddy.email/tutorials/setting-up/
	responseBody := `version: STSv1
mode: enforce
max_age: 604800
mx: mail.syncapod.com`

	res.Write([]byte(responseBody))
}

func (h *Handler) actionsDebugHandler(res http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		h.log.Debug("actions request, could not read request body", util.Err(err))
		return
	}

	h.log.Info("actions request", slog.String("body", string(body)))
}
