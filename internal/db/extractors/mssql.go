package extractors

import (
	"context"
	"database/sql"
	"fmt"

	"erddiagram/internal/db"
	"erddiagram/internal/introspect"
	"erddiagram/internal/logger"
)

// mssqlExtractor implements Extractor for Microsoft SQL Server.
type mssqlExtractor struct{}

// This is the extractor for Microsoft SQL Server
func (mssqlExtractor) Extract(ctx context.Context, dbConn *sql.DB) (introspect.Schema, error) {
	var s introspect.Schema

	// list tables with schema
	tr, err := dbConn.QueryContext(ctx, `
        SELECT 
          s.name AS schema_name, 
          t.name AS table_name, 
          sep.value AS comment, 
          sum(au.used_pages) as size_8k_pages
        FROM sys.schemas AS s
        JOIN sys.tables AS t 
		  ON s.schema_id = t.schema_id
        LEFT JOIN sys.extended_properties AS sep 
		  ON t.object_id = sep.major_id
         AND sep.minor_id = 0
         AND sep.name = 'MS_Description'
        LEFT JOIN sys.partitions AS p
          ON t.object_id = p.object_id
        LEFT JOIN sys.allocation_units AS au
          ON au.container_id = p.hobt_id
        GROUP BY s.name, t.name, sep.value
        ORDER BY s.name, t.name`)
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

	// columns and PKs for each table
	for i := range s.Tables {
		t := &s.Tables[i]

		cr, err := dbConn.QueryContext(ctx, `
            SELECT COLUMN_NAME, DATA_TYPE, CASE WHEN IS_NULLABLE='YES' THEN 1 ELSE 0 END
            FROM INFORMATION_SCHEMA.COLUMNS
            WHERE TABLE_SCHEMA = @schema AND TABLE_NAME = @table
            ORDER BY ORDINAL_POSITION`, sql.Named("schema", t.Schema), sql.Named("table", t.Name))
		if err != nil {
			return s, fmt.Errorf("query columns for %s.%s: %w", t.Schema, t.Name, err)
		}

		for cr.Next() {
			var col introspect.Column
			var nullableInt int
			if err := cr.Scan(&col.Name, &col.Type, &nullableInt); err != nil {
				cr.Close()
				return s, fmt.Errorf("scan column for %s.%s: %w", t.Schema, t.Name, err)
			}
			col.Nullable = nullableInt == 1
			t.Columns = append(t.Columns, col)
		}
		cr.Close()

		// primary keys
		pkr, err := dbConn.QueryContext(ctx, `
            SELECT k.COLUMN_NAME
            FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS t
            JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE k ON t.CONSTRAINT_NAME = k.CONSTRAINT_NAME AND t.TABLE_SCHEMA = k.TABLE_SCHEMA
            WHERE t.CONSTRAINT_TYPE = 'PRIMARY KEY' AND k.TABLE_SCHEMA = @schema AND k.TABLE_NAME = @table`, sql.Named("schema", t.Schema), sql.Named("table", t.Name))
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

	// foreign keys with schema information
	fkr, err := dbConn.QueryContext(ctx, `
        SELECT
            OBJECT_SCHEMA_NAME(fkc.parent_object_id) AS from_schema,
            OBJECT_NAME(fkc.parent_object_id) AS from_table,
            STRING_AGG(c.NAME, ', ') AS from_column,
            OBJECT_SCHEMA_NAME(fkc.referenced_object_id) AS to_schema,
            OBJECT_NAME(fkc.referenced_object_id) AS to_table,
            STRING_AGG(rc.NAME, ', ') AS to_column,
			fk.name AS constraint_name
        FROM sys.foreign_keys fk
		JOIN sys.foreign_key_columns fkc ON fk.object_id = fkc.constraint_object_id
        JOIN sys.columns c ON fkc.parent_object_id = c.object_id AND fkc.parent_column_id = c.column_id
        JOIN sys.columns rc ON fkc.referenced_object_id = rc.object_id AND fkc.referenced_column_id = rc.column_id
        GROUP BY fk.name, fkc.parent_object_id, fkc.referenced_object_id`)
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
	db.Register("sqlserver", mssqlExtractor{})
	db.Register("mssql", mssqlExtractor{})
}
