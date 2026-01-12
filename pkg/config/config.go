package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type DBConfig struct {
	Type         string `yaml:"type" json:"type"`
	Host         string `yaml:"host" json:"host"`
	Port         int    `yaml:"port" json:"port"`
	Username     string `yaml:"username" json:"username"`
	Password     string `yaml:"password" json:"password"`
	DatabaseName string `yaml:"database_name" json:"database_name"`
	DSN          string `yaml:"dsn" json:"dsn"` // optional explicit DSN
}

type ServerConfig struct {
	Port int `yaml:"port" json:"port"`
}

type AppConfig struct {
	Database DBConfig     `yaml:"database" json:"database"`
	Server   ServerConfig `yaml:"server" json:"server"`
}

// LoadFile loads YAML config from path.
func LoadFile(path string) (AppConfig, error) {
	var cfg AppConfig
	f, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(f, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// NormalizeDriver maps common aliases to canonical keys (keeps backwards compat).
func NormalizeDriver(d string) string {
	switch strings.ToLower(strings.TrimSpace(d)) {
	case "postgresql", "pg", "postgres":
		return "postgres"
	case "mysql", "mariadb":
		return "mysql"
	case "sqlite", "sqlite3":
		return "sqlite"
	case "mssql", "sqlserver":
		return "sqlserver"
	case "godror", "oracle":
		return "godror"
	default:
		return strings.ToLower(d)
	}
}

// BuildDriverAndDSN produces a driver name and DSN string for supported DB types.
func BuildDriverAndDSN(db DBConfig) (driver string, dsn string, err error) {
	// If explicit DSN provided, user must also set Type to choose driver or we guess
	t := NormalizeDriver(db.Type)

	if db.DSN != "" {
		return t, db.DSN, nil
	}

	switch t {
	case "postgres":
		driver = "postgres"
		// simple URL form
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			db.Username, db.Password, db.Host, db.Port, db.DatabaseName)
	case "mysql":
		driver = "mysql"
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
			db.Username, db.Password, db.Host, db.Port, db.DatabaseName)
	case "sqlite":
		driver = "sqlite"
		if db.DatabaseName == "" {
			return "", "", fmt.Errorf("sqlite needs a file path in database_name")
		}
		dsn = fmt.Sprintf("file:%s?mode=ro", db.DatabaseName)
	case "sqlserver":
		driver = "sqlserver"
		dsn = fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s",
			db.Username, db.Password, db.Host, db.Port, db.DatabaseName)
	case "godror":
		driver = "godror"
		// simple EZCONNECT style; may need adjustments per environment
		dsn = fmt.Sprintf("%s/%s@%s:%d/%s",
			db.Username, db.Password, db.Host, db.Port, db.DatabaseName)
	default:
		err = fmt.Errorf("unsupported database type: %s", db.Type)
	}
	return
}
