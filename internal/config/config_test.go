package config

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const mockConfig = `{
	"cert_dir": "/syncapod/cert",
	"port": 8090,
	"grpc_port": 50051,
	"grpc_gateway_port": 50052,
	"db_name": "syncapod",
	"db_user": "syncapod",
	"db_pass": "syncapod",
	"db_host": "localhost",
	"db_port": 5432,
	"production": false
}`

var mockConfigObj = &Config{
	CertDir:       "/syncapod/cert",
	Port:          8090,
	DbName:        "syncapod",
	DbUser:        "syncapod",
	DbPass:        "syncapod",
	DbHost:        "localhost",
	DbPort:        5432,
	Production:    false,
	MigrationsDir: "/syncapod/migrations",
}

func TestReadConfig(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr bool
	}{
		{name: "invalid", args: args{r: strings.NewReader("")}, want: nil, wantErr: true},
		{name: "valid", args: args{r: strings.NewReader(mockConfig)}, want: mockConfigObj, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadConfig(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}
