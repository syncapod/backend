package twirp

import (
	"context"
	"fmt"
	"strings"
	"time"

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

// CreateAccount verifies proper username, email, password, and acceptTerms fields
func (a *AuthService) CreateAccount(ctx context.Context, req *protos.CreateAccountReq) (*protos.CreateAccountRes, error) {
	// accept terms
	if !req.AcceptTerms {
		return &protos.CreateAccountRes{
			Error: "terms of use must be accepted",
		}, nil
	}

	// ensure user is older than 18
	if (time.Now().Unix() - req.DateOfBirth) < 567648000 {
		return &protos.CreateAccountRes{Error: "user must be at least 18 years old"}, nil
	}

	// password > 15 characters
	if len(req.Password) < 15 {
		return &protos.CreateAccountRes{Error: "password has to be at least than 15 characters"}, nil
	}

	// create account
	dob := time.Unix(req.DateOfBirth, 0)
	_, err := a.ac.CreateUser(ctx, req.Email, req.Username, req.Password, dob)
	if err != nil {
		if strings.Contains(err.Error(), "email") {
			return &protos.CreateAccountRes{Error: "email in use"}, nil
		}
		if strings.Contains(err.Error(), "username") {
			return &protos.CreateAccountRes{Error: "username in use"}, nil
		}
		return &protos.CreateAccountRes{
			Error: fmt.Sprintf("unknown error: %v", err.Error()),
		}, nil
	}

	// TODO: send activation email

	return &protos.CreateAccountRes{Error: ""}, nil
}

// ResetPassword method is called when user forgets password
func (a *AuthService) ResetPassword(ctx context.Context, req *protos.ResetPasswordReq) (*protos.ResetPasswordRes, error) {
	err := a.ac.ResetPassword(ctx, req.Email)
	return &protos.ResetPasswordRes{}, nil
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
