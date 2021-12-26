package twirp

import (
	"context"

	"github.com/google/uuid"
	"github.com/sschwartz96/syncapod-backend/internal/auth"
	protos "github.com/sschwartz96/syncapod-backend/internal/gen"
	"github.com/twitchtv/twirp"
)

// AuthService is the twirp service for authentication and authorization
type AuthService struct {
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
		return nil, twirp.InvalidArgument.Errorf("Error on login: %w", err)
	}
	return &protos.AuthenticateRes{
		SessionKey: seshRow.ID.String(),
		User:       convertUserFromDB(userRow),
	}, nil
}

// Authorize TODO: find use case
// func (a *AuthService) Authorize(ctx context.Context, req *protos.AuthorizeReq) (*protos.AuthorizeRes, error) {
// 	seshKey, err := uuid.Parse(req.GetSessionKey())
// 	if err != nil {
// 		return nil, twirp.InvalidArgument.Error("Malformed Session Key")
// 	}
// 	userRow, err := a.ac.Authorize(ctx, seshKey)
// 	if err != nil {
// 		return nil, twirp.Unauthenticated.Error("Session Invalid")
// 	}
// 	return &protos.AuthorizeRes{
// 		User: convertUserFromDB(userRow),
// 	}, nil
// }

// Logout removes the given session key from the db, in effect "logging out" of the user's session
func (a *AuthService) Logout(ctx context.Context, req *protos.LogoutReq) (*protos.LogoutRes, error) {
	seshKey, err := uuid.Parse(req.GetSessionKey())
	if err != nil {
		return nil, twirp.InvalidArgument.Error("Malformed session key uuid")
	}
	err = a.ac.Logout(ctx, seshKey)
	if err != nil {
		return nil, twirp.Internal.Errorf("Logout error: %w", err)
	}
	return &protos.LogoutRes{
		Success: true,
	}, nil
}
