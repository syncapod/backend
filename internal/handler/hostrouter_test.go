package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/matryer/is"
)

func TestHostRouter_SetHostRoute(t *testing.T) {
	type fields struct {
		hostRoutes map[string]http.Handler
	}
	type args struct {
		host    string
		handler http.Handler
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			fields: fields{
				hostRoutes: make(map[string]http.Handler),
			},
			args: args{
				host:    "test.syncapod.com",
				handler: nil,
			},
			wantErr: false,
		},
		{
			name: "error wildcard",
			fields: fields{
				hostRoutes: make(map[string]http.Handler),
			},
			args: args{
				host:    "*",
				handler: nil, // can be nil
			},
			wantErr: true,
		},
		{
			name: "invalid",
			fields: fields{
				hostRoutes: map[string]http.Handler{"test.syncapod.com": nil},
			},
			args: args{
				host:    "test.syncapod.com",
				handler: nil, // can be nil
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HostRouter{
				hostRoutes: tt.fields.hostRoutes,
			}
			if err := h.SetHostRoute(tt.args.host, tt.args.handler); (err != nil) != tt.wantErr {
				t.Errorf("HostRouter.SetHostRoute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHostRouter_Handler(t *testing.T) {
	is := is.New(t)

	// create and setup default router
	defaultHandler := chi.NewRouter()
	defaultHandler.Get("/", defaultRoute)

	// create and setup test.syncapod.com router
	testSyncapodHandler := chi.NewRouter()
	testSyncapodHandler.Get("/", syncapodTestRoute)

	// create out host router
	testHostRouter := NewHostRouter(defaultHandler)
	testHostRouter.SetHostRoute("test.syncapod.com", testSyncapodHandler)

	// mount host router on a new main handler
	// main handler will be used as our handler to serve traffic
	mainHandler := chi.NewRouter()
	mainHandler.Mount("/", testHostRouter.Handler())

	// test fallback
	defaultRecorder := httptest.NewRecorder()
	defaultRequest := httptest.NewRequest("GET", "http://syncapod.com", nil)
	mainHandler.ServeHTTP(defaultRecorder, defaultRequest)
	defaultBody, err := io.ReadAll(defaultRecorder.Body)
	is.NoErr(err)

	is.Equal(string(defaultBody), "default")

	// test test.syncapod.com
	testRecorder := httptest.NewRecorder()
	testRequest := httptest.NewRequest("GET", "http://test.syncapod.com", nil)
	mainHandler.ServeHTTP(testRecorder, testRequest)
	testBody, err := io.ReadAll(testRecorder.Body)
	is.NoErr(err)

	is.Equal(string(testBody), "test.syncapod.com route")
}

func defaultRoute(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("default"))
}

func syncapodTestRoute(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("test.syncapod.com route"))
}
