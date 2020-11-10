package auth

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sschwartz96/syncapod-backend/internal/db"
)

func TestAuthController_CreateAuthCode(t *testing.T) {
	type fields struct {
		authStore  db.AuthStore
		oauthStore db.OAuthStore
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
			args:    args{ctx: context.Background(), clientID: "oauthClient", userID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba")},
			fields:  fields{authStore: authStore, oauthStore: oauthStore},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AuthController{
				authStore:  tt.fields.authStore,
				oauthStore: tt.fields.oauthStore,
			}
			got, err := a.CreateAuthCode(tt.args.ctx, tt.args.userID, tt.args.clientID)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthController.CreateAuthCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// lookup in db
			_, err = oauthStore.GetAuthCode(context.Background(), got.Code)
			if err != nil {
				t.Errorf("AuthController.CreateAuthCode() error finding newly created auth code: %v", err)
			}
		})
	}
}

func TestAuthController_CreateAccessToken(t *testing.T) {
	gc, _ := DecodeKey("get_code")
	type fields struct {
		authStore  db.AuthStore
		oauthStore db.OAuthStore
	}
	type args struct {
		ctx      context.Context
		authCode *db.AuthCodeRow
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "valid",
			args:    args{ctx: context.Background(), authCode: &db.AuthCodeRow{Code: gc, ClientID: "get_client", Scope: "get_scope", UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba")}},
			fields:  fields{authStore: authStore, oauthStore: oauthStore},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AuthController{
				authStore:  tt.fields.authStore,
				oauthStore: tt.fields.oauthStore,
			}
			got, err := a.CreateAccessToken(tt.args.ctx, tt.args.authCode)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthController.CreateAccessToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			_, _, err = oauthStore.GetAccessTokenAndUser(context.Background(), got.Token)
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
		authStore  db.AuthStore
		oauthStore db.OAuthStore
	}
	type args struct {
		ctx  context.Context
		code string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *db.AuthCodeRow
		wantErr bool
	}{
		{
			name:    "valid",
			args:    args{ctx: context.Background(), code: EncodeKey(gc)},
			fields:  fields{authStore: authStore, oauthStore: oauthStore},
			want:    &db.AuthCodeRow{Code: gc, ClientID: "get_client", Scope: "get_scope", UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba")},
			wantErr: false,
		},
		{
			name:    "expired",
			args:    args{ctx: context.Background(), code: EncodeKey(ec)},
			fields:  fields{authStore: authStore, oauthStore: oauthStore},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AuthController{
				authStore:  tt.fields.authStore,
				oauthStore: tt.fields.oauthStore,
			}
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
		authStore  db.AuthStore
		oauthStore db.OAuthStore
	}
	type args struct {
		ctx   context.Context
		token string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *db.UserRow
		wantErr bool
	}{
		{
			name:    "valid",
			args:    args{ctx: context.Background(), token: EncodeKey(tk)},
			fields:  fields{authStore: authStore, oauthStore: oauthStore},
			want:    &db.UserRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Email: "get@test.test", Username: "get", Birthdate: time.Unix(0, 0).UTC()},
			wantErr: false,
		},
		{
			name:    "expired",
			args:    args{ctx: context.Background(), token: EncodeKey(dtk)},
			fields:  fields{authStore: authStore, oauthStore: oauthStore},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AuthController{
				authStore:  tt.fields.authStore,
				oauthStore: tt.fields.oauthStore,
			}
			got, err := a.ValidateAccessToken(tt.args.ctx, tt.args.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthController.ValidateAccessToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil {
				if !reflect.DeepEqual(got.ID, tt.want.ID) {
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
		authStore  db.AuthStore
		oauthStore db.OAuthStore
	}
	type args struct {
		ctx   context.Context
		token string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *db.AccessTokenRow
		wantErr bool
	}{
		{
			name:    "valid",
			args:    args{ctx: context.Background(), token: EncodeKey(rk)},
			fields:  fields{authStore: authStore, oauthStore: oauthStore},
			want:    &db.AccessTokenRow{AuthCode: gc, Created: time.Now(), Expires: 3600, RefreshToken: rk, Token: tk, UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba")},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AuthController{
				authStore:  tt.fields.authStore,
				oauthStore: tt.fields.oauthStore,
			}
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
