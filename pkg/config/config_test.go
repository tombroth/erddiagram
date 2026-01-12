package config

import (
	"testing"
)

func TestLoadFile(t *testing.T) {
	var tests = []struct {
		name     string
		filename string
		config   AppConfig
		errIsNil bool
	}{
		{"Valid Config",
			"./testdata/valid_config.yaml",
			AppConfig{
				Database: DBConfig{
					Type:         "testType",
					Host:         "testHost",
					Port:         9999,
					Username:     "testUser",
					Password:     "testPass",
					DatabaseName: "testDb",
					DSN:          "",
				},
				Server: ServerConfig{
					Port: 8080,
				},
			},
			true},
		{"Invalid Config", "./testdata/invalid_config.yaml", AppConfig{}, false},
		{"File Not Found", ".testdata/no_such_file", AppConfig{}, false},
	}

	for _, tt := range tests {
		// Use t.Run to run each case as a subtest with a descriptive name
		t.Run(tt.name, func(t *testing.T) {
			c, err := LoadFile(tt.filename)
			if c != tt.config {
				t.Errorf("\ngot config %v, wanted %v ", c, tt.config)
			} else if (err == nil) != tt.errIsNil {
				if tt.errIsNil {
					t.Errorf("\ngot unexpected error: \"%v\"", err)
				} else {
					t.Errorf("\nexpected an error, did not receive one")
				}
			}
		})
	}
}

func TestNormalizeDriver(t *testing.T) {
	var tests = []struct {
		driverIn  string
		driverOut string
	}{
		{"postgresql", "postgres"},
		{"pg", "postgres"},
		{"postgres", "postgres"},
		{"mysql", "mysql"},
		{"mariadb", "mysql"},
		{"sqlite", "sqlite"},
		{"sqlite3", "sqlite"},
		{"mssql", "sqlserver"},
		{"sqlserver", "sqlserver"},
		{"godror", "godror"},
		{"oracle", "godror"},
		{"UNKNOWN", "unknown"},
	}

	for _, tt := range tests {
		// Use t.Run to run each case as a subtest with a descriptive name
		t.Run(tt.driverIn, func(t *testing.T) {
			driver := NormalizeDriver(tt.driverIn)
			if driver != tt.driverOut {
				t.Errorf("\ngot driver %v, wanted %v ", driver, tt.driverOut)
			}
		})
	}
}

func TestBuildDriverAndDSN(t *testing.T) {
	var tests = []struct {
		name     string
		db       DBConfig
		driver   string
		dsn      string
		errIsNil bool
	}{
		{"postgresql",
			DBConfig{"postgresql", "localhost", 5432, "testuser", "testpass", "testdb", ""},
			"postgres",
			"postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable",
			true},
		{"pg",
			DBConfig{"pg", "", 0, "", "", "", "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"},
			"postgres",
			"postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable",
			true},
		{"mariadb",
			DBConfig{"mariadb", "localhost", 3306, "testuser", "testpass", "testdb", ""},
			"mysql",
			"testuser:testpass@tcp(localhost:3306)/testdb?parseTime=true",
			true},
		{"sqlite3",
			DBConfig{"sqlite3", "", 0, "", "", "testdb", ""},
			"sqlite",
			"file:testdb?mode=ro",
			true},
		{"sqlite error",
			DBConfig{"sqlite3", "", 0, "", "", "", ""},
			"",
			"",
			false},
		{"mssql",
			DBConfig{"mssql", "localhost", 1433, "testuser", "testpass", "testdb", ""},
			"sqlserver",
			"sqlserver://testuser:testpass@localhost:1433?database=testdb",
			true},
		{"oracle",
			DBConfig{"oracle", "localhost", 1521, "testuser", "testpass", "testdb", ""},
			"godror",
			"testuser/testpass@localhost:1521/testdb",
			true},
		{"UNKNOWN",
			DBConfig{"UNKNOWN", "localhost", 9999, "testuser", "testpass", "testdb", ""},
			"",
			"",
			false},
	}

	for _, tt := range tests {
		// Use t.Run to run each case as a subtest with a descriptive name
		t.Run(tt.name, func(t *testing.T) {
			driver, dsn, err := BuildDriverAndDSN(tt.db)
			if driver != tt.driver {
				t.Errorf("\ngot driver %v, wanted %v ", driver, tt.driver)
			} else if dsn != tt.dsn {
				t.Errorf("\ngot dsn %v, wanted %v", dsn, tt.dsn)
			} else if (err == nil) != tt.errIsNil {
				if tt.errIsNil {
					t.Errorf("\ngot unexpected error: \"%v\"", err)
				} else {
					t.Errorf("\nexpected an error, did not receive one")
				}
			}
		})
	}
}
