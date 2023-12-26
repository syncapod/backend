package auth

import (
	"github.com/sschwartz96/syncapod-backend/internal/db"
	protos "github.com/sschwartz96/syncapod-backend/internal/gen"
	"github.com/sschwartz96/syncapod-backend/internal/util"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (a *AuthController) ConvertUserFromDB(ur *db.User) *protos.User {
	id, _ := util.StringFromPGUUID(ur.ID)
	return &protos.User{
		Id:       id,
		Email:    ur.Email,
		Username: ur.Username,
		DOB:      timestamppb.New(ur.Birthdate.Time),
	}
}
