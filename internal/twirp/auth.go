package twirp

import (
	"context"
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
		return nil, twirp.InvalidArgumentError("acceptTerms", "user must accept terms")
	}

	// create account
	dob := time.Unix(req.DateOfBirth, 0)
	_, err := a.ac.CreateUser(ctx, req.Email, req.Username, req.Password, dob)
	if err != nil {
		if err.Error() == "email taken" {
			return nil, twirp.InvalidArgumentError("email", "email in use")
		}
		if err.Error() == "username taken" {
			return nil, twirp.InvalidArgumentError("email", "username in use")
		}
		return nil, twirp.Internal.Errorf("db error: %w", err)
	}

	return &protos.CreateAccountRes{}, nil
}

// ResetPassword method is called when user forgets password
func (a *AuthService) ResetPassword(ctx context.Context, req *protos.ResetPasswordReq) (*protos.ResetPasswordRes, error) {
	err := a.ac.ResetPassword(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	return &protos.ResetPasswordRes{}, nil
}

// Authenticate handles the authentication to syncapod and returns response
func (a *AuthService) Authenticate(ctx context.Context, req *protos.AuthenticateReq) (*protos.AuthenticateRes, error) {
	// don't even bother the db if username is empty or password is not valid (> 15)
	if req.Username == "" || len(req.Password) < 15 {
		return &protos.AuthenticateRes{
			SessionKey: "",
			User:       &protos.User{},
		}, nil
	}

	userRow, seshRow, err := a.ac.Login(ctx, req.Username, req.Password, req.UserAgent)
	if err != nil {
		return &protos.AuthenticateRes{
			SessionKey: "",
			User:       &protos.User{},
		}, nil
	}

	if req.Admin && !userRow.IsAdmin {
		return &protos.AuthenticateRes{
			SessionKey: "",
			User:       &protos.User{},
		}, nil
	}

	return &protos.AuthenticateRes{
		SessionKey: seshRow.ID.String(),
		User:       convertUserFromDB(userRow),
	}, nil
}

// Authorize authorizes user's session
func (a *AuthService) Authorize(ctx context.Context, req *protos.AuthorizeReq) (*protos.AuthorizeRes, error) {
	seshKey, err := uuid.Parse(req.GetSessionKey())
	if err != nil {
		return nil, twirp.InvalidArgument.Error("Malformed Session Key")
	}
	userRow, err := a.ac.Authorize(ctx, seshKey)
	if err != nil {
		return nil, twirp.Unauthenticated.Error("Session Invalid")
	}
	if req.Admin && !userRow.IsAdmin {
		return nil, twirp.PermissionDenied.Error("Not Admin")
	}
	return &protos.AuthorizeRes{
		User: convertUserFromDB(userRow),
	}, nil
}

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

// Activate activates account based on the activation token given
func (a *AuthService) Activate(ctx context.Context, req *protos.ActivateReq) (*protos.ActivateRes, error) {
	uuidToken, err := uuid.Parse(req.Token)
	if err != nil {
		return nil, twirp.InvalidArgumentError("token", "error parsing token")
	}
	_, err = a.ac.ActivateUser(ctx, uuidToken)
	if err != nil {
		return nil, twirp.Internal.Errorf("Error activating account: %w", err)
	}
	return &protos.ActivateRes{}, nil
}
