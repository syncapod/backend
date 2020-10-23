package config

import (
	"io"
	"reflect"
	"strings"
	"testing"
)

const mockConfig = `{
	"version": 0.123,
	"db_uri": "mongodb+srv://mockdburi",
	"port": 8090,
	"cert_file": "",
	"key_file": "",
	"grpc_port": 50051
}`

var mockConfigObj = &Config{Version: 0.123, DbURI: "mongodb+srv://mockdburi", Port: 8090, GRPCPort: 50051}

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
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
