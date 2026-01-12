//go:build oracle
// +build oracle

package extractors

import (
	"context"
	"database/sql"
	"fmt"

	"erddiagram/internal/db"
	"erddiagram/internal/introspect"
	"erddiagram/internal/logger"
)

// oracleExtractor implements Extractor for Oracle.
type oracleExtractor struct{}

// This is the extractor for Oracle
func (oracleExtractor) Extract(ctx context.Context, dbConn *sql.DB) (introspect.Schema, error) {
	var s introspect.Schema

	tr, err := dbConn.QueryContext(ctx, `
	    SELECT 
		   ausr.username, 
		   atab.table_name, 
		   acom.comments, 
		   nvl(atab.blocks*nvl(ts.block_size, 8192)/8192, 1) size_8k_pages
	    FROM all_users ausr
	    JOIN all_tables atab 
		  ON ausr.username = atab.owner
	    LEFT JOIN all_tab_comments acom 
		  ON acom.owner = atab.owner 
		 AND acom.table_name = atab.table_name
	    LEFT JOIN user_tablespaces ts 
		  ON atab.tablespace_name = ts.tablespace_name
	    WHERE ausr.oracle_maintained = 'N'
	    ORDER BY ausr.username, atab.table_name`)
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
            SELECT column_name, data_type, nullable
            FROM all_tab_columns
            WHERE owner = :1 AND table_name = :2
            ORDER BY column_id`, t.Schema, t.Name)
		if err != nil {
			return s, fmt.Errorf("query columns for %s.%s: %w", t.Schema, t.Name, err)
		}
		for cr.Next() {
			var col introspect.Column
			var nullable string
			if err := cr.Scan(&col.Name, &col.Type, &nullable); err != nil {
				cr.Close()
				return s, fmt.Errorf("scan column for %s.%s: %w", t.Schema, t.Name, err)
			}
			col.Nullable = (nullable == "Y")
			t.Columns = append(t.Columns, col)
		}
		cr.Close()

		pkr, err := dbConn.QueryContext(ctx, `
            SELECT acc.column_name
            FROM all_cons_columns acc
            JOIN all_constraints ac ON acc.owner = ac.owner AND acc.constraint_name = ac.constraint_name
            WHERE ac.constraint_type = 'P' AND acc.owner = :1 AND acc.table_name = :2`, t.Schema, t.Name)
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
        SELECT a.owner AS from_schema, a.table_name AS from_table, 
		       listagg(acc.column_name, ', ') within group (order by acc.position) AS from_column,
               rcc.owner AS to_schema, rcc.table_name AS to_table, 
			   listagg(rcc.column_name, ', ') within group (order by rcc.position) AS to_column,
			   a.constraint_name
        FROM all_users ausr
		JOIN all_constraints a
		  ON ausr.username = a.owner
        JOIN all_cons_columns acc 
		  ON a.owner = acc.owner 
		 AND a.constraint_name = acc.constraint_name
        JOIN all_cons_columns rcc 
		  ON a.r_owner = rcc.owner 
		 AND a.r_constraint_name = rcc.constraint_name
		 AND nvl(acc.position, 0) = nvl(rcc.position, 0)
        WHERE a.constraint_type = 'R' 
		  AND ausr.oracle_maintained = 'N'
		GROUP BY a.owner, a.table_name, rcc.owner, rcc.table_name, a.constraint_name`)
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
	db.Register("godror", oracleExtractor{})
	db.Register("oracle", oracleExtractor{})
}
