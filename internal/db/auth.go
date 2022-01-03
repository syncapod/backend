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

func scanActivationRow(row scanner, p *ActivationRow) error {
	return row.Scan(&p.Token, &p.UserID, &p.Expires)
}

func scanPasswordResetRow(row scanner, p *PasswordResetRow) error {
	return row.Scan(&p.Token, &p.UserID, &p.Expires)
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

// InsertPasswordReset inserts a password reset object
func (a *AuthStorePG) InsertPasswordReset(ctx context.Context, p *PasswordResetRow) error {
	_, err := a.db.Exec(ctx,
		"INSERT INTO PasswordReset (token,user_id,expires) VALUES($1,$2,$3)",
		p.Token, p.UserID, p.Expires)
	if err != nil {
		return fmt.Errorf("InsertPasswordReset() error: %w", err)
	}
	return nil
}

// FindPasswordReset finds a password reset row based on a given token
func (a *AuthStorePG) FindPasswordReset(ctx context.Context, token uuid.UUID) (*PasswordResetRow, error) {
	p := &PasswordResetRow{}
	row := a.db.QueryRow(ctx, "SELECT * FROM PasswordReset WHERE token = $1", &token)
	err := scanPasswordResetRow(row, p)
	if err != nil {
		return nil, fmt.Errorf("FindPasswordReset() error: %w", err)
	}
	return p, nil
}

// DeletePasswordReset
func (a *AuthStorePG) DeletePasswordReset(ctx context.Context, token uuid.UUID) error {
	_, err := a.db.Exec(ctx, "DELETE FROM PasswordReset WHERE token = $1", &token)
	if err != nil {
		return fmt.Errorf("DeletePasswordReset() error: %w", err)
	}
	return nil
}

// InsertActivation inserts an activation token in the Activation table
func (a *AuthStorePG) InsertActivation(ctx context.Context, p *ActivationRow) error {
	_, err := a.db.Exec(ctx,
		"INSERT INTO Activation (token,user_id,expires) VALUES($1,$2,$3)",
		p.Token, p.UserID, p.Expires)
	if err != nil {
		return fmt.Errorf("InsertActivationRow() error: %w", err)
	}
	return nil
}

// FindActivation finds a activation row based on a given token
func (a *AuthStorePG) FindActivation(ctx context.Context, token uuid.UUID) (*ActivationRow, error) {
	p := &ActivationRow{}
	row := a.db.QueryRow(ctx, "SELECT * FROM Activation WHERE token = $1", &token)
	err := scanActivationRow(row, p)
	if err != nil {
		return nil, fmt.Errorf("FindActivation() error: %w", err)
	}
	return p, nil
}

// DeleteActivation deletes activation row
func (a *AuthStorePG) DeleteActivation(ctx context.Context, token uuid.UUID) error {
	_, err := a.db.Exec(ctx, "DELETE FROM Activation WHERE token = $1", &token)
	if err != nil {
		return fmt.Errorf("DeleteActivation() error: %w", err)
	}
	return nil
}

// UpdateUserActivated updates the activated field to true of user
func (a *AuthStorePG) UpdateUserActivated(ctx context.Context, userID uuid.UUID) error {
	result, err := a.db.Exec(ctx, "UPDATE Users SET activated = $1 WHERE id = $2", true, userID)
	if err != nil {
		return fmt.Errorf("UpdateUserActivated() error: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("UpdateUserActivated() error: no rows updated")
	}
	return nil
}
