package handler

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/sschwartz96/syncapod-backend/internal/auth"
	"github.com/sschwartz96/syncapod-backend/internal/config"
	"github.com/sschwartz96/syncapod-backend/internal/podcast"
)

// Handler is the main handler for syncapod, all routes go through it
type Handler struct {
	OAuthHandler *OauthHandler
	AlexaHandler *AlexaHandler
}

// CreateHandler sets up the main handler
func CreateHandler(cfg *config.Config, authC auth.Auth, podCon *podcast.PodController) (*Handler, error) {
	oauthHandler, err := CreateOauthHandler(
		authC,
		map[string]string{
			cfg.AlexaClientID:   cfg.AlexaSecret,
			cfg.ActionsClientID: cfg.ActionsSecret,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("CreateHandler() error creating oauthHandler: %v", err)
	}
	alexaHandler := CreateAlexaHandler(authC, podCon)
	return &Handler{OAuthHandler: oauthHandler, AlexaHandler: alexaHandler}, nil
}

// ServeHTTP handles all requests
func (h *Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	// normal routing
	var head string
	head, req.URL.Path = ShiftPath(req.URL.Path)

	switch head {
	case "oauth":
		h.OAuthHandler.ServeHTTP(res, req)
	case "api":
		h.serveAPI(res, req)
	}
}

func (h *Handler) MtaSts(res http.ResponseWriter, req *http.Request) {
	// MTA-STS doc: https://maddy.email/tutorials/setting-up/
	host := strings.TrimSpace(strings.ToLower(req.Host))
	// check for mta-sts subdomain
	if strings.HasPrefix(host, "mta-sts") {
		res.Write([]byte("version: STSv1\n" +
			"mode: enforce\n" +
			"max_age: 604800\n" +
			"mx: mail.syncapod.com\n"),
		)
		return
	}
	res.Write([]byte("404 Page not Found"))
}

func (h *Handler) serveAPI(res http.ResponseWriter, req *http.Request) {
	var head string
	head, req.URL.Path = ShiftPath(req.URL.Path)

	switch head {
	case "alexa":
		h.AlexaHandler.Alexa(res, req)
	case "actions":
		log.Println("actions req")
		log.Println(ioutil.ReadAll(req.Body))
	}
}

// ShiftPath splits off the first component of p, which will be cleaned of
// relative components before processing. head will never contain a slash and
// tail will always be a rooted path without trailing slash.
func ShiftPath(p string) (head, tail string) {
	p = path.Clean("/" + p)
	i := strings.Index(p[1:], "/") + 1
	if i <= 0 {
		return p[1:], "/"
	}
	return p[1:i], p[i:]
}
