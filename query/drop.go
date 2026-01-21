package query

import (
	"context"
	"fmt"
	"strings"
)

// DropQuery represents a DROP query builder.
type DropQuery struct {
	baseQuery

	objectType string // TABLE, VIEW, DATABASE, DICTIONARY, etc.
	name       QueryWithArgs
	ifExists   bool
	temporary  bool
	sync       bool
	noDelay    bool
}

// NewDrop creates a new DROP query builder.
func NewDrop(executor Executor) *DropQuery {
	return &DropQuery{
		baseQuery: baseQuery{executor: executor},
	}
}

// DropTable creates a new DROP TABLE query.
func DropTable(name string) *DropQuery {
	return &DropQuery{
		objectType: "TABLE",
		name:       QueryWithArgs{Query: quoteIdentifier(name)},
	}
}

// DropView creates a new DROP VIEW query.
func DropView(name string) *DropQuery {
	return &DropQuery{
		objectType: "VIEW",
		name:       QueryWithArgs{Query: quoteIdentifier(name)},
	}
}

// DropDatabase creates a new DROP DATABASE query.
func DropDatabase(name string) *DropQuery {
	return &DropQuery{
		objectType: "DATABASE",
		name:       QueryWithArgs{Query: quoteIdentifier(name)},
	}
}

// DropDictionary creates a new DROP DICTIONARY query.
func DropDictionary(name string) *DropQuery {
	return &DropQuery{
		objectType: "DICTIONARY",
		name:       QueryWithArgs{Query: quoteIdentifier(name)},
	}
}

// DropFunction creates a new DROP FUNCTION query.
func DropFunction(name string) *DropQuery {
	return &DropQuery{
		objectType: "FUNCTION",
		name:       QueryWithArgs{Query: quoteIdentifier(name)},
	}
}

// Table sets this as a DROP TABLE query.
func (q *DropQuery) Table(name string) *DropQuery {
	q.objectType = "TABLE"
	q.name = QueryWithArgs{Query: quoteIdentifier(name)}
	return q
}

// View sets this as a DROP VIEW query.
func (q *DropQuery) View(name string) *DropQuery {
	q.objectType = "VIEW"
	q.name = QueryWithArgs{Query: quoteIdentifier(name)}
	return q
}

// Database sets this as a DROP DATABASE query.
func (q *DropQuery) Database(name string) *DropQuery {
	q.objectType = "DATABASE"
	q.name = QueryWithArgs{Query: quoteIdentifier(name)}
	return q
}

// IfExists adds IF EXISTS.
func (q *DropQuery) IfExists() *DropQuery {
	q.ifExists = true
	return q
}

// Temporary marks as dropping a temporary table.
func (q *DropQuery) Temporary() *DropQuery {
	q.temporary = true
	return q
}

// OnCluster adds ON CLUSTER clause.
func (q *DropQuery) OnCluster(cluster string) *DropQuery {
	q.onCluster = QueryWithArgs{Query: quoteIdentifier(cluster)}
	return q
}

// Sync adds SYNC (wait for completion).
func (q *DropQuery) Sync() *DropQuery {
	q.sync = true
	return q
}

// NoDelay adds NO DELAY.
func (q *DropQuery) NoDelay() *DropQuery {
	q.noDelay = true
	return q
}

// Setting adds a SETTINGS clause.
func (q *DropQuery) Setting(setting string) *DropQuery {
	q.appendSetting(setting)
	return q
}

// Build builds the DROP query.
func (q *DropQuery) Build() (string, []any, error) {
	if q.err != nil {
		return "", nil, q.err
	}

	if q.objectType == "" {
		return "", nil, fmt.Errorf("object type is required")
	}

	if q.name.IsEmpty() {
		return "", nil, fmt.Errorf("name is required")
	}

	var sb strings.Builder
	var args []any

	sb.WriteString("DROP ")

	if q.temporary {
		sb.WriteString("TEMPORARY ")
	}

	sb.WriteString(q.objectType)
	sb.WriteString(" ")

	if q.ifExists {
		sb.WriteString("IF EXISTS ")
	}

	args = q.name.AppendTo(&sb, args)

	// ON CLUSTER
	args = q.buildOnClusterClause(&sb, args)

	if q.sync {
		sb.WriteString(" SYNC")
	}

	if q.noDelay {
		sb.WriteString(" NO DELAY")
	}

	// SETTINGS
	args = q.buildSettingsClause(&sb, args)

	return sb.String(), args, nil
}

// String returns the SQL query with arguments interpolated (for debugging).
func (q *DropQuery) String() string {
	sql, args, err := q.Build()
	if err != nil {
		return fmt.Sprintf("ERROR: %v", err)
	}
	return interpolateArgs(sql, args)
}

// Exec executes the DROP query.
func (q *DropQuery) Exec(ctx context.Context) error {
	if q.executor == nil {
		return fmt.Errorf("no executor set")
	}
	sql, args, err := q.Build()
	if err != nil {
		return err
	}
	return q.executor.Exec(ctx, sql, args...)
}

// TruncateQuery represents a TRUNCATE query builder.
type TruncateQuery struct {
	baseQuery

	table    QueryWithArgs
	ifExists bool
	sync     bool
}

// NewTruncate creates a new TRUNCATE query builder.
func NewTruncate(executor Executor) *TruncateQuery {
	return &TruncateQuery{
		baseQuery: baseQuery{executor: executor},
	}
}

// Truncate creates a new TRUNCATE TABLE query.
func Truncate(table string) *TruncateQuery {
	return &TruncateQuery{
		table: QueryWithArgs{Query: quoteIdentifier(table)},
	}
}

// Table sets the table to truncate.
func (q *TruncateQuery) Table(table string) *TruncateQuery {
	q.table = QueryWithArgs{Query: quoteIdentifier(table)}
	return q
}

// TableExpr sets the table with a raw expression.
func (q *TruncateQuery) TableExpr(expr string, args ...any) *TruncateQuery {
	q.table = QueryWithArgs{Query: expr, Args: args}
	return q
}

// IfExists adds IF EXISTS.
func (q *TruncateQuery) IfExists() *TruncateQuery {
	q.ifExists = true
	return q
}

// OnCluster adds ON CLUSTER clause.
func (q *TruncateQuery) OnCluster(cluster string) *TruncateQuery {
	q.onCluster = QueryWithArgs{Query: quoteIdentifier(cluster)}
	return q
}

// Sync adds SYNC (wait for completion).
func (q *TruncateQuery) Sync() *TruncateQuery {
	q.sync = true
	return q
}

// Setting adds a SETTINGS clause.
func (q *TruncateQuery) Setting(setting string) *TruncateQuery {
	q.appendSetting(setting)
	return q
}

// Build builds the TRUNCATE query.
func (q *TruncateQuery) Build() (string, []any, error) {
	if q.err != nil {
		return "", nil, q.err
	}

	if q.table.IsEmpty() {
		return "", nil, fmt.Errorf("table is required")
	}

	var sb strings.Builder
	var args []any

	sb.WriteString("TRUNCATE TABLE ")

	if q.ifExists {
		sb.WriteString("IF EXISTS ")
	}

	args = q.table.AppendTo(&sb, args)

	// ON CLUSTER
	args = q.buildOnClusterClause(&sb, args)

	if q.sync {
		sb.WriteString(" SYNC")
	}

	// SETTINGS
	args = q.buildSettingsClause(&sb, args)

	return sb.String(), args, nil
}

// String returns the SQL query with arguments interpolated (for debugging).
func (q *TruncateQuery) String() string {
	sql, args, err := q.Build()
	if err != nil {
		return fmt.Sprintf("ERROR: %v", err)
	}
	return interpolateArgs(sql, args)
}

// Exec executes the TRUNCATE query.
func (q *TruncateQuery) Exec(ctx context.Context) error {
	if q.executor == nil {
		return fmt.Errorf("no executor set")
	}
	sql, args, err := q.Build()
	if err != nil {
		return err
	}
	return q.executor.Exec(ctx, sql, args...)
}
