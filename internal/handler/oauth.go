package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sschwartz96/syncapod-backend/internal/auth"
	"github.com/sschwartz96/syncapod-backend/internal/config"
	"github.com/sschwartz96/syncapod-backend/internal/util"
)

// OauthHandler handles authorization and authentication to oauth clients
type OauthHandler struct {
	authController *auth.AuthController
	loginTemplate  *template.Template
	authTemplate   *template.Template
	clients        map[string]string
	log            *slog.Logger
}

// CreateOauthHandler just intantiates an OauthHandler
func CreateOauthHandler(cfg *config.Config, authController *auth.AuthController, clients map[string]string, log *slog.Logger) (*OauthHandler, error) {
	loginT, err := template.ParseFiles(cfg.TemplatesDir + "/oauth/login.gohtml")
	if err != nil {
		return nil, err
	}
	authT, err := template.ParseFiles(cfg.TemplatesDir + "/oauth/auth.gohtml")
	if err != nil {
		return nil, err
	}
	return &OauthHandler{
		authController: authController,
		loginTemplate:  loginT,
		authTemplate:   authT,
		clients:        clients,
		log:            log,
	}, nil
}

func (h *OauthHandler) Routes() chi.Router {
	// to be mounted at /oauth
	router := chi.NewRouter()

	router.Get("/login", h.LoginGet)
	router.Post("/login", h.LoginPost)

	router.Get("/authorize", h.AuthorizeGet)
	router.Post("/authorize", h.AuthorizePost)

	router.Post("/token", h.TokenPost)

	return router
}

// LoginGet handles the get request of a login page
func (h *OauthHandler) LoginGet(res http.ResponseWriter, req *http.Request) {
	if err := h.loginTemplate.Execute(res, false); err != nil {
		h.log.Error("oauth login error executing loginTemplate", util.Err(err))
	}
	// TODO: send 500 internal error page
}

// LoginPost handles the post and request of a login page submission
func (h *OauthHandler) LoginPost(res http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		h.log.Debug("could not parse form values", util.Err(err))
		if err := h.loginTemplate.Execute(res, true); err != nil {
			h.log.Error("oauth login error executing loginTemplate", util.Err(err))
		}
		return
	}

	username := req.FormValue("uname")
	password := req.FormValue("pass")
	_, sesh, err := h.authController.Login(req.Context(), username, password, req.UserAgent())
	if err != nil {
		if err := h.loginTemplate.Execute(res, true); err != nil {
			h.log.Error("oauth loging error executing loginTemplate", util.Err(err))
		}
		return
	}

	req.Method = http.MethodGet
	values := url.Values{}
	seshIdStr, err := util.StringFromPGUUID(sesh.ID)
	if err != nil {
		h.log.Error("oauth logging error parsing session id", util.Err(err))
		return
	}
	values.Add("sesh_key", seshIdStr)
	values.Add("client_id", req.URL.Query().Get("client_id"))
	values.Add("redirect_uri", req.URL.Query().Get("redirect_uri"))
	values.Add("state", req.URL.Query().Get("state"))

	http.Redirect(res, req, "/oauth/authorize"+"?"+values.Encode(), http.StatusSeeOther)
}

// AuthorizeGet handles rendering oauth authorize template
func (h *OauthHandler) AuthorizeGet(res http.ResponseWriter, req *http.Request) {
	err := h.authTemplate.Execute(res, nil)
	if err != nil {
		h.log.Error("oauth authorize error executing template", util.Err(err))
	}
	// TODO: send 500 internal page
}

// AuthorizePost receives a POST and retrieves a session(access) token, validates it and sents back user info
func (h *OauthHandler) AuthorizePost(res http.ResponseWriter, req *http.Request) {
	// setup redirect url
	redirectURI := strings.TrimSpace(req.URL.Query().Get("redirect_uri"))
	// add query params
	values := url.Values{}
	values.Add("state", req.URL.Query().Get("state"))

	// get session key, validate and get user info
	seshKey := strings.TrimSpace(req.URL.Query().Get("sesh_key"))
	seshID, err := uuid.Parse(seshKey)
	if err != nil {
		h.log.Info("oauth authorize error invalid session key", util.Err(err))
		values.Add("error", "invalid_request")
		http.Redirect(res, req, redirectURI+"?"+values.Encode(), http.StatusNotFound)
		return
	}
	user, err := h.authController.Authorize(req.Context(), seshID)
	if err != nil {
		h.log.Info("oauth authorize could not validate session id, redirecting to login page", util.Err(err))
		values.Add("error", "access_denied")
		http.Redirect(res, req, redirectURI+"?"+values.Encode(), http.StatusNotFound)
		return
	}

	// create auth code
	clientID := strings.TrimSpace(req.URL.Query().Get("client_id"))
	authCode, err := h.authController.CreateAuthCode(req.Context(), user.ID.Bytes, clientID)
	if err != nil {
		h.log.Error("error creating oauth authorization code", util.Err(err))
		values.Add("error", "server_error")
		http.Redirect(res, req, redirectURI+"?"+values.Encode(), http.StatusNotFound)
		return
	}

	// add code to query params
	values.Add("code", auth.EncodeKey(authCode.Code))

	// redirect
	http.Redirect(res, req, redirectURI+"?"+values.Encode(), http.StatusSeeOther)
}

// Token handles authenticating the oauth client with the given token
func (h *OauthHandler) TokenPost(res http.ResponseWriter, req *http.Request) {
	// authenticate client as per RFC 6749 2.3.1.
	id, secret, ok := req.BasicAuth()
	if !ok {
		h.log.Warn("oauth token error getting client credentials from basic auth")
		body, err := io.ReadAll(req.Body)
		if err == nil {
			h.log.Debug("body of request:", slog.String("body", string(body)))
		}
		sendTokenError(res, "unauthorized_client", h.log)
		return
	}
	err := h.authenticateClient(id, secret)
	if err != nil {
		h.log.Warn("oauth token error authenticating client", util.Err(err))
		sendTokenError(res, "unauthorized_client", h.log)
		return
	}

	// ^^^^^^^^^^ client is authenticated after above ^^^^^^^^^^
	var queryCode string
	// find grant type: refresh token else authorization code
	if err := req.ParseForm(); err != nil {
		h.log.Warn("oauth token error parsing form", util.Err(err))
		sendTokenError(res, "server_error", h.log)
		return
	}
	grantType := req.FormValue("grant_type")
	switch grantType {
	case "refresh_token":
		refreshToken := req.FormValue("refresh_token")
		accessToken, err := h.authController.ValidateRefreshToken(req.Context(), refreshToken)
		if err != nil {
			h.log.Warn("oauth token could not find token based on refresh token", util.Err(err))
			sendTokenError(res, "invalid_grant", h.log)
			return
		}
		queryCode = auth.EncodeKey(accessToken.AuthCode)
	case "authorization_code":
		queryCode = req.FormValue("code")
	default:
		sendTokenError(res, "invalid_grant", h.log)
		return
	}

	// validate auth code
	authCode, err := h.authController.ValidateAuthCode(req.Context(), queryCode)
	if err != nil {
		h.log.Warn("could not find auth code", util.Err(err))
		sendTokenError(res, "invalid_grant", h.log)
		return
	}
	// create access token
	token, err := h.authController.CreateAccessToken(req.Context(), authCode)
	if err != nil {
		h.log.Error("error oauth handler(Token), could not create access token", util.Err(err))
		sendTokenError(res, "invalid_request", h.log)
		return
	}
	// setup json
	type tokenResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	tRes := &tokenResponse{
		AccessToken:  auth.EncodeKey(token.Token),
		RefreshToken: auth.EncodeKey(token.RefreshToken),
		ExpiresIn:    3600,
	}
	// marshal data and send off
	json, err := json.Marshal(&tRes)
	if err != nil {
		h.log.Error("oauth token error marshalling json", util.Err(err))
		// TODO: not technically a token request error message, but this shouldn't happen
		sendTokenError(res, "server_error", h.log)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Cache-Control", "no-store")
	res.Header().Set("Pragma", "no-cache")
	if _, err = res.Write(json); err != nil {
		h.log.Error("oauth token error writing response", util.Err(err))
	}
}

func (h *OauthHandler) authenticateClient(id, secret string) error {
	for i, s := range h.clients {
		if i == id && s == secret {
			return nil
		}
	}
	return errors.New("Oauth.authenticateClient() no matching credentials")
}

func sendTokenError(res http.ResponseWriter, err string, log *slog.Logger) {
	type tokenErrorResponse struct {
		Error string `json:"error"`
	}
	errRes := &tokenErrorResponse{err}
	errResJson, _ := json.Marshal(errRes)
	res.WriteHeader(http.StatusBadRequest)
	if _, err := res.Write(errResJson); err != nil {
		log.Error("oauth send token error writing response", util.Err(err))
	}
}
