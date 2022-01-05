package twirp

import (
	"context"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sschwartz96/syncapod-backend/internal"
	"github.com/sschwartz96/syncapod-backend/internal/auth"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	protos "github.com/sschwartz96/syncapod-backend/internal/gen"
	"github.com/sschwartz96/syncapod-backend/internal/podcast"
	"github.com/stretchr/testify/require"
	"github.com/twitchtv/twirp"
)

var (
	dbpg     *pgxpool.Pool
	testUser = &db.UserRow{
		ID:           uuid.MustParse("b921c6e3-9cd0-4aed-9c4e-1d88ae20c777"),
		Email:        "user@twirp.test",
		Username:     "user_twirp_test",
		Birthdate:    time.Unix(0, 0).UTC(),
		PasswordHash: []byte("$2y$12$ndywn/c6wcB0oPv1ZRMLgeSQjTpXzOUCQy.5vdYvJxO9CS644i6Ce"),
		Created:      time.Unix(0, 0),
		LastSeen:     time.Unix(0, 0),
		Activated:    false,
	}
)

type mailStub struct{}

func (m *mailStub) Queue(to, subject, body string) {}

func insertUser(u *db.UserRow) {
	a := db.NewAuthStorePG(dbpg)
	err := a.InsertUser(context.Background(), u)
	if err != nil {
		log.Println("db.auth_test.insertUser() id:", u.ID)
		log.Fatalln("db.auth_test.insertUser() error:", err)
	}
}

func TestMain(m *testing.M) {
	var dockerCleanFunc func() error
	var err error
	dbpg, dockerCleanFunc, err = internal.StartDockerDB("db_auth")
	if err != nil {
		log.Fatalf("auth.TestMain() error setting up docker db: %v", err)
	}

	// setup db
	err = setupAuthDB()
	if err != nil {
		log.Fatalf("twirp.TestMain() error setting up auth database: %v", err)
	}
	err = setupPodDB()
	if err != nil {
		log.Fatalf("twirp.TestMain() error setting up podcast database: %v", err)
	}
	err = setupAdmin()
	if err != nil {
		log.Fatalf("twirp.TestMain() error setting up db for admin: %v", err)
	}

	authController := auth.NewAuthController(db.NewAuthStorePG(dbpg), db.NewOAuthStorePG(dbpg), &mailStub{})
	podController, err := podcast.NewPodController(db.NewPodcastStore(dbpg))
	if err != nil {
		log.Fatalf("twirp.TestMain() error setting up PodController: %v", err)
	}
	rssController := podcast.NewRSSController(podController)

	twirpServer := NewServer(authController,
		NewAuthService(authController), NewPodcastService(podController),
		NewAdminService(podController, rssController),
	)

	go func() {
		err := twirpServer.Start()
		if err != nil {
			log.Fatalf("Twirp server failed to start: %v", err)
		}
	}()

	// run tests
	runCode := m.Run()

	// close pgx pool
	dbpg.Close()

	// cleanup docker container
	err = dockerCleanFunc()
	if err != nil {
		log.Fatalf("twirp.TestMain() error cleaning up docker container: %v", err)
	}

	os.Exit(runCode)
}

func setupAuthDB() error {
	insertUser(testUser)

	return nil
}

func TestAuthRpc(t *testing.T) {
	// setup auth client
	client := protos.NewAuthProtobufClient(
		"http://localhost:8081",
		http.DefaultClient,
		twirp.WithClientPathPrefix(prefix),
	)

	// CreateAccount
	testCreateUser := &protos.CreateAccountReq{
		Username:    "testCreate",
		Email:       "testCreate@syncapod.com",
		Password:    "myPassIsLongEnough",
		DateOfBirth: 1010223356,
		AcceptTerms: true,
	}
	_, err := client.CreateAccount(context.Background(), testCreateUser)
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	// retrieve activation token
	row := dbpg.QueryRow(context.Background(), "SELECT id FROM Users WHERE email=$1", testCreateUser.Email)
	if err != nil {
		t.Fatalf("could not find test user account row: %v", err)
	}
	testCreateUserID := uuid.UUID{}
	err = row.Scan(&testCreateUserID)
	if err != nil {
		t.Fatalf("error scanning id: %v", err)
	}

	row = dbpg.QueryRow(context.Background(), "SELECT token FROM Activation WHERE user_id=$1", testCreateUserID)
	testActivationCode := ""
	err = row.Scan(&testActivationCode)
	if err != nil {
		t.Fatalf("error scanning token: %v", err)
	}
	// Activate account
	_, err = client.Activate(context.Background(), &protos.ActivateReq{Token: testActivationCode})
	if err != nil {
		t.Fatal(err)
	}

	autheRes, err := client.Authenticate(context.Background(),
		&protos.AuthenticateReq{Username: testUser.Username, Password: "password"},
	)
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}
	require.NotEmpty(t, autheRes.SessionKey)
	seshKey := autheRes.SessionKey
	log.Println("got session key:", seshKey)

	//	Authorization
	authoRes, err := client.Authorize(context.Background(),
		&protos.AuthorizeReq{SessionKey: seshKey},
	)
	if err != nil {
		t.Fatalf("Authorize failed: %v", err)
	}
	require.NotEmpty(t, authoRes.User)
	log.Println("authorized user:", authoRes.User)

	header := make(http.Header)
	header.Add(authTokenKey, seshKey)
	ctx, err := twirp.WithHTTPRequestHeaders(context.Background(), header)
	if err != nil {
		t.Fatalf("Failed to add header to context: %v", err)
	}

	// Logout
	logoutRes, err := client.Logout(ctx, &protos.LogoutReq{SessionKey: seshKey})
	if err != nil {
		t.Fatalf("Logout failed: %v", err)
	}
	require.Equal(t, true, logoutRes.Success)
}

func TestAuthService_CreateAccount(t *testing.T) {
	client := protos.NewAuthProtobufClient(
		"http://localhost:8081",
		http.DefaultClient,
		twirp.WithClientPathPrefix(prefix),
	)

	type fields struct {
		client protos.Auth
	}
	type args struct {
		ctx context.Context
		req *protos.CreateAccountReq
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *protos.CreateAccountRes
		wantErr bool
	}{
		{
			name:   "success",
			fields: fields{client: client},
			args: args{
				ctx: context.Background(),
				req: &protos.CreateAccountReq{
					Username:    "TestCreateUser",
					Email:       "TestCreateUser@syncapod.com",
					Password:    "TheSecretPassword",
					DateOfBirth: 977908467,
					AcceptTerms: true,
				},
			},
			want:    &protos.CreateAccountRes{},
			wantErr: false,
		},
		{
			name:   "duplicate username",
			fields: fields{client: client},
			args: args{
				ctx: context.Background(),
				req: &protos.CreateAccountReq{
					Username:    testUser.Username,
					Email:       "TestCreateUser2@syncapod.com",
					Password:    "TheSecretPassword",
					DateOfBirth: 977908467,
					AcceptTerms: true,
				},
			},
			want:    &protos.CreateAccountRes{},
			wantErr: true,
		},
		{
			name:   "duplicate email",
			fields: fields{client: client},
			args: args{
				ctx: context.Background(),
				req: &protos.CreateAccountReq{
					Username:    "TestCreateUser2",
					Email:       testUser.Email,
					Password:    "TheSecretPassword",
					DateOfBirth: 977908467,
					AcceptTerms: true,
				},
			},
			want:    &protos.CreateAccountRes{},
			wantErr: true,
		},
		{
			name:   "break terms",
			fields: fields{client: client},
			args: args{
				ctx: context.Background(),
				req: &protos.CreateAccountReq{
					Username:    "TestCreateUser3",
					Email:       "TestCreateUser3@syncapod.com",
					Password:    "TheSecretPassword",
					DateOfBirth: 977908467,
					AcceptTerms: false,
				},
			},
			want:    &protos.CreateAccountRes{},
			wantErr: true,
		},
		{
			name:   "age restriction",
			fields: fields{client: client},
			args: args{
				ctx: context.Background(),
				req: &protos.CreateAccountReq{
					Username:    "TestCreateUser4",
					Email:       "TestCreateUser4@syncapod.com",
					Password:    "TheSecretPassword",
					DateOfBirth: time.Now().Unix() - (536112000),
					AcceptTerms: true,
				},
			},
			want:    &protos.CreateAccountRes{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.fields.client.CreateAccount(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthService.CreateAccount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

// TestAuthService_Activate tests all invalid ways to hit activate endpoint
func TestAuthService_Activate(t *testing.T) {
	client := protos.NewAuthProtobufClient(
		"http://localhost:8081",
		http.DefaultClient,
		twirp.WithClientPathPrefix(prefix),
	)
	type args struct {
		req *protos.ActivateReq
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid uuid format",
			args:  args{req:  &protos.ActivateReq{Token: "asdf"}},
			wantErr: true,
		},
		{
			name: "invalid uuid",
			args:  args{req:  &protos.ActivateReq{Token: uuid.New().String()}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Activate(context.Background(), tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthService.Activate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
