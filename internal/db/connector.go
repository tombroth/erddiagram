package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"

	"erddiagram/internal/introspect"
	"erddiagram/pkg/config"
)

type Extractor interface {

	// Extract takes a database connection and returns data for the ERD
	Extract(ctx context.Context, db *sql.DB) (introspect.Schema, error)
}

var dialects = map[string]Extractor{}

// Register makes an Extractor available under name.
func Register(name string, e Extractor) {
	dialects[strings.ToLower(name)] = e
}

// listRegistered returns the registered dialect keys (for diagnostics).
func listRegistered() []string {
	keys := make([]string, 0, len(dialects))
	for k := range dialects {
		keys = append(keys, k)
	}
	return keys
}

// ConnectAndExtract connects to the database and extracts information for the ERD
func ConnectAndExtract(driver, dsn string, timeoutSec int) (introspect.Schema, error) {
	driver = config.NormalizeDriver(driver)
	extractor, ok := dialects[driver]
	if !ok {
		return introspect.Schema{}, fmt.Errorf("dialect not registered: %q (available: %v)", driver, listRegistered())
	}
	dbConn, err := sql.Open(config.NormalizeDriver(driver), dsn)
	if err != nil {
		return introspect.Schema{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()
	if err := dbConn.PingContext(ctx); err != nil {
		return introspect.Schema{}, err
	}
	return extractor.Extract(ctx, dbConn)
}

// RegisteredDialects is a helper that allows main to print registered dialects
func RegisteredDialects() []string {
	return listRegistered()
}
