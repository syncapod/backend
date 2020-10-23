package auth

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/sschwartz96/stockpile/db"
	"github.com/sschwartz96/stockpile/mock"
	"github.com/sschwartz96/syncapod/internal/database"
	"github.com/sschwartz96/syncapod/internal/protos"
	"github.com/sschwartz96/syncapod/internal/util"
	"golang.org/x/crypto/bcrypt"
)

func TestHash(t *testing.T) {
	type args struct {
		password string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "one",
			args: args{password: "password"},
		},
		{
			name: "two",
			args: args{password: "simplePhraseButVeryLongString"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Hash(tt.args.password)
			if err != nil {
				t.Errorf("Hash() error = %v", err)
			}
			if err = bcrypt.CompareHashAndPassword([]byte(got), []byte(tt.args.password)); err != nil {
				t.Errorf("Hash() error = %v, did not match hash", err)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	type args struct {
		hash     string
		password string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "correct",
			args: args{password: "password"},
			want: true,
		},
		{
			name: "wrong",
			args: args{password: "simplePhraseButVeryLongString"},
			want: false,
		},
	}

	// generate hashes
	for i := range tests {
		var err error
		tests[i].args.hash, err = Hash(tests[i].args.password)
		if !tests[i].want {
			// change password to make sure it doesn't match
			tests[i].args.password = strings.ToLower(tests[i].args.password)
		}
		if err != nil {
			t.Errorf("TestHash() setup, could not hash password: %v", err)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Compare(tt.args.hash, tt.args.password); got != tt.want {
				t.Errorf("Compare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateSession(t *testing.T) {
	mockDB := mock.CreateDB()
	type args struct {
		dbClient     db.Database
		userID       *protos.ObjectID
		userAgent    string
		stayLoggedIn bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test1",
			args: args{
				dbClient:     mockDB,
				stayLoggedIn: false,
				userAgent:    "",
				userID:       protos.ObjectIDFromHex("userID1"),
			},
			wantErr: false,
		},
		{
			name: "test2",
			args: args{
				dbClient:     mockDB,
				stayLoggedIn: true,
				userAgent:    "testAgent",
				userID:       protos.ObjectIDFromHex("userID2"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateSession(tt.args.dbClient, tt.args.userID, tt.args.userAgent, tt.args.stayLoggedIn)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSession() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			var sesh protos.Session
			err = tt.args.dbClient.FindOne(database.ColSession, &sesh, &db.Filter{"userid": tt.args.userID}, db.CreateOptions())
			if err != nil {
				t.Errorf("CreateSession() error = %v, wantErr %v", err, tt.wantErr)
			}
			if sesh.SessionKey != got {
				t.Errorf("CreateSession() error = keys do not match! Found %v, wanted %v", sesh.SessionKey, got)
			}
		})
	}
}

func TestCreateKey(t *testing.T) {
	type args struct {
		l int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test1",
			args: args{
				l: 64,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateKey(tt.args.l)
			if err != nil {
				t.Errorf("CreateKey() error = %v", err)
			}
			if len(got) != tt.args.l {
				t.Errorf("CreateKey() error = didn't get correct length")
			}
		})
	}
}

func insertOrFail(t *testing.T, db db.Database, col string, object interface{}) {
	err := db.Insert(col, object)
	if err != nil {
		t.Fatalf("error inserting object into: %s, %v", col, err)
	}
}

func TestValidateSession(t *testing.T) {
	mockDB := mock.CreateDB()
	user := &protos.User{Id: protos.NewObjectID(), Email: "test@test.org"}
	insertOrFail(t, mockDB, database.ColUser, user)
	testSesh1, _ := CreateSession(mockDB, user.Id, "testAgent", true)
	testSesh2 := &protos.Session{Id: protos.NewObjectID(), SessionKey: "key", Expires: util.AddToTimestamp(ptypes.TimestampNow(), time.Minute*-1), UserID: user.Id}
	insertOrFail(t, mockDB, database.ColSession, testSesh2)

	type args struct {
		dbClient db.Database
		key      string
	}
	tests := []struct {
		name    string
		args    args
		want    *protos.User
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				dbClient: mockDB,
				key:      testSesh1,
			},
			want:    user,
			wantErr: false,
		},
		{
			name: "invalid",
			args: args{
				dbClient: mockDB,
				key:      testSesh2.SessionKey,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateSession(tt.args.dbClient, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSession() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidateSession() = %v, want %v", got, tt.want)
			}
		})
	}
}
