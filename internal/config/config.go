package config

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
)

// Defaults
const (
	dbuserDefault        = "syncapod"
	dbportDefault        = 5432
	dbnameDefault        = "syncapod"
	dbhostDefault        = "localhost"
	migrationsDirDefault = "/syncapod/migrations"
	portDefault          = 3030
	grpcPortDefault      = 50051
)

// Config holds variables for our server
type Config struct {
	DbUser          string `json:"db_user,omitempty"` // env:PG_USER
	DbPass          string `json:"db_pass,omitempty"` // env:PG_PASS
	DbHost          string `json:"db_host"`
	DbPort          int    `json:"db_port"` // env:PG_PORT
	DbName          string `json:"db_name"` // env:PG_DB_NAME
	MigrationsDir   string `json:"migrations_dir"`
	Port            int    `json:"port"`
	AlexaClientID   string `json:"alexa_client_id"`
	AlexaSecret     string `json:"alexa_secret"`
	ActionsClientID string `json:"actions_client_id"`
	ActionsSecret   string `json:"actions_secret"`
	GRPCPort        int    `json:"grpc_port"`
	Production      bool   `json:"production"`
	CertDir         string `json:"cert_dir"` // only used if production=true
}

// ReadConfig reads the config file encoded in JSON
func ReadConfig(r io.Reader) (*Config, error) {
	config := &Config{
		DbUser:        dbuserDefault,
		DbPort:        dbportDefault,
		DbName:        dbnameDefault,
		DbHost:        dbhostDefault,
		MigrationsDir: migrationsDirDefault,
		Port:          portDefault,
		GRPCPort:      grpcPortDefault,
	}
	// Unmarshal into config var
	err := json.NewDecoder(r).Decode(config)
	if err != nil {
		return nil, fmt.Errorf("ReadConfig() error decoding config: %v", err)
	}
	readEnv(config)
	return config, nil
}

func readEnv(cfg *Config) {
	dbUser := os.Getenv("PG_USER")
	if len(dbUser) > 0 {
		cfg.DbUser = dbUser
	}
	dbPass := os.Getenv("PG_PASS")
	if len(dbPass) > 0 {
		cfg.DbPass = dbPass
	}
	dbPortString := os.Getenv("PG_PORT")
	if len(dbPortString) > 0 {
		dbPort, err := strconv.Atoi(dbPortString)
		if err != nil {
			log.Println("readEnv() error: PG_PORT not valid integer")
		}
		cfg.DbPort = dbPort
	}
	dbName := os.Getenv("PG_DB_NAME")
	if len(dbName) > 0 {
		cfg.DbName = dbName
	}
}
