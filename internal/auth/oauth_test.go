package auth

import (
	"context"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sschwartz96/syncapod-backend/internal/db_new"
	protos "github.com/sschwartz96/syncapod-backend/internal/gen"
	"github.com/sschwartz96/syncapod-backend/internal/util"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAuthController_CreateAuthCode(t *testing.T) {
	type fields struct {
		queries *db_new.Queries
	}
	type args struct {
		ctx      context.Context
		userID   uuid.UUID
		clientID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "valid",
			args:    args{ctx: context.Background(), clientID: "oauthClient", userID: getTestUser.ID.Bytes},
			fields:  fields{queries: queries},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAuthController(tt.fields.queries, slog.Default())

			got, err := a.CreateAuthCode(tt.args.ctx, tt.args.userID, tt.args.clientID)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthController.CreateAuthCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// lookup in db
			_, err = queries.GetAuthCode(context.Background(), got.Code)
			if err != nil {
				t.Errorf("AuthController.CreateAuthCode() error finding newly created auth code: %v", err)
			}
		})
	}
}

func TestAuthController_CreateAccessToken(t *testing.T) {
	gc, _ := DecodeKey("get_code")
	type fields struct {
		queries *db_new.Queries
	}
	type args struct {
		ctx      context.Context
		authCode *db_new.Authcode
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			args: args{ctx: context.Background(),
				authCode: &db_new.Authcode{
					Code: gc, ClientID: "get_client",
					Scope:  "get_scope",
					UserID: getTestUser.ID,
				},
			},
			fields:  fields{queries: queries},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAuthController(tt.fields.queries, slog.Default())
			got, err := a.CreateAccessToken(tt.args.ctx, tt.args.authCode)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthController.CreateAccessToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			_, err = queries.GetAccessTokenAndUser(context.Background(), got.Token)
			if err != nil {
				t.Errorf("AuthController.CreateAccessToken() error finding newly created access token: %v", err)
			}
		})
	}
}

func TestAuthController_ValidateAuthCode(t *testing.T) {
	gc, _ := DecodeKey("get_code")
	ec, _ := DecodeKey("expire_code")
	type fields struct {
		queries *db_new.Queries
	}
	type args struct {
		ctx  context.Context
		code string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *db_new.Authcode
		wantErr bool
	}{
		{
			name:    "valid",
			args:    args{ctx: context.Background(), code: EncodeKey(gc)},
			fields:  fields{queries: queries},
			want:    &db_new.Authcode{Code: gc, ClientID: "get_client", Scope: "get_scope", UserID: getTestUser.ID},
			wantErr: false,
		},
		{
			name:    "expired",
			args:    args{ctx: context.Background(), code: EncodeKey(ec)},
			fields:  fields{queries: queries},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAuthController(tt.fields.queries, slog.Default())
			got, err := a.ValidateAuthCode(tt.args.ctx, tt.args.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthController.ValidateAuthCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want == nil {
				return
			}
			if !reflect.DeepEqual(got.Code, tt.want.Code) {
				t.Errorf("AuthController.ValidateAuthCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthController_ValidateAccessToken(t *testing.T) {
	tk, _ := DecodeKey("token")
	dtk, _ := DecodeKey("del_token")
	type fields struct {
		queries *db_new.Queries
	}
	type args struct {
		ctx   context.Context
		token string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *protos.User
		wantErr bool
	}{
		{
			name:    "valid",
			args:    args{ctx: context.Background(), token: EncodeKey(tk)},
			fields:  fields{queries: queries},
			want:    &protos.User{Id: uuid.UUID(getTestUser.ID.Bytes).String(), Email: "get@test.test", Username: "get", DOB: timestamppb.New(time.Unix(0, 0).UTC())},
			wantErr: false,
		},
		{
			name:    "expired",
			args:    args{ctx: context.Background(), token: EncodeKey(dtk)},
			fields:  fields{queries: queries},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAuthController(tt.fields.queries, slog.Default())
			got, err := a.ValidateAccessToken(tt.args.ctx, tt.args.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthController.ValidateAccessToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil {
				if !reflect.DeepEqual(got.Id, tt.want.Id) {
					t.Errorf("AuthController.ValidateAccessToken() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestAuthController_ValidateRefreshToken(t *testing.T) {
	tk, _ := DecodeKey("token")
	rk, _ := DecodeKey("rftoken")
	gc, _ := DecodeKey("get_code")
	type fields struct {
		queries *db_new.Queries
	}
	type args struct {
		ctx   context.Context
		token string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *db_new.Accesstoken
		wantErr bool
	}{
		{
			name:    "valid",
			args:    args{ctx: context.Background(), token: EncodeKey(rk)},
			fields:  fields{queries: queries},
			want:    &db_new.Accesstoken{AuthCode: gc, Created: util.PGFromTime(time.Now()), Expires: 3600, RefreshToken: rk, Token: tk, UserID: getTestUser.ID},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAuthController(tt.fields.queries, slog.Default())
			got, err := a.ValidateRefreshToken(tt.args.ctx, tt.args.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthController.ValidateRefreshToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.Token, tt.want.Token) {
				t.Errorf("AuthController.ValidateRefreshToken() = \n%v\n, want: \n%v", got, tt.want)
			}
		})
	}
}

func Test_createKey(t *testing.T) {
	type args struct {
		l int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "valid",
			args:    args{l: 32},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createKey(tt.args.l)
			if (err != nil) != tt.wantErr {
				t.Errorf("createKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil {
				t.Errorf("createKey() nil byte array")
			}
		})
	}
}

func TestEncodeDecodeKey(t *testing.T) {
	encKey := EncodeKey([]byte("this_is_my_key_for_it_to_decode"))
	key, _ := DecodeKey(encKey)
	type args struct {
		key []byte
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "valid",
			args: args{key: key},
			want: encKey,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EncodeKey(tt.args.key); got != tt.want {
				t.Errorf("EncodeKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
