package twirp

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/sschwartz96/syncapod-backend/internal/auth"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	protos "github.com/sschwartz96/syncapod-backend/internal/gen"
	"github.com/twitchtv/twirp"
	"golang.org/x/crypto/acme/autocert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	authTokenKey = "Auth_Token"
	userIDKey    = "User_ID"
	prefix       = "/rpc"
)

var (
	excludedMethods = []string{"Activate", "Authenticate", "Authorize", "CreateAccount", "ResetPassword"}
)

// Server is truly needed for its Intercept method which authenticates users before accessing services, but also useful to have all the grpc server boilerplate contained within NewServer function
type Server struct {
	authC    *auth.AuthController
	services []service
}

type service struct {
	name        string
	twirpServer protos.TwirpServer
}

func NewServer(aC *auth.AuthController, aS protos.Auth, pS protos.Pod, adminS protos.Admin) *Server {
	s := &Server{authC: aC}
	twirpServices := []service{
		{
			name: "admin",
			twirpServer: protos.NewAdminServer(
				adminS,
				twirp.WithServerPathPrefix(prefix),
				twirp.WithServerHooks(s.authorizeAdminHook()),
			),
		},
		{
			name: "auth",
			twirpServer: protos.NewAuthServer(
				aS,
				twirp.WithServerPathPrefix(prefix),
				twirp.WithServerHooks(s.authorizeHook()),
			),
		},
		{
			name: "podcast",
			twirpServer: protos.NewPodServer(
				pS,
				twirp.WithServerPathPrefix(prefix),
				twirp.WithServerHooks(s.authorizeHook()),
			),
		},
	}
	s.services = twirpServices
	return s
}

func (s *Server) authorizeAdminHook() *twirp.ServerHooks {
	hooks := &twirp.ServerHooks{}
	hooks.RequestRouted = func(ctx context.Context) (context.Context, error) {
		// extract auth token from context
		authTokenString, ok := ctx.Value(twirpHeaderKey{}).(string)
		if !ok {
			return ctx, twirp.NotFound.Error("Auth Hook, Could Not Convert Auth Token to String")
		}

		authToken, err := uuid.Parse(authTokenString)
		if err != nil {
			return ctx, twirp.Unauthenticated.Error("Auth Hook, Could Not Parse Auth Token -> UUID ")
		}

		user, err := s.authC.Authorize(ctx, authToken)
		if err != nil {
			return ctx, twirp.Unauthenticated.Error("")
		}
		log.Println(user)
		if !user.IsAdmin {
			return ctx, twirp.PermissionDenied.Error("User is not admin")
		}
		ctx = context.WithValue(ctx, twirpHeaderKey{}, twirpCtxData{
			authToken: authToken,
			user:      user,
		})
		return ctx, nil

	}
	return hooks
}

func (s *Server) authorizeHook() *twirp.ServerHooks {
	hooks := &twirp.ServerHooks{}
	hooks.RequestRouted = func(ctx context.Context) (context.Context, error) {
		// get the method name
		methodName, ok := twirp.MethodName(ctx)
		if !ok {
			return ctx, twirp.NotFound.Error("Auth Hook, Method Not Found")
		}
		// allow certain RPC methods to go through
		if in(methodName, excludedMethods) {
			return ctx, nil
		}

		// extract auth token from context
		authTokenString, ok := ctx.Value(twirpHeaderKey{}).(string)
		if !ok {
			return ctx, twirp.NotFound.Error("Auth Hook, Could Not Convert Auth Token to String")
		}

		authToken, err := uuid.Parse(authTokenString)
		if err != nil {
			return ctx, twirp.Unauthenticated.Error("Auth Hook, Could Not Parse Auth Token -> UUID ")
		}

		user, err := s.authC.Authorize(ctx, authToken)
		if err != nil {
			return ctx, twirp.Unauthenticated.Error("")
		}
		ctx = context.WithValue(ctx, twirpHeaderKey{}, twirpCtxData{
			authToken: authToken,
			user:      user,
		})
		return ctx, nil

	}
	return hooks
}

// withAuthTokenMiddleware extracts the Auth_Token from header and inserts it into context
func withAuthTokenMiddleware(next http.Handler) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		// get the auth token from http header
		authToken := req.Header.Get(authTokenKey)

		newCtx := context.WithValue(req.Context(), twirpHeaderKey{}, authToken)

		// call original hander's ServeHTTP function
		next.ServeHTTP(res, req.WithContext(newCtx))
	}
}

type twirpHeaderKey struct{}

type twirpCtxData struct {
	authToken uuid.UUID
	user      *db.UserRow
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	for _, service := range s.services {
		mux.Handle(service.twirpServer.PathPrefix(), withAuthTokenMiddleware(service.twirpServer))
	}
	return http.ListenAndServe(":8081", mux)
}

func (s *Server) RegisterRouter(router *httprouter.Router) *httprouter.Router {
	for _, service := range s.services {
		router.POST(
			service.twirpServer.PathPrefix()+":method",
			toHttpRouterHandle(withAuthTokenMiddleware(service.twirpServer)),
		)
		// router.Handle(service.twirpServer.PathPrefix(), withAuthTokenMiddleware(service.twirpServer))
	}
	return router
}

func getCredsOpt(a *autocert.Manager) grpc.ServerOption {
	if a != nil {
		tlsConfig := &tls.Config{GetCertificate: a.GetCertificate}
		return grpc.Creds(
			credentials.NewTLS(
				tlsConfig,
			),
		)
	}
	return grpc.EmptyServerOption{}
}

func toHttpRouterHandle(handlerFunc http.HandlerFunc) httprouter.Handle {
	return func(res http.ResponseWriter, req *http.Request, p httprouter.Params) {
		handlerFunc(res, req)
	}
}

func in(s string, list []string) bool {
	for _, val := range list {
		if s == val {
			return true
		}
	}
	return false
}
