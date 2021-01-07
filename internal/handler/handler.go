package handler

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/sschwartz96/syncapod-backend/internal/auth"
	"github.com/sschwartz96/syncapod-backend/internal/config"
	"github.com/sschwartz96/syncapod-backend/internal/podcast"
)

// Handler is the main handler for syncapod, all routes go through it
type Handler struct {
	oauthHandler *OauthHandler
	alexaHandler *AlexaHandler
}

// CreateHandler sets up the main handler
func CreateHandler(config *config.Config, authC auth.Auth, podCon *podcast.PodController) (*Handler, error) {
	oauthHandler, err := CreateOauthHandler(authC, config.AlexaClientID, config.AlexaSecret)
	if err != nil {
		return nil, fmt.Errorf("CreateHandler() error creating oauthHandler: %v", err)
	}
	alexaHandler := CreateAlexaHandler(authC, podCon)
	return &Handler{oauthHandler: oauthHandler, alexaHandler: alexaHandler}, nil
}

// ServeHTTP handles all requests
func (h *Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var head string
	head, req.URL.Path = ShiftPath(req.URL.Path)

	switch head {
	case "oauth":
		h.oauthHandler.ServeHTTP(res, req)
	case "api": //TODO: update this to better reflect
		h.alexaHandler.Alexa(res, req)
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
