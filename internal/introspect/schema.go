package introspect

// Column represents a table column.
type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	PK       bool   `json:"pk"`
}

// ForeignKey represents a foreign key relationship.
type ForeignKey struct {
	FromSchema string `json:"from_schema,omitempty"`
	FromTable  string `json:"from_table"`
	FromColumn string `json:"from_column"`
	ToSchema   string `json:"to_schema,omitempty"`
	ToTable    string `json:"to_table"`
	ToColumn   string `json:"to_column"`
	Constraint string `json:"constraint,omitempty"`
}

// Table represents a database table and its columns.
type Table struct {
	Schema      string   `json:"schema,omitempty"`
	Name        string   `json:"name"`
	Columns     []Column `json:"columns"`
	Rows        int64    `json:"rows,omitempty"`        // optional row estimate/counted value
	Comment     *string  `json:"comment,omitempty"`     // optional table comment
	Size8kPages int64    `json:"size8kPages,omitempty"` // optional size in 8k pages
}

// Schema is the full DB schema extracted for visualization.
type Schema struct {
	Tables      []Table      `json:"tables"`
	ForeignKeys []ForeignKey `json:"foreign_keys"`
}
