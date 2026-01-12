package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"erddiagram/internal/introspect"
)

var testdialect string = "testdialect"

type testExtractor struct{}

func (testExtractor) Extract(ctx context.Context, dbConn *sql.DB) (introspect.Schema, error) {
	var s introspect.Schema
	return s, errors.New("not implemented")
}

func TestRegister(t *testing.T) {
	// tests both Register and RegisteredDialects because they take the same setup

	Register(testdialect, testExtractor{})

	if _, ok := dialects[testdialect]; !ok {
		t.Errorf("\ndialect %v not registered correctly in %v", testdialect, dialects)
	}

	rd := RegisteredDialects()

	if !(len(rd) == 1 && rd[0] == testdialect) {
		t.Errorf("\nRegisteredDialects returned unexpected result %v", rd)
	}
}

func TestConnectAndExtract(t *testing.T) {

	var tests = []struct {
		name          string
		dialect       string
		dsn           string
		timeout       int
		registerFirst bool
		errIsNil      bool
	}{
		{"unregistered dialect", testdialect, "", 10, false, false},
		{"sqlite with testExtractor", "sqlite", ":memory:", 10, true, false},
	}

	for _, tt := range tests {
		// Use t.Run to run each case as a subtest with a descriptive name
		t.Run(tt.name, func(t *testing.T) {
			if tt.registerFirst {
				Register(tt.dialect, testExtractor{})
			}

			_, err := ConnectAndExtract(tt.dialect, tt.dsn, tt.timeout)

			if (err == nil) != tt.errIsNil {
				if tt.errIsNil {
					t.Errorf("\ngot unexpected error: \"%v\"", err)
				} else {
					t.Errorf("\nexpected an error, did not receive one")
				}
			}
		})
	}
}
