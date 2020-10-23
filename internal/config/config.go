package config

import (
	"encoding/json"
	"fmt"
	"io"
)

// Config holds variables for our server
type Config struct {
	Version       float64 `json:"version,omitempty"`
	DbURI         string  `json:"db_uri,omitempty"`
	DbUser        string  `json:"db_user,omitempty"`
	DbPass        string  `json:"db_pass,omitempty"`
	Port          int     `json:"port"`
	CertFile      string  `json:"cert_file"`
	KeyFile       string  `json:"key_file"`
	AlexaClientID string  `json:"alexa_client_id"`
	AlexaSecret   string  `json:"alexa_secret"`
	GRPCPort      int     `json:"grpc_port"`
}

// ReadConfig reads the config file encoded in JSON
func ReadConfig(r io.Reader) (*Config, error) {
	// Unmarshal into config var
	var config Config
	err := json.NewDecoder(r).Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("ReadConfig() error decoding config: %v", err)
	}
	return &config, nil
}
