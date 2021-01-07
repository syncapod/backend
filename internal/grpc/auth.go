package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/sschwartz96/syncapod-backend/internal/auth"
	"github.com/sschwartz96/syncapod-backend/internal/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthService is the gRPC service for authentication and authorization
type AuthService struct {
	*protos.UnimplementedAuthServer
	ac *auth.AuthController
}

// NewAuthService creates a new *AuthService
func NewAuthService(a *auth.AuthController) *AuthService {
	return &AuthService{ac: a}
}

// Authenticate handles the authentication to syncapod and returns response
func (a *AuthService) Authenticate(ctx context.Context, req *protos.AuthenticateReq) (*protos.AuthenticateRes, error) {
	userRow, seshRow, err := a.ac.Login(ctx, req.Username, req.Password, req.UserAgent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid username or password")
	}
	return &protos.AuthenticateRes{
		SessionKey: seshRow.ID.String(),
		User:       convertUserFromDB(userRow),
	}, nil
}

// Authorize authorizes user based on a session key
func (a *AuthService) Authorize(ctx context.Context, req *protos.AuthorizeReq) (*protos.AuthorizeRes, error) {
	seshKey, err := uuid.Parse(req.GetSessionKey())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "malformed session key uuid")
	}
	userRow, err := a.ac.Authorize(ctx, seshKey)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid session key")
	}
	return &protos.AuthorizeRes{
		User: convertUserFromDB(userRow),
	}, nil
}

// Logout removes the given session key
func (a *AuthService) Logout(ctx context.Context, req *protos.LogoutReq) (*protos.LogoutRes, error) {
	seshKey, err := uuid.Parse(req.GetSessionKey())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "malformed session key uuid")
	}
	err = a.ac.Logout(ctx, seshKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error: %v", err)
	}
	return &protos.LogoutRes{
		Success: true,
	}, nil
}
