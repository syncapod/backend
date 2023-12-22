package auth

import (
	"context"
	"log"
	"log/slog"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sschwartz96/syncapod-backend/internal"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	"github.com/sschwartz96/syncapod-backend/internal/db_new"
	"github.com/stretchr/testify/require"
)

var (
	dbpg        *pgxpool.Pool
	authStore   db.AuthStore
	oauthStore  db.OAuthStore
	queries     *db_new.Queries
	getTestUser = &db.UserRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Email: "get@test.auth", Username: "getTestAuth", Birthdate: time.Unix(0, 0).UTC()}
)

// user TestMain to setup
func TestMain(m *testing.M) {
	var dockerCleanFunc func() error
	var err error
	dbpg, dockerCleanFunc, err = internal.StartDockerDB("db_auth")
	if err != nil {
		log.Fatalf("auth.TestMain() error setting up docker db: %v", err)
	}

	// setup db
	setupAuthDB()

	// setup store
	authStore = db.NewAuthStorePG(dbpg)
	oauthStore = db.NewOAuthStorePG(dbpg)
	queries = db_new.New(dbpg)

	// run tests
	runCode := m.Run()

	// purge docker resources
	err = dockerCleanFunc()
	if err != nil {
		log.Fatalf("auth.TestMain() error purging docker resources: %v", err)
	}

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

func TestAuthController_CreateUser(t *testing.T) {
	a := &AuthController{
		authStore:  authStore,
		oauthStore: oauthStore,
		queries:    queries,
		log:        slog.Default(),
	}
	email, username, pwd := "testCreateUser@syncapod.com", "testCreateUser", "secret"
	u, err := a.CreateUser(context.Background(), email, username, pwd, time.Now())
	require.Nil(t, err)
	require.NotNil(t, u)
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
	a := db.NewAuthStorePG(dbpg)
	// test users
	getTestUser.PasswordHash = []byte("$2a$10$rUH2xp2xIt3ASkdpvH7duugL//F.HsqP58DKvcAAnTmXRWM0fSiRS")
	insertUser(a, getTestUser)
	getTestUser.PasswordHash = nil

	updateUser := &db.UserRow{ID: uuid.MustParse("b813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Email: "update@test.auth", Username: "updateAuth", Birthdate: time.Unix(10001, 0).UTC(), PasswordHash: []byte("pass")}
	insertUser(a, updateUser)
	updatePassUser := &db.UserRow{ID: uuid.MustParse("c813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Email: "updatePass@test.auth", Username: "updatePassAuth", Birthdate: time.Unix(10002, 0).UTC(), PasswordHash: []byte("pass")}
	insertUser(a, updatePassUser)
	deleteUser := &db.UserRow{ID: uuid.MustParse("d813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Email: "delete@test.auth", Username: "deleteAuth", Birthdate: time.Unix(10002, 0).UTC(), PasswordHash: []byte("pass")}
	insertUser(a, deleteUser)

	// test sessions
	getSesh := &db.SessionRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d87ae20c111"), UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"),
		Expires: time.Now().Add(time.Hour), LastSeenTime: time.Unix(1000, 0), LoginTime: time.Unix(1000, 0), UserAgent: "testAgent"}
	insertSession(a, getSesh)
	updateSesh := &db.SessionRow{ID: uuid.MustParse("b813c6e3-9cd0-4aed-9c4e-1d87ae20c111"), UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"),
		Expires: time.Unix(1000, 0), LastSeenTime: time.Unix(1000, 0), LoginTime: time.Unix(1000, 0), UserAgent: "testAgent"}
	insertSession(a, updateSesh)
	deleteSesh := &db.SessionRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d87ae20c222"), UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"),
		Expires: time.Unix(1000, 0), LastSeenTime: time.Unix(1000, 0), LoginTime: time.Unix(1000, 0), UserAgent: "testAgent"}
	insertSession(a, deleteSesh)

	o := db.NewOAuthStorePG(dbpg)
	// test auth codes
	gc, _ := DecodeKey("get_code")
	getAuth := &db.AuthCodeRow{Code: gc, ClientID: "get_client", Scope: "get_scope", UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Expires: time.Now().Add(time.Minute * 5)}
	insertAuthCode(o, getAuth)
	ec, _ := DecodeKey("expired_code")
	expiredAuth := &db.AuthCodeRow{Code: ec, ClientID: "client", Scope: "scope", UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Expires: time.Now().Add(time.Minute * -5)}
	insertAuthCode(o, expiredAuth)
	dc, _ := DecodeKey("delete_code")
	deleteAuth := &db.AuthCodeRow{Code: dc, ClientID: "client", Scope: "scope", UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Expires: time.Now().Add(time.Minute * 5)}
	insertAuthCode(o, deleteAuth)

	// test access tokens
	tk, _ := DecodeKey("token")
	rk, _ := DecodeKey("rftoken")
	getAccessByRefresh := &db.AccessTokenRow{AuthCode: gc, Created: time.Now(), Expires: 3600, RefreshToken: rk, Token: tk, UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba")}
	insertAccessToken(o, getAccessByRefresh)
	dtk, _ := DecodeKey("del_token")
	drk, _ := DecodeKey("del_rftoken")
	deleteToken := &db.AccessTokenRow{AuthCode: gc, Created: time.Unix(1000, 0), Expires: 3600, RefreshToken: drk, Token: dtk, UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba")}
	insertAccessToken(o, deleteToken)
}

func insertUser(a *db.AuthStorePG, u *db.UserRow) {
	err := a.InsertUser(context.Background(), u)
	if err != nil {
		log.Println("db.auth_test.insertUser() id:", u.ID)
		log.Println("db.auth_test.insertUser() id:", u.Email)
		log.Fatalln("db.auth_test.insertUser() error:", err)
	}
}

func insertSession(a *db.AuthStorePG, s *db.SessionRow) {
	err := a.InsertSession(context.Background(), s)
	if err != nil {
		log.Fatalln("db.auth_test.insertSession() error:", err)
	}
}

func insertAuthCode(o *db.OAuthStorePG, a *db.AuthCodeRow) {
	err := o.InsertAuthCode(context.Background(), a)
	if err != nil {
		log.Fatalln("db.auth_test.insertAuthCode() error:", err)
	}
}
func insertAccessToken(o *db.OAuthStorePG, a *db.AccessTokenRow) {
	err := o.InsertAccessToken(context.Background(), a)
	if err != nil {
		log.Fatalln("db.auth_test.insertAccessToken() error:", err)
	}
}
