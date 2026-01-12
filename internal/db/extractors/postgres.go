package extractors

import (
	"context"
	"database/sql"
	"fmt"

	"erddiagram/internal/db"
	"erddiagram/internal/introspect"
	"erddiagram/internal/logger"
)

// pgExtractor implements Extractor using information_schema + pg_catalog queries.
type pgExtractor struct{}

// This is the extractor for PostgreSQL
func (pgExtractor) Extract(ctx context.Context, dbConn *sql.DB) (introspect.Schema, error) {
	var s introspect.Schema

	tr, err := dbConn.QueryContext(ctx, `
        SELECT table_schema, table_name, 
		       obj_description((quote_ident(table_schema)||'.'||quote_ident(table_name))::regclass) AS table_comment,
			   pg_table_size(quote_ident(table_schema)||'.'||quote_ident(table_name))/8192 AS size_8k_pages
        FROM information_schema.tables
        WHERE table_type = 'BASE TABLE'
          AND table_schema NOT IN ('pg_catalog','information_schema','pg_toast')
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
            SELECT column_name, data_type, is_nullable = 'YES'
            FROM information_schema.columns
            WHERE table_schema = $1 AND table_name = $2
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
            SELECT a.attname
            FROM pg_index i
            JOIN pg_class c ON i.indrelid = c.oid
            JOIN pg_namespace ns ON c.relnamespace = ns.oid
            JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(i.indkey)
            WHERE ns.nspname = $1 AND c.relname = $2 AND i.indisprimary`, t.Schema, t.Name)
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
        SELECT
          tc.table_schema from_schema, 
          tc.table_name from_table, 
          string_agg(kcu.column_name, ', ' ORDER BY kcu.ordinal_position) from_columns, 
          rkcu.table_schema to_schema, 
          rkcu.table_name to_table,
          string_agg(rkcu.column_name, ', ' ORDER BY rkcu.ordinal_position) to_columns,
		  tc.constraint_name
        FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage kcu
          ON tc.constraint_name = kcu.constraint_name 
         AND tc.constraint_schema = kcu.constraint_schema
        JOIN information_schema.referential_constraints rc
          ON tc.constraint_name = rc.constraint_name 
         AND tc.constraint_schema = rc.constraint_schema
        JOIN information_schema.key_column_usage rkcu
          ON rc.unique_constraint_name = rkcu.constraint_name 
         AND rc.unique_constraint_schema = rkcu.constraint_schema 
         AND kcu.ordinal_position = rkcu.ordinal_position
        WHERE tc.constraint_type = 'FOREIGN KEY'
          AND tc.table_schema NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
        GROUP BY tc.table_schema, tc.table_name, rkcu.table_schema, rkcu.table_name, tc.constraint_name`)
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
	db.Register("postgres", pgExtractor{})
	db.Register("postgresql", pgExtractor{})
}
