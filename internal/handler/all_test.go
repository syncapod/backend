package handler

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sschwartz96/syncapod-backend/internal"
	"github.com/sschwartz96/syncapod-backend/internal/auth"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	"github.com/stretchr/testify/require"
)

var testHandler *Handler

func TestMain(t *testing.M) {
	// connect to db
	var pgdb *pgxpool.Pool
	var dockerCleanFunc func() error
	var err error
	pgdb, dockerCleanFunc, err = internal.StartDockerDB("db_auth")
	if err != nil {
		log.Fatalf("auth.TestMain() error setting up docker db: %v", err)
	}

	// create controllers
	authC := auth.NewAuthController(db.NewAuthStorePG(pgdb), db.NewOAuthStorePG(pgdb))

	// create handlers
	oauthHandler, err := createTestOAuthHandler(authC)
	if err != nil {
		log.Fatalf("Handler.TestMain() error creating oauthHandler: %v", err)
	}
	testHandler = &Handler{oauthHandler: oauthHandler}

	// setup database
	setup(pgdb)

	// run tests
	runCode := t.Run()

	// cleanup docker container
	err = dockerCleanFunc()
	if err != nil {
		log.Fatalf("grpc.TestMain() error cleaning up docker container: %v", err)
	}

	os.Exit(runCode)
}

func Test_Oauth(t *testing.T) {
	// oauth/login GET
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "https://syncapod.com/oauth/login", nil)
	testHandler.oauthHandler.Login(rec, req)
	body, err := ioutil.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("Test_Oauth() GET login error: %v", err)
	}
	require.Contains(t, string(body), "<h1>syncapod oauth2.0 login</h1>")

	// oauth/login POST
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "https://syncapod.com/oauth/login", nil)
	req.Form = url.Values{"uname": {"oauthTest"}, "pass": {"password"}, "redirect_uri": {"https://testuri.com"}}
	testHandler.oauthHandler.Login(rec, req)
	body, err = ioutil.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("Test_Oauth() POST login error: %v", err)
	}
	bodyString := string(body)
	require.Contains(t, string(body), "<a href=\"/oauth/authorize?")
	uri := "https://syncapod.com/" + strings.ReplaceAll(bodyString[10:115], "&amp;", "&")

	// oauth/authorize GET
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", uri, nil)
	testHandler.oauthHandler.Authorize(rec, req)
	body, err = ioutil.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("Test_Oauth() GET authorize error: %v", err)
	}
	require.Contains(t, string(body), "<title>syncapod oauth authorization</title>")

	// oauth/authorize POST
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("POST", uri, nil)
	testHandler.oauthHandler.Authorize(rec, req)
	res := rec.Result()
	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("Test_Oauth() POST authorize error: %v", err)
	}
	require.Equal(t, 303, res.StatusCode)
	loc, err := res.Location()
	if err != nil {
		t.Fatal("Test_Oauth() POST authorize location error")
	}
	authCode := loc.Query().Get("code")
	require.NotEmpty(t, authCode)

	// oauth/token auth code
	rToken := testOauthToken(t, map[string]string{"grant_type": "authorization_code", "code": authCode})

	// oauth/token refresh token
	testOauthToken(t, map[string]string{"grant_type": "refresh_token", "refresh_token": rToken})
}

func testOauthToken(t *testing.T, urlValues map[string]string) string {
	rec := httptest.NewRecorder()
	vals := url.Values{}
	for k, v := range urlValues {
		vals.Set(k, v)
	}
	req := httptest.NewRequest("POST", "https://syncapod.com/oauth/token", strings.NewReader(vals.Encode()))
	req.SetBasicAuth("testClientID", "testClientSecret")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(vals.Encode())))
	testHandler.oauthHandler.Token(rec, req)

	res := rec.Result()
	require.Equal(t, 200, res.StatusCode)

	tRes := struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}{}
	jsonDecoder := json.NewDecoder(res.Body)
	err := jsonDecoder.Decode(&tRes)
	if err != nil {
		t.Fatalf("Test_Oauth() POST token: error decoding json response: %v", err)
	}
	require.NotEmpty(t, tRes.AccessToken)
	require.NotEmpty(t, tRes.RefreshToken)
	return tRes.RefreshToken
}

//func Test_HTTP(t *testing.T) {
//	type args struct {
//		method string
//		url    string
//	}
//	tests := []struct {
//		name         string
//		args         args
//		resultCode   int
//		bodyContains string
//	}{}
//}

func createTestOAuthHandler(authC auth.Auth) (*OauthHandler, error) {
	loginT, err := template.ParseFiles("../../templates/oauth/login.gohtml")
	if err != nil {
		return nil, err
	}
	authT, err := template.ParseFiles("../../templates/oauth/auth.gohtml")
	if err != nil {
		return nil, err
	}
	return &OauthHandler{authC, loginT, authT, map[string]string{"testClientID": "testClientSecret"}}, nil
}

func setup(pg *pgxpool.Pool) {
	a := db.NewAuthStorePG(pg)
	insertUser(a, &db.UserRow{ID: uuid.MustParse("b7f85a20-9b8f-47f9-8cee-a553a24f2b6d"),
		Birthdate: time.Unix(0, 0), Email: "oauthTest@test.com", Username: "oauthTest",
		PasswordHash: []byte("$2a$10$bAkGU1SFc.oy9jz5/psXweSCqWG6reZr3Tl3oTKAgzBksPKHLG4bS"),
		Created:      time.Unix(0, 0), LastSeen: time.Unix(0, 0)})
}

func insertUser(a *db.AuthStorePG, u *db.UserRow) {
	err := a.InsertUser(context.Background(), u)
	if err != nil {
		log.Println("insertUser() id:", u.ID)
		log.Fatalln("insertUser() error:", err)
	}
}
