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
	templatesDirDefault  = "./templates/"
	portDefault          = 3030
	hostDefault          = "syncapod.com"
)

// Config holds variables for our server
type Config struct {
	DbUser          string `json:"db_user,omitempty"` // env:PG_USER
	DbPass          string `json:"db_pass,omitempty"` // env:PG_PASS
	DbHost          string `json:"db_host,omitempty"`
	DbPort          int    `json:"db_port,omitempty"` // env:PG_PORT
	DbName          string `json:"db_name,omitempty"` // env:PG_DB_NAME
	MigrationsDir   string `json:"migrations_dir,omitempty"`
	TemplatesDir    string `json:"templates_dir,omitempty"`
	Host            string `json:"host,omitempty"`
	Port            int    `json:"port,omitempty"`
	AlexaClientID   string `json:"alexa_client_id,omitempty"`
	AlexaSecret     string `json:"alexa_secret,omitempty"`
	ActionsClientID string `json:"actions_client_id,omitempty"`
	ActionsSecret   string `json:"actions_secret,omitempty"`
	Production      bool   `json:"production,omitempty"`
	CertDir         string `json:"cert_dir,omitempty"` // only used if production=true
	Debug           bool   `json:"debug,omitempty"`
}

// ReadConfig reads the config file encoded in JSON
func ReadConfig(r io.Reader) (*Config, error) {
	config := &Config{
		DbUser:        dbuserDefault,
		DbPort:        dbportDefault,
		DbName:        dbnameDefault,
		DbHost:        dbhostDefault,
		MigrationsDir: migrationsDirDefault,
		TemplatesDir:  templatesDirDefault,
		Port:          portDefault,
		Host:          hostDefault,
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
