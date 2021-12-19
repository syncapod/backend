package grpc

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/google/uuid"
	"github.com/sschwartz96/syncapod-backend/internal/auth"
	"github.com/sschwartz96/syncapod-backend/internal/protos"
	"golang.org/x/crypto/acme/autocert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
)

// Server is truly needed for its Intercept method which authenticates users before accessing services, but also useful to have all the grpc server boilerplate contained within NewServer function
type Server struct {
	server *grpc.Server
	authC  *auth.AuthController
}

func NewServer(a *autocert.Manager, aC *auth.AuthController, aS protos.AuthServer, pS protos.PodServer, adminS protos.AdminServer) *Server {
	var grpcServer *grpc.Server
	s := &Server{authC: aC}
	// setup server
	gOptCreds := getCredsOpt(a)
	gOptInter := grpc.UnaryInterceptor(s.Intercept())
	grpcServer = grpc.NewServer(gOptCreds, gOptInter)
	s.server = grpcServer
	// register services
	reflection.Register(grpcServer)
	protos.RegisterAuthServer(s.server, aS)
	protos.RegisterPodServer(s.server, pS)
	protos.RegisterAdminServer(s.server, adminS)
	return s
}

func (s *Server) Start(lis net.Listener) error {
	return s.server.Serve(lis)
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

func (s *Server) Intercept() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		// if this is going to the Auth service allow through
		if strings.Contains(info.FullMethod, "protos.Auth") {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, errors.New("invalid metadata")
		}
		token := md.Get("token")
		if len(token) == 0 {
			return nil, errors.New("no access token sent")
		}

		uuidTkn, err := uuid.Parse(token[0])
		if err != nil {
			return nil, fmt.Errorf("invalid uuid token: %v", err)
		}
		user, err := s.authC.Authorize(ctx, uuidTkn)
		if err != nil {
			return nil, fmt.Errorf("invalid access token: %v", err)
		}

		newMD := md.Copy()
		newMD.Set("user_id", user.ID.String())
		newCtx := metadata.NewIncomingContext(ctx, newMD)

		return handler(newCtx, req)
	}
}
