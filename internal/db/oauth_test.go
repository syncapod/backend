package db

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

func TestOAuthStorePG_InsertAuthCode(t *testing.T) {
	type fields struct {
		db *pgxpool.Pool
	}
	type args struct {
		ctx context.Context
		a   *AuthCodeRow
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				ctx: context.Background(),
				a:   &AuthCodeRow{Code: []byte("test_code"), ClientID: "test_client", Scope: "test_scope", UserID: getUserID},
			},
			fields:  fields{db: testDB},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &OAuthStorePG{
				db: tt.fields.db,
			}
			if err := o.InsertAuthCode(tt.args.ctx, tt.args.a); (err != nil) != tt.wantErr {
				t.Errorf("OAuthStorePG.InsertAuthCode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOAuthStorePG_GetAuthCode(t *testing.T) {
	type fields struct {
		db *pgxpool.Pool
	}
	type args struct {
		ctx  context.Context
		code []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *AuthCodeRow
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				ctx:  context.Background(),
				code: []byte("get_code"),
			},
			fields:  fields{db: testDB},
			want:    &AuthCodeRow{Code: []byte("get_code"), ClientID: "get_client", Scope: "get_scope", UserID: getUserID, Expires: time.Unix(0, 1000)},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &OAuthStorePG{
				db: tt.fields.db,
			}
			got, err := o.GetAuthCode(tt.args.ctx, tt.args.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("OAuthStorePG.GetAuthCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OAuthStorePG.GetAuthCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOAuthStorePG_DeleteAuthCode(t *testing.T) {
	type fields struct {
		db *pgxpool.Pool
	}
	type args struct {
		ctx  context.Context
		code []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "valid",
			args:    args{ctx: context.Background(), code: []byte("delete_code")},
			fields:  fields{db: testDB},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &OAuthStorePG{
				db: tt.fields.db,
			}
			if err := o.DeleteAuthCode(tt.args.ctx, tt.args.code); (err != nil) != tt.wantErr {
				t.Errorf("OAuthStorePG.DeleteAuthCode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOAuthStorePG_InsertAccessToken(t *testing.T) {
	type fields struct {
		db *pgxpool.Pool
	}
	type args struct {
		ctx context.Context
		a   *AccessTokenRow
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
				a: &AccessTokenRow{AuthCode: []byte("get_code"),
					Created:      time.Unix(1000, 0),
					Expires:      3600,
					RefreshToken: []byte("token"),
					Token:        []byte("token"),
					UserID:       getUserID},
			},
			fields:  fields{db: testDB},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &OAuthStorePG{
				db: tt.fields.db,
			}
			if err := o.InsertAccessToken(tt.args.ctx, tt.args.a); (err != nil) != tt.wantErr {
				t.Errorf("OAuthStorePG.InsertAccessToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOAuthStorePG_GetAccessTokenByRefresh(t *testing.T) {
	type fields struct {
		db *pgxpool.Pool
	}
	type args struct {
		ctx          context.Context
		refreshToken []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *AccessTokenRow
		wantErr bool
	}{
		{
			name:    "valid",
			args:    args{ctx: context.Background(), refreshToken: []byte("refresh_token")},
			fields:  fields{db: testDB},
			want:    &AccessTokenRow{AuthCode: []byte("get_code"), Created: time.Unix(1000, 0), Expires: 3600, RefreshToken: []byte("refresh_token"), Token: []byte("refresh_token"), UserID: getUserID},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &OAuthStorePG{
				db: tt.fields.db,
			}
			got, err := o.GetAccessTokenByRefresh(tt.args.ctx, tt.args.refreshToken)
			if (err != nil) != tt.wantErr {
				t.Errorf("OAuthStorePG.GetAccessTokenByRefresh() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OAuthStorePG.GetAccessTokenByRefresh() = \n%v, \nwant %v", got, tt.want)
			}
		})
	}
}

func TestOAuthStorePG_DeleteAccessToken(t *testing.T) {
	type fields struct {
		db *pgxpool.Pool
	}
	type args struct {
		ctx   context.Context
		token []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "valid",
			args:    args{ctx: context.Background(), token: []byte("delete_token")},
			fields:  fields{db: testDB},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &OAuthStorePG{
				db: tt.fields.db,
			}
			if err := o.DeleteAccessToken(tt.args.ctx, tt.args.token); (err != nil) != tt.wantErr {
				t.Errorf("OAuthStorePG.DeleteAccessToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOAuthStorePG_GetAccessTokenAndUser(t *testing.T) {
	type fields struct {
		db *pgxpool.Pool
	}
	type args struct {
		ctx   context.Context
		token []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *UserRow
		want1   *AccessTokenRow
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				ctx:   context.Background(),
				token: []byte("refresh_token"),
			},
			fields: fields{db: testDB},
			want: &UserRow{ID: getUserID,
				Email:        "get@test.test",
				Username:     "get",
				Birthdate:    time.Unix(0, 0).UTC(),
				PasswordHash: []byte("pass"),
				Created:      time.Unix(0, 0),
				LastSeen:     time.Unix(0, 0),
			},
			want1: &AccessTokenRow{
				AuthCode:     []byte("get_code"),
				Created:      time.Unix(1000, 0),
				Expires:      3600,
				RefreshToken: []byte("refresh_token"),
				Token:        []byte("refresh_token"),
				UserID:       getUserID,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &OAuthStorePG{
				db: tt.fields.db,
			}
			got, got1, err := o.GetAccessTokenAndUser(tt.args.ctx, tt.args.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("OAuthStorePG.GetAccessTokenAndUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OAuthStorePG.GetAccessTokenAndUser() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("OAuthStorePG.GetAccessTokenAndUser() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
