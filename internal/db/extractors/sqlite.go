package extractors

import (
	"context"
	"database/sql"
	"fmt"

	"erddiagram/internal/db"
	"erddiagram/internal/introspect"
	"erddiagram/internal/logger"
)

// sqliteExtractor implements Extractor for SQLite.
type sqliteExtractor struct{}

// This is the extractor for SQLite
func (sqliteExtractor) Extract(ctx context.Context, dbConn *sql.DB) (introspect.Schema, error) {
	var s introspect.Schema
	dbName := "main"

	if rows, err := dbConn.QueryContext(ctx, `PRAGMA database_list`); err == nil {
		defer rows.Close()
		var seq int
		var name, file sql.NullString
		if rows.Next() {
			if err := rows.Scan(&seq, &name, &file); err == nil && name.Valid {
				dbName = name.String
			}
		}
	} else {
		logger.Error("database list: %v", err)
	}

	trQuery := `
	    SELECT m.name, s.size_8k_pages
		FROM sqlite_master m
		LEFT JOIN (SELECT name, CAST(CEIL(SUM(pgsize) / 8192.0) AS integer) AS size_8k_pages
				FROM dbstat
				GROUP BY name) s ON m.name = s.name
		WHERE m.type='table' 
		AND m.name NOT LIKE 'sqlite_%' 
		ORDER BY m.name`
	//fmt.Sprintf("SELECT name FROM %s.sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%%' ORDER BY name", dbName)
	tr, err := dbConn.QueryContext(ctx, trQuery)
	if err != nil {
		return s, fmt.Errorf("query tables: %w", err)
	}
	defer tr.Close()

	for tr.Next() {
		var tab introspect.Table
		if err := tr.Scan(&tab.Name, &tab.Size8kPages); err != nil {
			return s, fmt.Errorf("scan table row: %w", err)
		}
		s.Tables = append(s.Tables, tab)
	}

	for i := range s.Tables {
		t := &s.Tables[i]
		tiQuery := fmt.Sprintf("PRAGMA %s.table_info('%s')", dbName, t.Name)
		pr, err := dbConn.QueryContext(ctx, tiQuery)
		if err != nil {
			return s, fmt.Errorf("query columns for %s.%s: %w", t.Schema, t.Name, err)
		}
		for pr.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dflt sql.NullString
			if err := pr.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
				pr.Close()
				return s, fmt.Errorf("scan column for %s.%s: %w", t.Schema, t.Name, err)
			}
			col := introspect.Column{
				Name:     name,
				Type:     ctype,
				Nullable: notnull == 0,
				PK:       pk != 0,
			}
			t.Columns = append(t.Columns, col)
		}
		pr.Close()

		fkQuery := fmt.Sprintf(`
		    SELECT "table", string_agg("from", ', ') AS from_column, string_agg("to", ', ') AS to_column
		    FROM pragma_foreign_key_list('%s') 
			GROUP BY "table"`, t.Name)
		fkRows, err := dbConn.QueryContext(ctx, fkQuery)

		if err == nil {
			for fkRows.Next() {
				var table, from, to sql.NullString
				if err := fkRows.Scan(&table, &from, &to); err == nil {
					if table.Valid && from.Valid && to.Valid {
						s.ForeignKeys = append(s.ForeignKeys, introspect.ForeignKey{
							FromTable:  t.Name,
							FromColumn: from.String,
							ToTable:    table.String,
							ToColumn:   to.String,
						})
					}
				} else {
					logger.Error("scan foreign key: %v", err)
				}
			}
			fkRows.Close()
		} else {
			logger.Error("query foreign key: %v", err)
		}
	}

	return s, nil
}

func init() {
	db.Register("sqlite3", sqliteExtractor{})
	db.Register("sqlite", sqliteExtractor{})
}
