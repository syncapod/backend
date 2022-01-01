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

func NewAuthStorePG(db *pgxpool.Pool) *AuthStorePG {
	return &AuthStorePG{db: db}
}

func scanUserRow(row scanner, u *UserRow) error {
	return row.Scan(&u.ID, &u.Email, &u.Username, &u.Birthdate, &u.PasswordHash, &u.Created, &u.LastSeen, &u.Activated)
}

// User
func (a *AuthStorePG) InsertUser(ctx context.Context, u *UserRow) error {
	_, err := a.db.Exec(ctx,
		"INSERT INTO Users (id,email,username,birthdate,password_hash,created,last_seen,activated) VALUES($1,$2,$3,$4,$5,$6,$7,$8)",
		&u.ID, &u.Email, &u.Username, &u.Birthdate, &u.PasswordHash, &u.Created, &u.LastSeen, &u.Activated)
	if err != nil {
		return fmt.Errorf("InsertUser() error: %v", err)
	}
	return nil
}

func (a *AuthStorePG) FindUserByID(ctx context.Context, id uuid.UUID) (*UserRow, error) {
	u := &UserRow{}
	row := a.db.QueryRow(ctx, "SELECT * FROM Users WHERE id=$1", id)
	err := scanUserRow(row, u)
	if err != nil {
		return nil, fmt.Errorf("GetUserByID() error: %v", err)
	}
	return u, nil
}

func (a *AuthStorePG) FindUserByEmail(ctx context.Context, email string) (*UserRow, error) {
	u := &UserRow{}
	row := a.db.QueryRow(ctx, "SELECT * FROM Users WHERE LOWER(email)=LOWER($1)", email)
	err := scanUserRow(row, u)
	if err != nil {
		return nil, fmt.Errorf("GetUserByEmail() error: %v", err)
	}
	return u, nil
}

func (a *AuthStorePG) FindUserByUsername(ctx context.Context, username string) (*UserRow, error) {
	u := &UserRow{}
	row := a.db.QueryRow(ctx, "SELECT * FROM Users WHERE LOWER(username)=LOWER($1)", username)
	err := scanUserRow(row, u)
	if err != nil {
		return nil, fmt.Errorf("GetUserByUsername() error: %v", err)
	}
	return u, nil
}

// TODO: probably need specific update functions
// func (a *AuthStorePG) UpdateUser(ctx context.Context, u *UserRow) error {
// 	_, err := a.db.Exec(ctx,
// 		"UPDATE Users SET email=$2,username=$3,birthdate=$4,last_seen=$5 WHERE id=$1",
// 		&u.ID, &u.Email, &u.Username, &u.Birthdate, &u.LastSeen)
// 	if err != nil {
// 		return fmt.Errorf("UpdateUser() error: %v", err)
// 	}
// 	return nil
// }

func (a AuthStorePG) UpdateUserPassword(ctx context.Context, id uuid.UUID, password_hash []byte) error {
	_, err := a.db.Exec(ctx,
		"UPDATE Users SET password_hash=$1 WHERE id=$2",
		password_hash, id)
	if err != nil {
		return fmt.Errorf("UpdateUserPassword() error: %v", err)
	}
	return nil
}

func (a *AuthStorePG) DeleteUser(ctx context.Context, id uuid.UUID) error {
	_, err := a.db.Exec(ctx, "DELETE FROM Users WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("DeleteUser() error: %v", err)
	}
	return nil
}

// Session
func (a *AuthStorePG) InsertSession(ctx context.Context, s *SessionRow) error {
	_, err := a.db.Exec(ctx,
		"INSERT INTO Sessions (id,user_id,login_time,last_seen_time,expires,user_agent) VALUES($1,$2,$3,$4,$5,$6)",
		s.ID, s.UserID, s.LoginTime, s.LastSeenTime, s.Expires, s.UserAgent)
	if err != nil {
		return fmt.Errorf("InsertSession() error: %v", err)
	}
	return nil
}

func (a *AuthStorePG) GetSession(ctx context.Context, id uuid.UUID) (*SessionRow, error) {
	s := &SessionRow{}
	row := a.db.QueryRow(ctx, "SELECT * FROM Sessions WHERE id=$1", id)
	err := row.Scan(&s.ID, &s.UserID, &s.LoginTime, &s.LastSeenTime, &s.Expires, &s.UserAgent)
	if err != nil {
		return nil, fmt.Errorf("GetSession() error: %v", err)
	}
	return s, err
}

func (a *AuthStorePG) UpdateSession(ctx context.Context, s *SessionRow) error {
	_, err := a.db.Exec(ctx,
		"UPDATE Sessions SET user_id=$2,login_time=$3,last_seen_time=$4,expires=$5,user_agent=$6 WHERE id=$1",
		s.ID, s.UserID, s.LoginTime, s.LastSeenTime, s.Expires, s.UserAgent,
	)
	if err != nil {
		return fmt.Errorf("UpdateSession() error: %v", err)
	}
	return nil
}

func (a *AuthStorePG) DeleteSession(ctx context.Context, id uuid.UUID) error {
	_, err := a.db.Exec(ctx, "DELETE FROM Sessions WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("DeleteSession() error: %v", err)
	}
	return nil
}

func (a *AuthStorePG) GetSessionAndUser(ctx context.Context, sessionID uuid.UUID) (*SessionRow, *UserRow, error) {
	s := &SessionRow{}
	u := &UserRow{}
	result := a.db.QueryRow(ctx,
		"SELECT * FROM Sessions s JOIN Users u ON s.user_id=u.id WHERE s.id=$1",
		&sessionID,
	)
	err := result.Scan(
		&s.ID, &s.UserID, &s.LoginTime, &s.LastSeenTime, &s.Expires, &s.UserAgent,
		&u.ID, &u.Email, &u.Username, &u.Birthdate, &u.PasswordHash, &u.Created, &u.LastSeen, &u.Activated,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("GetSessionAndUser() error: %v", err)
	}
	return s, u, nil
}
