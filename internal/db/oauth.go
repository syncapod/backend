package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
)

type OAuthStorePG struct {
	db *pgxpool.Pool
}

func NewOAuthStorePG(db *pgxpool.Pool) *OAuthStorePG {
	return &OAuthStorePG{db: db}
}

func (o *OAuthStorePG) InsertAuthCode(ctx context.Context, a *AuthCodeRow) error {
	_, err := o.db.Exec(ctx,
		"INSERT INTO AuthCodes (code,client_id,user_id,scope,expires) VALUES($1,$2,$3,$4,$5)",
		&a.Code, &a.ClientID, &a.UserID, &a.Scope, &a.Expires)
	if err != nil {
		return fmt.Errorf("InsertAuthCode() error: %v", err)
	}
	return nil
}

func (o *OAuthStorePG) GetAuthCode(ctx context.Context, code []byte) (*AuthCodeRow, error) {
	a := &AuthCodeRow{}
	row := o.db.QueryRow(ctx, "SELECT * FROM AuthCodes WHERE code=$1", &code)
	err := row.Scan(&a.Code, &a.ClientID, &a.UserID, &a.Scope, &a.Expires)
	if err != nil {
		return nil, fmt.Errorf("GetAuthCode() error scanning row: %v", err)
	}
	return a, nil
}

func (o *OAuthStorePG) DeleteAuthCode(ctx context.Context, code []byte) error {
	_, err := o.db.Exec(ctx, "DELETE FROM AuthCodes WHERE code=$1", &code)
	if err != nil {
		return fmt.Errorf("DeleteAuthCode() error deleting: %v", err)
	}
	return nil
}

func (o *OAuthStorePG) InsertAccessToken(ctx context.Context, a *AccessTokenRow) error {
	_, err := o.db.Exec(ctx,
		"INSERT INTO AccessTokens (token,auth_code,refresh_token,user_id,created,expires) VALUES($1,$2,$3,$4,$5,$6)",
		&a.Token, &a.AuthCode, &a.RefreshToken, &a.UserID, &a.Created, &a.Expires)
	if err != nil {
		return fmt.Errorf("InsertAccessToken() error: %v", err)
	}
	return nil
}

func (o *OAuthStorePG) GetAccessTokenByRefresh(ctx context.Context, refreshToken []byte) (*AccessTokenRow, error) {
	a := &AccessTokenRow{}
	row := o.db.QueryRow(ctx, "SELECT * FROM AccessTokens WHERE refresh_token=$1", &refreshToken)
	err := row.Scan(&a.Token, &a.AuthCode, &a.RefreshToken, &a.UserID, &a.Created, &a.Expires)
	if err != nil {
		return nil, fmt.Errorf("GetAccessTokenByRefresh() error scanning row: %v", err)
	}
	return a, nil
}

func (o *OAuthStorePG) DeleteAccessToken(ctx context.Context, token []byte) error {
	_, err := o.db.Exec(ctx, "DELETE FROM AccessTokens WHERE token=$1", &token)
	if err != nil {
		return fmt.Errorf("DeleteAccessToken() error deleting: %v", err)
	}
	return nil
}

func (o *OAuthStorePG) GetAccessTokenAndUser(ctx context.Context, token []byte) (*UserRow, *AccessTokenRow, error) {
	a := &AccessTokenRow{}
	u := &UserRow{}
	result := o.db.QueryRow(ctx,
		"SELECT * FROM AccessTokens a JOIN Users u ON a.user_id=u.id WHERE a.token=$1",
		&token,
	)
	err := result.Scan(
		&a.Token, &a.AuthCode, &a.RefreshToken, &a.UserID, &a.Created, &a.Expires,
		&u.ID, &u.Email, &u.Username, &u.Birthdate, &u.PasswordHash, &u.Created, &u.LastSeen, &u.Activated,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("GetAccessTokenAndUser() error: %v", err)
	}
	return u, a, nil
}
