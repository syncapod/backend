package twirp

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/google/uuid"
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
)

// Server is truly needed for its Intercept method which authenticates users before accessing services, but also useful to have all the grpc server boilerplate contained within NewServer function
type Server struct {
	authC    *auth.AuthController
	services []TwirpService
}

type TwirpService struct {
	name        string
	twirpServer protos.TwirpServer
}

func NewServer(a *autocert.Manager, aC *auth.AuthController, aS protos.Auth, pS protos.Pod, adminS protos.Admin) *Server {
	s := &Server{authC: aC}
	twirpServices := []TwirpService{
		{
			name: "admin",
			twirpServer: protos.NewAdminServer(
				adminS,
				twirp.WithServerPathPrefix("/rpc/admin"),
				// twirp.WithServerInterceptors(s.authIntercept()),
				twirp.WithServerHooks(s.authorizeHook()),
			),
		},
		{
			name: "auth",
			twirpServer: protos.NewAuthServer(
				aS,
				twirp.WithServerPathPrefix("/rpc/auth"),
				// twirp.WithServerInterceptors(s.authIntercept()),
				twirp.WithServerHooks(s.authorizeHook()),
			),
		},
		{
			name: "podcast",
			twirpServer: protos.NewPodServer(
				pS,
				twirp.WithServerPathPrefix("/rpc/podcast"),
				// twirp.WithServerInterceptors(s.authIntercept()),
				twirp.WithServerHooks(s.authorizeHook()),
			),
		},
	}
	s.services = twirpServices
	return s
}

// could use this instead of hooks?
// func (s *Server) authIntercept() twirp.Interceptor {
// 	return func(next twirp.Method) twirp.Method {
// 		return func(ctx context.Context, req interface{}) (interface{}, error) {
// 			// get the method name
// 			methodName, ok := twirp.MethodName(ctx)
// 			if !ok {
// 				return nil, twirp.NotFound.Error("Auth Intercept, Method Not Found")
// 			}
// 			// if Authenticate method then allow the method to proceed
// 			if methodName == "Authenticate" {
// 				return next(ctx, req)
// 			}

// 			// check for header and auth token
// 			header, ok := twirp.HTTPRequestHeaders(ctx)
// 			if !ok {
// 				return nil, twirp.NotFound.Error("Auth Intercept, HTTP Header Not Present")
// 			}

// 			authTokenString := header.Get(authTokenKey)
// 			authToken, err := uuid.Parse(authTokenString)
// 			if err != nil {
// 				return ctx, twirp.Unauthenticated.Error("")
// 			}
// 			user, err := s.authC.Authorize(ctx, authToken)
// 			if err != nil {
// 				return ctx, twirp.Unauthenticated.Error("")
// 			}
// 			ctx = context.WithValue(ctx, twirpContextKey{}, twirpContextValue{
// 				authToken: authToken,
// 				user:      user,
// 			})
// 			return next(ctx, req)
// 		}
// 	}
// }

func (s *Server) authorizeHook() *twirp.ServerHooks {
	hooks := &twirp.ServerHooks{}
	hooks.RequestRouted = func(ctx context.Context) (context.Context, error) {
		// get the method name
		methodName, ok := twirp.MethodName(ctx)
		if !ok {
			return ctx, twirp.NotFound.Error("Auth Hook, Method Not Found")
		}
		// if Authenticate method then allow the method to proceed
		if methodName == "Authenticate" {
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
		// mux.Handle(service.twirpServer.PathPrefix(), service.twirpServer)
	}
	return http.ListenAndServe(":8081", mux)
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
