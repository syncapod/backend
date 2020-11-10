package auth

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
	"github.com/sschwartz96/syncapod-backend/internal/db"
)

var (
	dbpg        *pgxpool.Pool
	authStore   db.AuthStore
	oauthStore  db.OAuthStore
	getTestUser = &db.UserRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Email: "get@test.auth", Username: "getTestAuth", Birthdate: time.Unix(0, 0).UTC()}
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
		dbpg, err = pgxpool.Connect(context.Background(),
			fmt.Sprintf(
				"postgres://postgres:secret@localhost:5432/postgres?sslmode=disable",
			),
		)
		time.Sleep(time.Millisecond * 50)
	}

	// setup db
	setupAuthDB()

	// setup store
	authStore = db.NewAuthStorePG(dbpg)
	oauthStore = db.NewOAuthStorePG(dbpg)

	// run tests
	runCode := m.Run()

	os.Exit(runCode)
}

func TestAuthController_Login(t *testing.T) {
	type fields struct {
		authStore  db.AuthStore
		oauthStore db.OAuthStore
	}
	type args struct {
		ctx      context.Context
		username string
		password string
		agent    string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *db.UserRow
		wantErr bool
	}{
		{
			name: "valid",
			args: args{ctx: context.Background(),
				agent:    "testAgent",
				password: "pass",
				username: getTestUser.Username,
			},
			fields:  fields{authStore: authStore, oauthStore: oauthStore},
			want:    getTestUser,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AuthController{
				authStore:  tt.fields.authStore,
				oauthStore: tt.fields.oauthStore,
			}
			got, got1, err := a.Login(tt.args.ctx, tt.args.username, tt.args.password, tt.args.agent)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthController.Login() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.ID, tt.want.ID) {
				t.Errorf("AuthController.Login() got = \n%v\n, want \n%v", got, tt.want)
			}
			_, err = a.authStore.GetSession(context.Background(), got1.ID)
			if err != nil {
				t.Error("AuthController.Login() did not find session in database")
			}
		})
	}
}

func TestAuthController_Authorize(t *testing.T) {
	type fields struct {
		authStore  db.AuthStore
		oauthStore db.OAuthStore
	}
	type args struct {
		ctx       context.Context
		sessionID uuid.UUID
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
			args:    args{ctx: context.Background(), sessionID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d87ae20c111")},
			fields:  fields{authStore: authStore, oauthStore: oauthStore},
			want:    getTestUser,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AuthController{
				authStore:  tt.fields.authStore,
				oauthStore: tt.fields.oauthStore,
			}
			got, err := a.Authorize(tt.args.ctx, tt.args.sessionID)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthController.Authorize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.ID, tt.want.ID) {
				t.Errorf("AuthController.Authorize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthController_Logout(t *testing.T) {
	type fields struct {
		authStore  db.AuthStore
		oauthStore db.OAuthStore
	}
	type args struct {
		ctx       context.Context
		sessionID uuid.UUID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "valid",
			args:    args{ctx: context.Background(), sessionID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d87ae20c222")},
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
			if err := a.Logout(tt.args.ctx, tt.args.sessionID); (err != nil) != tt.wantErr {
				t.Errorf("AuthController.Logout() error = %v, wantErr %v", err, tt.wantErr)
			}
			// make sure session is removed
			_, err := authStore.GetSession(context.Background(), tt.args.sessionID)
			if err == nil {
				t.Errorf("AuthController.Logout() session still found within database")
			}
		})
	}
}

func TestAuthController_findUserByEmailOrUsername(t *testing.T) {
	type fields struct {
		authStore  db.AuthStore
		oauthStore db.OAuthStore
	}
	type args struct {
		ctx context.Context
		u   string
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
			args:    args{ctx: context.Background(), u: getTestUser.Username},
			fields:  fields{authStore: authStore, oauthStore: oauthStore},
			want:    getTestUser,
			wantErr: false,
		},
		{
			name:    "valid",
			args:    args{ctx: context.Background(), u: getTestUser.Email},
			fields:  fields{authStore: authStore, oauthStore: oauthStore},
			want:    getTestUser,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AuthController{
				authStore:  tt.fields.authStore,
				oauthStore: tt.fields.oauthStore,
			}
			got, err := a.findUserByEmailOrUsername(tt.args.ctx, tt.args.u)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthController.findUserByEmailOrUsername() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.ID, tt.want.ID) {
				t.Errorf("AuthController.findUserByEmailOrUsername() = %v, want %v", got, tt.want)
			}
		})
	}
}

func setupAuthDB() {
	// test users
	getTestUser.PasswordHash = []byte("$2a$10$rUH2xp2xIt3ASkdpvH7duugL//F.HsqP58DKvcAAnTmXRWM0fSiRS")
	insertUser(getTestUser)
	getTestUser.PasswordHash = nil

	updateUser := &db.UserRow{ID: uuid.MustParse("b813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Email: "update@test.auth", Username: "updateAuth", Birthdate: time.Unix(10001, 0).UTC(), PasswordHash: []byte("pass")}
	insertUser(updateUser)
	updatePassUser := &db.UserRow{ID: uuid.MustParse("c813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Email: "updatePass@test.auth", Username: "updatePassAuth", Birthdate: time.Unix(10002, 0).UTC(), PasswordHash: []byte("pass")}
	insertUser(updatePassUser)
	deleteUser := &db.UserRow{ID: uuid.MustParse("d813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Email: "delete@test.auth", Username: "deleteAuth", Birthdate: time.Unix(10002, 0).UTC(), PasswordHash: []byte("pass")}
	insertUser(deleteUser)

	// test sessions
	getSesh := &db.SessionRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d87ae20c111"), UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"),
		Expires: time.Now().Add(time.Hour), LastSeenTime: time.Unix(1000, 0), LoginTime: time.Unix(1000, 0), UserAgent: "testAgent"}
	insertSession(getSesh)
	updateSesh := &db.SessionRow{ID: uuid.MustParse("b813c6e3-9cd0-4aed-9c4e-1d87ae20c111"), UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"),
		Expires: time.Unix(1000, 0), LastSeenTime: time.Unix(1000, 0), LoginTime: time.Unix(1000, 0), UserAgent: "testAgent"}
	insertSession(updateSesh)
	deleteSesh := &db.SessionRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d87ae20c222"), UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"),
		Expires: time.Unix(1000, 0), LastSeenTime: time.Unix(1000, 0), LoginTime: time.Unix(1000, 0), UserAgent: "testAgent"}
	insertSession(deleteSesh)

	// test auth codes
	gc, _ := DecodeKey("get_code")
	getAuth := &db.AuthCodeRow{Code: gc, ClientID: "get_client", Scope: "get_scope", UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Expires: time.Now().Add(time.Minute * 5)}
	insertAuthCode(getAuth)
	ec, _ := DecodeKey("expired_code")
	expiredAuth := &db.AuthCodeRow{Code: ec, ClientID: "client", Scope: "scope", UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Expires: time.Now().Add(time.Minute * -5)}
	insertAuthCode(expiredAuth)
	dc, _ := DecodeKey("delete_code")
	deleteAuth := &db.AuthCodeRow{Code: dc, ClientID: "client", Scope: "scope", UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Expires: time.Now().Add(time.Minute * 5)}
	insertAuthCode(deleteAuth)

	// test access tokens
	tk, _ := DecodeKey("token")
	rk, _ := DecodeKey("rftoken")
	getAccessByRefresh := &db.AccessTokenRow{AuthCode: gc, Created: time.Now(), Expires: 3600, RefreshToken: rk, Token: tk, UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba")}
	insertAccessToken(getAccessByRefresh)
	dtk, _ := DecodeKey("del_token")
	drk, _ := DecodeKey("del_rftoken")
	deleteToken := &db.AccessTokenRow{AuthCode: gc, Created: time.Unix(1000, 0), Expires: 3600, RefreshToken: drk, Token: dtk, UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba")}
	insertAccessToken(deleteToken)
}

func insertUser(u *db.UserRow) {
	_, err := dbpg.Exec(context.Background(),
		"INSERT INTO users (id,email,username,birthdate,password_hash) VALUES($1,$2,$3,$4,$5)",
		u.ID, u.Email, u.Username, u.Birthdate, u.PasswordHash)
	if err != nil {
		log.Fatalln("insertUser() error:", err)
	}
}

func insertSession(s *db.SessionRow) {
	_, err := dbpg.Exec(context.Background(),
		"INSERT INTO sessions (id,user_id,login_time,last_seen_time,expires,user_agent) VALUES($1,$2,$3,$4,$5,$6)",
		s.ID, s.UserID, s.LoginTime, s.LastSeenTime, s.Expires, s.UserAgent)
	if err != nil {
		log.Fatalln("insertSession() error:", err)
	}
}

func insertAuthCode(a *db.AuthCodeRow) {
	_, err := dbpg.Exec(context.Background(),
		"INSERT INTO AuthCodes (code,client_id,user_id,scope,expires) VALUES($1,$2,$3,$4,$5)",
		&a.Code, &a.ClientID, &a.UserID, &a.Scope, &a.Expires)
	if err != nil {
		log.Fatalln("insertAuthCode() error:", err)
	}
}
func insertAccessToken(a *db.AccessTokenRow) {
	_, err := dbpg.Exec(context.Background(),
		"INSERT INTO AccessTokens (token,auth_code,refresh_token,user_id,created,expires) VALUES($1,$2,$3,$4,$5,$6)",
		&a.Token, &a.AuthCode, &a.RefreshToken, &a.UserID, &a.Created, &a.Expires)
	if err != nil {
		log.Fatalln("insertAccessToken() error:", err)
	}
}
