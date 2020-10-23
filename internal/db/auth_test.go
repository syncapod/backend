package db

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/ory/dockertest"
)

var db *pgxpool.Pool

// user TestMain to setup
func TestMain(m *testing.M) {
	// connect to docker pool
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %v", err)
	}

	// startup docker container
	resource, err := pool.Run("postgres", "", []string{"POSTGRES_PASSWORD=secret"})
	if err != nil {
		log.Fatalf("Could not start docker resource: %v", err)
	}

	// connect stop after 5 seconds
	start := time.Now()
	fiveSec := time.Second * 5
	port := resource.GetPort("5432/tcp")
	log.Println("port connected:", port)
	err = errors.New("start loop")
	for err != nil {
		if time.Since(start) > fiveSec {
			log.Fatal(`Could not connect to postgres\n
				Took longer than 5 seconds, maybe download postgres image`)
		}
		db, err = pgxpool.Connect(context.Background(),
			fmt.Sprintf(
				"postgres://postgres:secret@localhost:%s/postgres?sslmode=disable",
				port,
			),
		)
		time.Sleep(time.Millisecond * 250)
	}

	// run migrations up
	migrateUp()

	// setup db
	setupAuthDB()

	// run tests
	runCode := m.Run()

	// cleanup
	err = pool.Purge(resource)
	if err != nil {
		log.Fatalf("Could not purge resource: %v", err)
	}

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
			args:    args{ctx: context.Background(), id: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba")},
			fields:  fields{db: db},
			want:    &UserRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Email: "get@test.test", Username: "get", Birthdate: time.Unix(0, 0).UTC(), PasswordHash: []byte("pass")},
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
			want:    &UserRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Email: "get@test.test", Username: "get", Birthdate: time.Unix(0, 0).UTC(), PasswordHash: []byte("pass")},
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
			want:    &UserRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Email: "get@test.test", Username: "get", Birthdate: time.Unix(0, 0).UTC(), PasswordHash: []byte("pass")},
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
				u:   &UserRow{ID: uuid.MustParse("b813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Email: "update@updated.test", Username: "updated", Birthdate: time.Unix(0, 0).UTC(), PasswordHash: []byte("pass")},
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
				id:            uuid.MustParse("c813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"),
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
				id:  uuid.MustParse("d813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"),
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
				s: &SessionRow{ID: uuid.MustParse("a113c6e3-9cd0-4aed-9c4e-1d87ae20c8ba"), UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"),
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
			want: &SessionRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d87ae20c8ba"), UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"),
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
				s: &SessionRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d87ae20c8bb"), UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"),
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

func migrateUp() {
	// get all migration(up) files
	files := getMigrateUpFiles()

	// files should be in order because the names start with the number
	log.Println("migration files:", files)
	for _, f := range files {
		sqlCmd, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatalf("could not read from file: %v", err)
		}
		c, err := db.Exec(context.Background(), string(sqlCmd))
		if err != nil {
			log.Fatalf("could not run migrate command: %s, error: %v", string(sqlCmd), err)
		}
		log.Println(c.String())
	}
}
func getMigrateUpFiles() []string {
	// get working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Could not get working directory: %v", err)
	}
	// find the syncapod-backend directory
	split := strings.SplitAfter(wd, "syncapod-backend")
	if len(split) != 2 {
		log.Fatalf("Could not find syncapod-backend directory: %v", err)
	}
	syncapodDir := split[0]
	// scan files
	upFiles := []string{}
	err = filepath.Walk(syncapodDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if strings.Contains(info.Name(), ".up.") && !strings.Contains(info.Name(), ".swp") {
			upFiles = append(upFiles, filepath.Join(syncapodDir, "migrations", info.Name()))
		}
		return nil
	})
	return upFiles
}

func setupAuthDB() {
	// test users
	getUser := &UserRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Email: "get@test.test", Username: "get", Birthdate: time.Unix(10000, 0).UTC(), PasswordHash: []byte("pass")}
	insertUser(getUser)
	updateUser := &UserRow{ID: uuid.MustParse("b813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Email: "update@test.test", Username: "update", Birthdate: time.Unix(10001, 0).UTC(), PasswordHash: []byte("pass")}
	insertUser(updateUser)
	updatePassUser := &UserRow{ID: uuid.MustParse("c813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Email: "updatePass@test.test", Username: "updatePass", Birthdate: time.Unix(10002, 0).UTC(), PasswordHash: []byte("pass")}
	insertUser(updatePassUser)
	deleteUser := &UserRow{ID: uuid.MustParse("d813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"), Email: "delete@test.test", Username: "delete", Birthdate: time.Unix(10002, 0).UTC(), PasswordHash: []byte("pass")}
	insertUser(deleteUser)

	// test sessions
	getSesh := &SessionRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d87ae20c8ba"), UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"),
		Expires: time.Unix(1000, 0), LastSeenTime: time.Unix(1000, 0), LoginTime: time.Unix(1000, 0), UserAgent: "testAgent"}
	insertSession(getSesh)
	updateSesh := &SessionRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d87ae20c8bb"), UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"),
		Expires: time.Unix(1000, 0), LastSeenTime: time.Unix(1000, 0), LoginTime: time.Unix(1000, 0), UserAgent: "testAgent"}
	insertSession(updateSesh)
	deleteSesh := &SessionRow{ID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d87ae20c8bc"), UserID: uuid.MustParse("a813c6e3-9cd0-4aed-9c4e-1d88ae20c8ba"),
		Expires: time.Unix(1000, 0), LastSeenTime: time.Unix(1000, 0), LoginTime: time.Unix(1000, 0), UserAgent: "testAgent"}
	insertSession(deleteSesh)
}

func insertUser(u *UserRow) {
	_, err := db.Exec(context.Background(),
		"INSERT INTO users (id,email,username,birthdate,password_hash) VALUES($1,$2,$3,$4,$5)",
		u.ID, u.Email, u.Username, u.Birthdate, u.PasswordHash)
	if err != nil {
		log.Fatalln("insertUser() error:", err)
	}
}

func insertSession(s *SessionRow) {
	_, err := db.Exec(context.Background(),
		"INSERT INTO sessions (id,user_id,login_time,last_seen_time,expires,user_agent) VALUES($1,$2,$3,$4,$5,$6)",
		s.ID, s.UserID, s.LoginTime, s.LastSeenTime, s.Expires, s.UserAgent)
	if err != nil {
		log.Fatalln("insertSession() error:", err)
	}
}
