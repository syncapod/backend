package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

type AuthStorePG struct {
	db *pgxpool.Pool
}

// User
func (a *AuthStorePG) InsertUser(ctx context.Context, u *UserRow) error {
	_, err := a.db.Exec(ctx,
		"INSERT INTO users (id,email,username,birthdate,password_hash) VALUES($1,$2,$3,$4,$5)",
		u.ID, u.Email, u.Username, u.Birthdate, u.PasswordHash)
	if err != nil {
		return fmt.Errorf("InsertUser() error: %v", err)
	}
	return nil
}

func (a *AuthStorePG) GetUserByID(ctx context.Context, id uuid.UUID) (*UserRow, error) {
	u := &UserRow{}
	row := a.db.QueryRow(ctx, "SELECT * FROM users WHERE id=$1", id)
	err := row.Scan(&u.ID, &u.Email, &u.Username, &u.Birthdate, &u.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("GetUserByID() error: %v", err)
	}
	return u, nil
}

func (a *AuthStorePG) GetUserByEmail(ctx context.Context, email string) (*UserRow, error) {
	u := &UserRow{}
	row := a.db.QueryRow(ctx, "SELECT * FROM users WHERE email=$1", email)
	err := row.Scan(&u.ID, &u.Email, &u.Username, &u.Birthdate, &u.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("GetUserByEmail() error: %v", err)
	}
	return u, nil
}

func (a *AuthStorePG) GetUserByUsername(ctx context.Context, username string) (*UserRow, error) {
	u := &UserRow{}
	row := a.db.QueryRow(ctx, "SELECT * FROM users WHERE username=$1", username)
	err := row.Scan(&u.ID, &u.Email, &u.Username, &u.Birthdate, &u.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("GetUserByUsername() error: %v", err)
	}
	return u, nil
}

func (a *AuthStorePG) UpdateUser(ctx context.Context, u *UserRow) error {
	_, err := a.db.Exec(ctx,
		"UPDATE users SET email=$1,username=$2,birthdate=$3 WHERE id=$4",
		u.Email, u.Username, u.Birthdate, u.ID)
	if err != nil {
		return fmt.Errorf("UpdateUser() error: %v", err)
	}
	return nil
}

func (a AuthStorePG) UpdateUserPassword(ctx context.Context, id uuid.UUID, password_hash []byte) error {
	_, err := a.db.Exec(ctx,
		"UPDATE users SET password_hash=$1 WHERE id=$2",
		password_hash, id)
	if err != nil {
		return fmt.Errorf("UpdateUserPassword() error: %v", err)
	}
	return nil
}

func (a *AuthStorePG) DeleteUser(ctx context.Context, id uuid.UUID) error {
	_, err := a.db.Exec(ctx, "DELETE FROM users WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("DeleteUser() error: %v", err)
	}
	return nil
}

// Session
func (a *AuthStorePG) InsertSession(ctx context.Context, s *SessionRow) error {
	_, err := a.db.Exec(ctx,
		"INSERT INTO sessions (id,user_id,login_time,last_seen_time,expires,user_agent) VALUES($1,$2,$3,$4,$5,$6)",
		s.ID, s.UserID, s.LoginTime, s.LastSeenTime, s.Expires, s.UserAgent)
	if err != nil {
		return fmt.Errorf("InsertSession() error: %v", err)
	}
	return nil
}

func (a *AuthStorePG) GetSession(ctx context.Context, id uuid.UUID) (*SessionRow, error) {
	s := &SessionRow{}
	row := a.db.QueryRow(ctx, "SELECT * FROM sessions WHERE id=$1", id)
	err := row.Scan(&s.ID, &s.UserID, &s.LoginTime, &s.LastSeenTime, &s.Expires, &s.UserAgent)
	if err != nil {
		return nil, fmt.Errorf("GetSession() error: %v", err)
	}
	return s, err
}

func (a *AuthStorePG) UpdateSession(ctx context.Context, s *SessionRow) error {
	_, err := a.db.Exec(ctx,
		"UPDATE sessions SET user_id=$2,login_time=$3,last_seen_time=$4,expires=$5,user_agent=$6 WHERE id=$1",
		s.ID, s.UserID, s.LoginTime, s.LastSeenTime, s.Expires, s.UserAgent,
	)
	if err != nil {
		return fmt.Errorf("UpdateSession() error: %v", err)
	}
	return nil
}

func (a *AuthStorePG) DeleteSession(ctx context.Context, id uuid.UUID) error {
	_, err := a.db.Exec(ctx, "DELETE FROM sessions WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("DeleteSession() error: %v", err)
	}
	return nil
}
