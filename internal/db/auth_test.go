package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	db        *pgxpool.Pool
	getUserID = uuid.MustParse("c724c6e3-9cd0-4aed-9c4e-1d88ae20c8ba")
)

// user TestMain to setup
func TestMain(m *testing.M) {
	// connect stop after 5 seconds
	start := time.Now()
	fiveSec := time.Second * 5
	err := errors.New("start loop")
	for err != nil {
		if time.Since(start) > fiveSec {
			log.Fatal(`Could not connect to postgres\n
				Took longer than 5 seconds, maybe download postgres image`)
		}
		db, err = pgxpool.Connect(context.Background(),
			fmt.Sprintf(
				"postgres://postgres:secret@localhost:5432/postgres?sslmode=disable",
			),
		)
		time.Sleep(time.Millisecond * 250)
	}

	// setup db
	setupAuthDB()

	// run tests
	runCode := m.Run()

	db.Close()

	os.Exit(runCode)
}

func TestAuthStorePG_InsertUser(t *testing.T) {
	u := UserRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d78ae20c8ba"), Email: "testInsert@test.test", Username: "testInsert", Birthdate: time.Now(), PasswordHash: []byte("pass")}
	type fields struct {
		db *pgxpool.Pool
	}
	type args struct {
		ctx context.Context
		u   *UserRow
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "valid",
			args:    args{ctx: context.Background(), u: &u},
			fields:  fields{db: db},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AuthStorePG{
				db: tt.fields.db,
			}
			if err := a.InsertUser(tt.args.ctx, tt.args.u); (err != nil) != tt.wantErr {
				t.Errorf("AuthStorePG.InsertUser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthStorePG_GetUserByID(t *testing.T) {
	type fields struct {
		db *pgxpool.Pool
	}
	type args struct {
		ctx context.Context
		id  uuid.UUID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *UserRow
		wantErr bool
	}{
		{
			name:    "valid",
			args:    args{ctx: context.Background(), id: getUserID},
			fields:  fields{db: db},
			want:    &UserRow{ID: getUserID, Email: "get@test.test", Username: "get", Birthdate: time.Unix(0, 0).UTC(), PasswordHash: []byte("pass")},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AuthStorePG{
				db: tt.fields.db,
			}
			got, err := a.GetUserByID(tt.args.ctx, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthStorePG.GetUserByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AuthStorePG.GetUserByID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthStorePG_GetUserByEmail(t *testing.T) {
	type fields struct {
		db *pgxpool.Pool
	}
	type args struct {
		ctx   context.Context
		email string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *UserRow
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				ctx:   context.Background(),
				email: "get@test.test",
			},
			fields:  fields{db: db},
			want:    &UserRow{ID: getUserID, Email: "get@test.test", Username: "get", Birthdate: time.Unix(0, 0).UTC(), PasswordHash: []byte("pass")},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AuthStorePG{
				db: tt.fields.db,
			}
			got, err := a.GetUserByEmail(tt.args.ctx, tt.args.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthStorePG.GetUserByEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AuthStorePG.GetUserByEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthStorePG_GetUserByUsername(t *testing.T) {
	type fields struct {
		db *pgxpool.Pool
	}
	type args struct {
		ctx      context.Context
		username string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *UserRow
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				ctx:      context.Background(),
				username: "get",
			},
			fields:  fields{db: db},
			want:    &UserRow{ID: getUserID, Email: "get@test.test", Username: "get", Birthdate: time.Unix(0, 0).UTC(), PasswordHash: []byte("pass")},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AuthStorePG{
				db: tt.fields.db,
			}
			got, err := a.GetUserByUsername(tt.args.ctx, tt.args.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthStorePG.GetUserByUsername() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AuthStorePG.GetUserByUsername() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthStorePG_UpdateUser(t *testing.T) {
	type fields struct {
		db *pgxpool.Pool
	}
	type args struct {
		ctx context.Context
		u   *UserRow
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
				u:   &UserRow{ID: uuid.MustParse("b813c6e3-9cd0-4aed-9c4e-1d88ae20c777"), Email: "update@updated.test", Username: "updated", Birthdate: time.Unix(0, 0).UTC(), PasswordHash: []byte("pass")},
			},
			fields: fields{db: db},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AuthStorePG{
				db: tt.fields.db,
			}
			if err := a.UpdateUser(tt.args.ctx, tt.args.u); (err != nil) != tt.wantErr {
				t.Errorf("AuthStorePG.UpdateUser() error = %v, wantErr %v", err, tt.wantErr)
			}
			updated, err := a.GetUserByID(context.Background(), tt.args.u.ID)
			if err != nil {
				t.Errorf("AuthStorePG.UpdateUser() error finding updated value: %v", err)
			}
			if !reflect.DeepEqual(tt.args.u, updated) {
				t.Errorf("AuthStorePG.UpdateUser() error updated field does not match\nwant:%v\ngot: %v", tt.args.u, updated)
			}
		})
	}
}

func TestAuthStorePG_UpdateUserPassword(t *testing.T) {
	type fields struct {
		db *pgxpool.Pool
	}
	type args struct {
		ctx           context.Context
		id            uuid.UUID
		password_hash []byte
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
				ctx:           context.Background(),
				id:            uuid.MustParse("c813c6e3-9cd0-4aed-9c4e-1d88ae20c777"),
				password_hash: []byte("pass_updated"),
			},
			fields:  fields{db: db},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := AuthStorePG{
				db: tt.fields.db,
			}
			if err := a.UpdateUserPassword(tt.args.ctx, tt.args.id, tt.args.password_hash); (err != nil) != tt.wantErr {
				t.Errorf("AuthStorePG.UpdateUserPassword() error = %v, wantErr %v", err, tt.wantErr)
			}
			updated, err := a.GetUserByID(context.Background(), tt.args.id)
			if err != nil {
				t.Errorf("AuthStorePG.UpdateUserPassword() error finding updated value: %v", err)
			}
			if !reflect.DeepEqual(tt.args.password_hash, updated.PasswordHash) {
				t.Errorf("AuthStorePG.UpdateUserPassword() error updated field does not match")
			}

		})
	}
}

func TestAuthStorePG_DeleteUser(t *testing.T) {
	type fields struct {
		db *pgxpool.Pool
	}
	type args struct {
		ctx context.Context
		id  uuid.UUID
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
				id:  uuid.MustParse("d813c6e3-9cd0-4aed-9c4e-1d88ae20c777"),
			},
			fields:  fields{db: db},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AuthStorePG{
				db: tt.fields.db,
			}
			if err := a.DeleteUser(tt.args.ctx, tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("AuthStorePG.DeleteUser() error = %v, wantErr %v", err, tt.wantErr)
			}
			_, shouldErr := a.GetUserByID(context.Background(), tt.args.id)
			if shouldErr == nil {
				t.Errorf("AuthStorePG.DeleteUser() found deleted entry")
			}
		})
	}
}

func TestAuthStorePG_InsertSession(t *testing.T) {
	type fields struct {
		db *pgxpool.Pool
	}
	type args struct {
		ctx context.Context
		s   *SessionRow
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
				s: &SessionRow{ID: uuid.MustParse("a113c6e3-9cd0-4aed-9c4e-1d87ae20c8ba"), UserID: getUserID,
					Expires: time.Now(), LastSeenTime: time.Now(), LoginTime: time.Now(), UserAgent: "testAgent"},
			},
			fields:  fields{db: db},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AuthStorePG{
				db: tt.fields.db,
			}
			if err := a.InsertSession(tt.args.ctx, tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("AuthStorePG.InsertSession() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthStorePG_GetSession(t *testing.T) {
	type fields struct {
		db *pgxpool.Pool
	}
	type args struct {
		ctx context.Context
		id  uuid.UUID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *SessionRow
		wantErr bool
	}{
		{
			name:   "valid",
			args:   args{ctx: context.Background(), id: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d87ae20c8ba")},
			fields: fields{db: db},
			want: &SessionRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d87ae20c8ba"), UserID: getUserID,
				Expires: time.Unix(1000, 0), LastSeenTime: time.Unix(1000, 0), LoginTime: time.Unix(1000, 0), UserAgent: "testAgent"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AuthStorePG{
				db: tt.fields.db,
			}
			got, err := a.GetSession(tt.args.ctx, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthStorePG.GetSession() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AuthStorePG.GetSession() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthStorePG_UpdateSession(t *testing.T) {
	type fields struct {
		db *pgxpool.Pool
	}
	type args struct {
		ctx context.Context
		s   *SessionRow
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
				s: &SessionRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d87ae20c8bb"), UserID: getUserID,
					Expires: time.Unix(1000, 0), LastSeenTime: time.Unix(1000, 0), LoginTime: time.Unix(1000, 0), UserAgent: "testAgentUpdated"},
			},
			fields:  fields{db: db},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AuthStorePG{
				db: tt.fields.db,
			}
			if err := a.UpdateSession(tt.args.ctx, tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("AuthStorePG.UpdateSession() error = %v, wantErr %v", err, tt.wantErr)
			}
			updated, err := a.GetSession(context.Background(), tt.args.s.ID)
			if err != nil {
				t.Errorf("AuthStorePG.UpdateSession() error finding updated value: %v", err)
			}
			if !reflect.DeepEqual(tt.args.s, updated) {
				t.Errorf("AuthStorePG.UpdateSession() error updated field does not match")
			}
		})
	}
}

func TestAuthStorePG_DeleteSession(t *testing.T) {
	type fields struct {
		db *pgxpool.Pool
	}
	type args struct {
		ctx context.Context
		id  uuid.UUID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "valid",
			args:    args{ctx: context.Background(), id: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d87ae20c8bc")},
			fields:  fields{db: db},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AuthStorePG{
				db: tt.fields.db,
			}
			if err := a.DeleteSession(tt.args.ctx, tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("AuthStorePG.DeleteSession() error = %v, wantErr %v", err, tt.wantErr)
			}
			_, shouldErr := a.GetSession(context.Background(), tt.args.id)
			if shouldErr == nil {
				t.Errorf("AuthStorePG.DeleteSession() found deleted entry")
			}
		})
	}
}

func TestAuthStorePG_GetSessionAndUser(t *testing.T) {
	type fields struct {
		db *pgxpool.Pool
	}
	type args struct {
		ctx       context.Context
		sessionID uuid.UUID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *SessionRow
		want1   *UserRow
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				ctx:       context.Background(),
				sessionID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d87ae20c8ba")},
			fields: fields{db: db},
			want: &SessionRow{
				ID:           uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d87ae20c8ba"),
				UserID:       getUserID,
				Expires:      time.Unix(1000, 0),
				LastSeenTime: time.Unix(1000, 0),
				LoginTime:    time.Unix(1000, 0),
				UserAgent:    "testAgent"},
			want1: &UserRow{ID: getUserID,
				Email: "get@test.test", Username: "get",
				Birthdate: time.Unix(0, 0).UTC(), PasswordHash: []byte("pass")},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AuthStorePG{
				db: tt.fields.db,
			}
			got, got1, err := a.GetSessionAndUser(tt.args.ctx, tt.args.sessionID)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthStorePG.GetSessionAndUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AuthStorePG.GetSessionAndUser() got =\n%v, want \n%v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("AuthStorePG.GetSessionAndUser() got1 = \n%v, want \n%v", got1, tt.want1)
			}
		})
	}
}

func setupAuthDB() {
	// test users
	getUser := &UserRow{ID: getUserID, Email: "get@test.test", Username: "get", Birthdate: time.Unix(10000, 0).UTC(), PasswordHash: []byte("pass")}
	insertUser(getUser)
	updateUser := &UserRow{ID: uuid.MustParse("b813c6e3-9cd0-4aed-9c4e-1d88ae20c777"), Email: "update@test.test", Username: "update", Birthdate: time.Unix(10001, 0).UTC(), PasswordHash: []byte("pass")}
	insertUser(updateUser)
	updatePassUser := &UserRow{ID: uuid.MustParse("c813c6e3-9cd0-4aed-9c4e-1d88ae20c777"), Email: "updatePass@test.test", Username: "updatePass", Birthdate: time.Unix(10002, 0).UTC(), PasswordHash: []byte("pass")}
	insertUser(updatePassUser)
	deleteUser := &UserRow{ID: uuid.MustParse("d813c6e3-9cd0-4aed-9c4e-1d88ae20c777"), Email: "delete@test.test", Username: "delete", Birthdate: time.Unix(10002, 0).UTC(), PasswordHash: []byte("pass")}
	insertUser(deleteUser)

	// test sessions
	getSesh := &SessionRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d87ae20c8ba"), UserID: getUserID,
		Expires: time.Unix(1000, 0), LastSeenTime: time.Unix(1000, 0), LoginTime: time.Unix(1000, 0), UserAgent: "testAgent"}
	insertSession(getSesh)
	updateSesh := &SessionRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d87ae20c8bb"), UserID: getUserID,
		Expires: time.Unix(1000, 0), LastSeenTime: time.Unix(1000, 0), LoginTime: time.Unix(1000, 0), UserAgent: "testAgent"}
	insertSession(updateSesh)
	deleteSesh := &SessionRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d87ae20c8bc"), UserID: getUserID,
		Expires: time.Unix(1000, 0), LastSeenTime: time.Unix(1000, 0), LoginTime: time.Unix(1000, 0), UserAgent: "testAgent"}
	insertSession(deleteSesh)

	// test auth codes
	getAuth := &AuthCodeRow{Code: []byte("get_code"), ClientID: "get_client", Scope: "get_scope", UserID: getUserID, Expires: time.Unix(0, 1000)}
	insertAuthCode(getAuth)
	deleteAuth := &AuthCodeRow{Code: []byte("delete_code"), ClientID: "client", Scope: "scope", UserID: getUserID, Expires: time.Unix(0, 1500)}
	insertAuthCode(deleteAuth)

	// test access tokens
	getAccessByRefresh := &AccessTokenRow{AuthCode: []byte("get_code"), Created: time.Unix(1000, 0), Expires: 3600, RefreshToken: []byte("refresh_token"), Token: []byte("refresh_token"), UserID: getUserID}
	insertAccessToken(getAccessByRefresh)
	deleteToken := &AccessTokenRow{AuthCode: []byte("get_code"), Created: time.Unix(1000, 0), Expires: 3600, RefreshToken: []byte("asdf"), Token: []byte("delete_token"), UserID: getUserID}
	insertAccessToken(deleteToken)
}

func insertUser(u *UserRow) {
	_, err := db.Exec(context.Background(),
		"INSERT INTO users (id,email,username,birthdate,password_hash) VALUES($1,$2,$3,$4,$5)",
		&u.ID, &u.Email, &u.Username, &u.Birthdate, &u.PasswordHash)
	if err != nil {
		log.Println("db.auth_test.insertUser() id:", u.ID)
		log.Fatalln("db.auth_test.insertUser() error:", err)
	}
}

func insertSession(s *SessionRow) {
	_, err := db.Exec(context.Background(),
		"INSERT INTO sessions (id,user_id,login_time,last_seen_time,expires,user_agent) VALUES($1,$2,$3,$4,$5,$6)",
		s.ID, s.UserID, s.LoginTime, s.LastSeenTime, s.Expires, s.UserAgent)
	if err != nil {
		log.Fatalln("db.auth_test.insertSession() error:", err)
	}
}

func insertAuthCode(a *AuthCodeRow) {
	_, err := db.Exec(context.Background(),
		"INSERT INTO AuthCodes (code,client_id,user_id,scope,expires) VALUES($1,$2,$3,$4,$5)",
		&a.Code, &a.ClientID, &a.UserID, &a.Scope, &a.Expires)
	if err != nil {
		log.Fatalln("db.auth_test.insertAuthCode() error:", err)
	}
}
func insertAccessToken(a *AccessTokenRow) {
	_, err := db.Exec(context.Background(),
		"INSERT INTO AccessTokens (token,auth_code,refresh_token,user_id,created,expires) VALUES($1,$2,$3,$4,$5,$6)",
		&a.Token, &a.AuthCode, &a.RefreshToken, &a.UserID, &a.Created, &a.Expires)
	if err != nil {
		log.Fatalln("db.auth_test.insertAccessToken() error:", err)
	}
}
