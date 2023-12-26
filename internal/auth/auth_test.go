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
	"github.com/sschwartz96/syncapod-backend/internal/db_new"
	"github.com/sschwartz96/syncapod-backend/internal/util"
	"github.com/stretchr/testify/require"
)

var (
	dbpg                    *pgxpool.Pool
	queries                 *db_new.Queries
	getTestUser             db_new.User
	getTestUserInsertParams = db_new.InsertUserParams{
		Email:     "get@test.auth",
		Created:   util.PGNow(),
		LastSeen:  util.PGNow(),
		Username:  "getTestAuth",
		Birthdate: util.PGDateFromTime(time.Unix(0, 0).UTC()),
	}

	getSesh, updateSesh, deleteSesh db_new.Session
)

// user TestMain to setup
func TestMain(m *testing.M) {
	var dockerCleanFunc func() error
	var err error
	dbpg, dockerCleanFunc, err = internal.StartDockerDB("db_auth")
	if err != nil {
		log.Fatalf("auth.TestMain() error setting up docker db: %v", err)
	}

	// setup store
	queries = db_new.New(dbpg)

	// setup db
	setupAuthDB()

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
		queries *db_new.Queries
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
		want    db_new.User
		wantErr bool
	}{
		{
			name: "valid",
			args: args{ctx: context.Background(),
				agent:    "testAgent",
				password: "pass",
				username: getTestUserInsertParams.Username,
			},
			fields:  fields{queries: queries},
			want:    getTestUser,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAuthController(tt.fields.queries, slog.Default())
			got, got1, err := a.Login(tt.args.ctx, tt.args.username, tt.args.password, tt.args.agent)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthController.Login() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.ID, tt.want.ID) {
				t.Errorf("AuthController.Login() got = \n%v\n, want \n%v", got, tt.want)
			}
			_, err = a.queries.GetSession(context.Background(), got1.ID)
			if err != nil {
				t.Error("AuthController.Login() did not find session in database")
			}
		})
	}
}

func TestAuthController_Authorize(t *testing.T) {
	type fields struct {
		queries *db_new.Queries
	}
	type args struct {
		ctx       context.Context
		sessionID uuid.UUID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    db_new.User
		wantErr bool
	}{
		{
			name:    "valid",
			args:    args{ctx: context.Background(), sessionID: uuid.UUID(getSesh.ID.Bytes)},
			fields:  fields{queries: queries},
			want:    getTestUser,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAuthController(tt.fields.queries, slog.Default())
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
		queries *db_new.Queries
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
			args:    args{ctx: context.Background(), sessionID: uuid.UUID(getSesh.ID.Bytes)},
			fields:  fields{queries: queries},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAuthController(queries, slog.Default())
			if err := a.Logout(tt.args.ctx, tt.args.sessionID); (err != nil) != tt.wantErr {
				t.Errorf("AuthController.Logout() error = %v, wantErr %v", err, tt.wantErr)
			}
			// make sure session is removed
			_, err := queries.GetSession(context.Background(), util.PGUUID(tt.args.sessionID))
			if err == nil {
				t.Errorf("AuthController.Logout() session still found within database")
			}
		})
	}
}

func TestAuthController_CreateUser(t *testing.T) {
	a := NewAuthController(queries, slog.Default())
	email, username, pwd := "testCreateUser@syncapod.com", "testCreateUser", "secret"
	u, err := a.CreateUser(context.Background(), email, username, pwd, time.Now())
	require.Nil(t, err)
	require.NotNil(t, u)
}

func TestAuthController_findUserByEmailOrUsername(t *testing.T) {
	type fields struct {
		queries *db_new.Queries
	}
	type args struct {
		ctx context.Context
		u   string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    db_new.User
		wantErr bool
	}{
		{
			name:    "valid",
			args:    args{ctx: context.Background(), u: getTestUserInsertParams.Username},
			fields:  fields{queries: queries},
			want:    getTestUser,
			wantErr: false,
		},
		{
			name:    "valid",
			args:    args{ctx: context.Background(), u: getTestUserInsertParams.Email},
			fields:  fields{queries: queries},
			want:    getTestUser,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAuthController(tt.fields.queries, slog.Default())
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
	getTestUserInsertParams.PasswordHash = []byte("$2a$10$rUH2xp2xIt3ASkdpvH7duugL//F.HsqP58DKvcAAnTmXRWM0fSiRS")
	getTestUser = insertUser(queries, getTestUserInsertParams)

	getTestUserInsertParams.PasswordHash = nil

	updateUser := db_new.InsertUserParams{
		Email:        "update@test.auth",
		Username:     "updateAuth",
		Birthdate:    util.PGDateFromTime(time.Unix(10001, 0).UTC()),
		PasswordHash: []byte("pass"),
		Created:      util.PGNow(),
		LastSeen:     util.PGNow(),
	}
	insertUser(queries, updateUser)

	updatePassUser := db_new.InsertUserParams{
		Email:        "updatePass@test.auth",
		Username:     "updatePassAuth",
		Birthdate:    util.PGDateFromTime(time.Unix(10002, 0).UTC()),
		PasswordHash: []byte("pass"),
		Created:      util.PGNow(),
		LastSeen:     util.PGNow(),
	}
	insertUser(queries, updatePassUser)

	deleteUser := db_new.InsertUserParams{
		Email:        "delete@test.auth",
		Username:     "deleteAuth",
		Birthdate:    util.PGDateFromTime(time.Unix(10002, 0).UTC()),
		PasswordHash: []byte("pass"),
		Created:      util.PGNow(),
		LastSeen:     util.PGNow(),
	}
	insertUser(queries, deleteUser)

	// test sessions
	getSeshParams := db_new.InsertSessionParams{UserID: getTestUser.ID,
		Expires: util.PGFromTime(time.Now().Add(time.Hour)), LastSeenTime: util.PGFromTime(time.Unix(1000, 0)), LoginTime: util.PGFromTime(time.Unix(1000, 0)), UserAgent: "testAgent"}
	getSesh = insertSession(queries, getSeshParams)
	updateSeshParams := db_new.InsertSessionParams{UserID: getTestUser.ID,
		Expires: util.PGFromTime(time.Unix(1000, 0)), LastSeenTime: util.PGFromTime(time.Unix(1000, 0)), LoginTime: util.PGFromTime(time.Unix(1000, 0)), UserAgent: "testAgent"}
	updateSesh = insertSession(queries, updateSeshParams)
	deleteSeshParams := db_new.InsertSessionParams{UserID: getTestUser.ID,
		Expires: util.PGFromTime(time.Unix(1000, 0)), LastSeenTime: util.PGFromTime(time.Unix(1000, 0)), LoginTime: util.PGFromTime(time.Unix(1000, 0)), UserAgent: "testAgent"}
	deleteSesh = insertSession(queries, deleteSeshParams)

	// test auth codes
	gc, _ := DecodeKey("get_code")
	getAuth := db_new.InsertAuthCodeParams{Code: gc, ClientID: "get_client", Scope: "get_scope", UserID: getTestUser.ID, Expires: util.PGFromTime(time.Now().Add(time.Minute * 5))}
	insertAuthCode(queries, getAuth)
	ec, _ := DecodeKey("expired_code")
	expiredAuth := db_new.InsertAuthCodeParams{Code: ec, ClientID: "client", Scope: "scope", UserID: getTestUser.ID, Expires: util.PGFromTime(time.Now().Add(time.Minute * -5))}
	insertAuthCode(queries, expiredAuth)
	dc, _ := DecodeKey("delete_code")
	deleteAuth := db_new.InsertAuthCodeParams{Code: dc, ClientID: "client", Scope: "scope", UserID: getTestUser.ID, Expires: util.PGFromTime(time.Now().Add(time.Minute * 5))}
	insertAuthCode(queries, deleteAuth)

	// test access tokens
	tk, _ := DecodeKey("token")
	rk, _ := DecodeKey("rftoken")
	getAccessByRefresh := db_new.InsertAccessTokenParams{AuthCode: gc, Created: util.PGFromTime(time.Now()), Expires: 3600, RefreshToken: rk, Token: tk, UserID: getTestUser.ID}
	insertAccessToken(queries, getAccessByRefresh)
	dtk, _ := DecodeKey("del_token")
	drk, _ := DecodeKey("del_rftoken")
	deleteToken := db_new.InsertAccessTokenParams{AuthCode: gc, Created: util.PGFromTime(time.Unix(1000, 0)), Expires: 3600, RefreshToken: drk, Token: dtk, UserID: getTestUser.ID}
	insertAccessToken(queries, deleteToken)
}

// insertUser is a helper function to insert the user or print failure
func insertUser(queries *db_new.Queries, u db_new.InsertUserParams) db_new.User {
	user, err := queries.InsertUser(context.Background(), u)
	if err != nil {
		log.Println("db.auth_test.insertUser() email:", u.Email)
		log.Fatalln("db.auth_test.insertUser() error:", err)
	}
	return user
}

func insertSession(queries *db_new.Queries, s db_new.InsertSessionParams) db_new.Session {
	session, err := queries.InsertSession(context.Background(), s)
	if err != nil {
		log.Fatalln("db.auth_test.insertSession() error:", err)
	}
	return session
}

func insertAuthCode(queries *db_new.Queries, a db_new.InsertAuthCodeParams) {
	err := queries.InsertAuthCode(context.Background(), a)
	if err != nil {
		log.Fatalln("db.auth_test.insertAuthCode() error:", err)
	}
}
func insertAccessToken(queries *db_new.Queries, a db_new.InsertAccessTokenParams) {
	err := queries.InsertAccessToken(context.Background(), a)
	if err != nil {
		log.Fatalln("db.auth_test.insertAccessToken() error:", err)
	}
}
