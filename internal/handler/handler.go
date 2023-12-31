package handler

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog/v2"
	"github.com/sschwartz96/syncapod-backend/internal/auth"
	"github.com/sschwartz96/syncapod-backend/internal/config"
	"github.com/sschwartz96/syncapod-backend/internal/podcast"
	"github.com/sschwartz96/syncapod-backend/internal/util"
)

// Handler is the main handler for syncapod, all routes go through it
type Handler struct {
	router       *chi.Mux
	oauthHandler *OauthHandler
	alexaHandler *AlexaHandler
	log          *slog.Logger
}

type RequestLoggerAdapter struct {
	logger *slog.Logger
}

func (l *RequestLoggerAdapter) Print(args ...any) {
	if len(args) > 0 {
		arg0, ok := args[0].(string)
		if ok {
			l.logger.Info(arg0, args[1:]...)
		} else {
			l.logger.Info("", args...)
		}
	}
}

// CreateHandler sets up the main handler
func CreateHandler(cfg *config.Config, authC *auth.AuthController, podCon *podcast.PodController, log *slog.Logger) (*Handler, error) {
	router := chi.NewRouter()
	syncapodRouter := chi.NewRouter()
	mtaSTSRouter := chi.NewRouter()

	oauthHandler, err := CreateOauthHandler(
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
	alexaHandler := CreateAlexaHandler(authC, podCon, log)

	httpLogger := httplog.NewLogger("syncapod")
	httpLogger.Logger = log
	router.Use(httplog.RequestLogger(httpLogger))

	// this handles routing to various hostnames
	hostRouter := NewHostRouter(syncapodRouter)
	// hostRouter.SetHostRoute(cfg.Host, syncapodRouter)
	hostRouter.SetHostRoute(fmt.Sprintf("mta-sts.%s", cfg.Host), mtaSTSRouter)

	router.Mount("/", hostRouter.Handler())

	log.Info(fmt.Sprintf("mta-sts.%s", cfg.Host))

	syncapodRouter.Get("/", func(res http.ResponseWriter, req *http.Request) {
		log.Info("req.Host", slog.Any("req.Host", req.Host))
		log.Info("req.Header.Get(\"Host\")", slog.Any("req.Header.Get(\"Host\")", req.Header.Get("Host")))
		res.Write([]byte("hello world"))
	})

	mtaSTSRouter.Get("/.well-known/mta-sts.txt", mtaSTSTXT)

	return &Handler{
		router:       router,
		oauthHandler: oauthHandler,
		alexaHandler: alexaHandler,
		log:          log,
	}, nil
}

func (h *Handler) GetHandler() http.Handler {
	return h.router
}

func mtaSTSTXT(res http.ResponseWriter, req *http.Request) {
	responseBody := `version: STSv1
mode: enforce
max_age: 604800
mx: mail.syncapod.com`

	res.Write([]byte(responseBody))
}

// ServeHTTP handles all requests
func (h *Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// first check for mta-sts subdomain
	// MTA-STS doc: https://maddy.email/tutorials/setting-up/
	host := strings.TrimSpace(strings.ToLower(req.Host))
	if strings.HasPrefix(host, "mta-sts") {
		if strings.HasSuffix(req.URL.Path, "/.well-known/mta-sts.txt") {

			return
		}
		res.Write([]byte("404 Page not Found"))
		return
	}

	// normal routing
	var head string
	head = ""

	switch head {
	case "oauth":
		h.oauthHandler.ServeHTTP(res, req)
	case "api":
		h.serveAPI(res, req)
	}
}

func (h *Handler) serveAPI(res http.ResponseWriter, req *http.Request) {
	var head string
	// head, req.URL.Path = ShiftPath(req.URL.Path)
	head = ""

	switch head {
	case "alexa":
		h.alexaHandler.Alexa(res, req)
	case "actions":
		body, err := io.ReadAll(req.Body)
		if err != nil {
			h.log.Debug("actions request, could not read request body", util.Err(err))
			return
		}

		h.log.Info("actions request", slog.String("body", string(body)))
	}
}
