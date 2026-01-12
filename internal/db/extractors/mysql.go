package extractors

import (
	"context"
	"database/sql"
	"fmt"

	"erddiagram/internal/db"
	"erddiagram/internal/introspect"
	"erddiagram/internal/logger"
)

// myExtractor implements Extractor for MySQL (information_schema).
type myExtractor struct{}

// This is the extractor for MySQL
func (myExtractor) Extract(ctx context.Context, dbConn *sql.DB) (introspect.Schema, error) {
	var s introspect.Schema

	tr, err := dbConn.QueryContext(ctx, `
        SELECT table_schema, table_name, table_comment, round(data_length/8192) AS size_8k_pages
        FROM information_schema.tables
        WHERE table_type = 'BASE TABLE'
          AND table_schema NOT IN ('mysql','information_schema','performance_schema','sys')
        ORDER BY table_schema, table_name`)
	if err != nil {
		return s, fmt.Errorf("query tables: %w", err)
	}
	defer tr.Close()

	for tr.Next() {
		var tab introspect.Table
		if err := tr.Scan(&tab.Schema, &tab.Name, &tab.Comment, &tab.Size8kPages); err != nil {
			return s, fmt.Errorf("scan table row: %w", err)
		}
		s.Tables = append(s.Tables, tab)
	}

	for i := range s.Tables {
		t := &s.Tables[i]
		cr, err := dbConn.QueryContext(ctx, `
            SELECT column_name, column_type, is_nullable = 'YES'
            FROM information_schema.columns
            WHERE table_schema = ? AND table_name = ?
            ORDER BY ordinal_position`, t.Schema, t.Name)
		if err != nil {
			return s, fmt.Errorf("query columns for %s.%s: %w", t.Schema, t.Name, err)
		}
		for cr.Next() {
			var col introspect.Column
			if err := cr.Scan(&col.Name, &col.Type, &col.Nullable); err != nil {
				cr.Close()
				return s, fmt.Errorf("scan column for %s.%s: %w", t.Schema, t.Name, err)
			}
			t.Columns = append(t.Columns, col)
		}
		cr.Close()

		pkr, err := dbConn.QueryContext(ctx, `
            SELECT k.COLUMN_NAME
            FROM information_schema.key_column_usage k
            JOIN information_schema.table_constraints tc ON k.constraint_name = tc.constraint_name AND k.table_schema = tc.table_schema
            WHERE tc.constraint_type = 'PRIMARY KEY' AND k.table_schema = ? AND k.table_name = ?`, t.Schema, t.Name)
		if err == nil {
			for pkr.Next() {
				var pkcol string
				if err := pkr.Scan(&pkcol); err == nil {
					for j := range t.Columns {
						if t.Columns[j].Name == pkcol {
							t.Columns[j].PK = true
						}
					}
				} else {
					logger.Error("scan primary key: %v", err)
				}
			}
			pkr.Close()
		} else {
			logger.Error("query primary key: %v", err)
		}
	}

	fkr, err := dbConn.QueryContext(ctx, `
        SELECT table_schema AS from_schema, table_name AS from_table, 
		       group_concat(column_name separator ', ') AS from_column,
               referenced_table_schema AS to_schema, referenced_table_name AS to_table, 
			   group_concat(referenced_column_name separator ', ') AS to_column,
			   constraint_name
        FROM information_schema.key_column_usage
        WHERE referenced_table_name IS NOT NULL AND table_schema NOT IN ('mysql','information_schema','performance_schema','sys')
		GROUP BY table_schema, table_name, referenced_table_schema, referenced_table_name, constraint_name`)
	if err == nil {
		defer fkr.Close()
		for fkr.Next() {
			var fk introspect.ForeignKey
			if err := fkr.Scan(&fk.FromSchema, &fk.FromTable, &fk.FromColumn, &fk.ToSchema, &fk.ToTable, &fk.ToColumn, &fk.Constraint); err == nil {
				s.ForeignKeys = append(s.ForeignKeys, fk)
			} else {
				logger.Error("scan foreign key: %v", err)
			}
		}
	} else {
		logger.Error("query foreign key: %v", err)
	}

	return s, nil
}

func init() {
	db.Register("mysql", myExtractor{})
	db.Register("mariadb", myExtractor{})
}
