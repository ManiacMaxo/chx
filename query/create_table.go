package query

import (
	"context"
	"fmt"
	"strings"
)

// CreateTableQuery represents a CREATE TABLE query builder.
type CreateTableQuery struct {
	baseQuery

	table       QueryWithArgs
	ifNotExists bool
	temporary   bool

	// Column definitions
	columns []columnDefinition

	// Engine configuration
	engine      QueryWithArgs
	orderBy     QueryWithArgs
	partitionBy QueryWithArgs
	primaryKey  QueryWithArgs
	sampleBy    QueryWithArgs
	ttl         QueryWithArgs
	comment     string

	// Create from
	asSelect Builder
	asTable  string
}

// columnDefinition represents a column definition.
type columnDefinition struct {
	name         string
	typ          string
	nullable     bool
	defaultExpr  string
	materialized string
	alias        string
	codec        string
	ttl          string
	comment      string
}

// NewCreateTable creates a new CREATE TABLE query builder.
func NewCreateTable(executor Executor) *CreateTableQuery {
	return &CreateTableQuery{
		baseQuery: baseQuery{executor: executor},
	}
}

// CreateTable creates a new CREATE TABLE query.
func CreateTable(table string) *CreateTableQuery {
	return &CreateTableQuery{
		table: QueryWithArgs{Query: quoteIdentifier(table)},
	}
}

// Table sets the table name.
func (q *CreateTableQuery) Table(table string) *CreateTableQuery {
	q.table = QueryWithArgs{Query: quoteIdentifier(table)}
	return q
}

// TableExpr sets the table with a raw expression.
func (q *CreateTableQuery) TableExpr(expr string, args ...any) *CreateTableQuery {
	q.table = QueryWithArgs{Query: expr, Args: args}
	return q
}

// IfNotExists adds IF NOT EXISTS.
func (q *CreateTableQuery) IfNotExists() *CreateTableQuery {
	q.ifNotExists = true
	return q
}

// Temporary creates a temporary table.
func (q *CreateTableQuery) Temporary() *CreateTableQuery {
	q.temporary = true
	return q
}

// OnCluster adds ON CLUSTER clause.
func (q *CreateTableQuery) OnCluster(cluster string) *CreateTableQuery {
	q.onCluster = QueryWithArgs{Query: quoteIdentifier(cluster)}
	return q
}

// OnClusterExpr adds ON CLUSTER with a raw expression.
func (q *CreateTableQuery) OnClusterExpr(expr string, args ...any) *CreateTableQuery {
	q.onCluster = QueryWithArgs{Query: expr, Args: args}
	return q
}

// Column adds a column definition.
func (q *CreateTableQuery) Column(name, typ string) *ColumnBuilder {
	return &ColumnBuilder{
		parent: q,
		col: columnDefinition{
			name: name,
			typ:  typ,
		},
	}
}

// ColumnExpr adds a raw column expression.
func (q *CreateTableQuery) ColumnExpr(expr string, args ...any) *CreateTableQuery {
	// Parse the expression to extract column name (simplified)
	q.columns = append(q.columns, columnDefinition{
		name: expr, // Store the whole expression
	})
	return q
}

// Engine sets the table engine.
func (q *CreateTableQuery) Engine(engine string) *CreateTableQuery {
	q.engine = QueryWithArgs{Query: engine}
	return q
}

// EngineExpr sets the table engine with a raw expression.
func (q *CreateTableQuery) EngineExpr(expr string, args ...any) *CreateTableQuery {
	q.engine = QueryWithArgs{Query: expr, Args: args}
	return q
}

// MergeTree sets MergeTree engine.
func (q *CreateTableQuery) MergeTree() *CreateTableQuery {
	return q.Engine("MergeTree()")
}

// ReplacingMergeTree sets ReplacingMergeTree engine.
func (q *CreateTableQuery) ReplacingMergeTree(ver ...string) *CreateTableQuery {
	if len(ver) > 0 {
		return q.Engine(fmt.Sprintf("ReplacingMergeTree(%s)", ver[0]))
	}
	return q.Engine("ReplacingMergeTree()")
}

// SummingMergeTree sets SummingMergeTree engine.
func (q *CreateTableQuery) SummingMergeTree(columns ...string) *CreateTableQuery {
	if len(columns) > 0 {
		return q.Engine(fmt.Sprintf("SummingMergeTree((%s))", strings.Join(columns, ", ")))
	}
	return q.Engine("SummingMergeTree()")
}

// AggregatingMergeTree sets AggregatingMergeTree engine.
func (q *CreateTableQuery) AggregatingMergeTree() *CreateTableQuery {
	return q.Engine("AggregatingMergeTree()")
}

// CollapsingMergeTree sets CollapsingMergeTree engine.
func (q *CreateTableQuery) CollapsingMergeTree(sign string) *CreateTableQuery {
	return q.Engine(fmt.Sprintf("CollapsingMergeTree(%s)", sign))
}

// VersionedCollapsingMergeTree sets VersionedCollapsingMergeTree engine.
func (q *CreateTableQuery) VersionedCollapsingMergeTree(sign, version string) *CreateTableQuery {
	return q.Engine(fmt.Sprintf("VersionedCollapsingMergeTree(%s, %s)", sign, version))
}

// ReplicatedMergeTree sets ReplicatedMergeTree engine.
func (q *CreateTableQuery) ReplicatedMergeTree(zkPath, replica string) *CreateTableQuery {
	return q.Engine(fmt.Sprintf("ReplicatedMergeTree('%s', '%s')", escapeString(zkPath), escapeString(replica)))
}

// ReplicatedReplacingMergeTree sets ReplicatedReplacingMergeTree engine.
func (q *CreateTableQuery) ReplicatedReplacingMergeTree(zkPath, replica string, ver ...string) *CreateTableQuery {
	if len(ver) > 0 {
		return q.Engine(fmt.Sprintf("ReplicatedReplacingMergeTree('%s', '%s', %s)", escapeString(zkPath), escapeString(replica), ver[0]))
	}
	return q.Engine(fmt.Sprintf("ReplicatedReplacingMergeTree('%s', '%s')", escapeString(zkPath), escapeString(replica)))
}

// Memory sets Memory engine.
func (q *CreateTableQuery) Memory() *CreateTableQuery {
	return q.Engine("Memory")
}

// Log sets Log engine.
func (q *CreateTableQuery) Log() *CreateTableQuery {
	return q.Engine("Log")
}

// TinyLog sets TinyLog engine.
func (q *CreateTableQuery) TinyLog() *CreateTableQuery {
	return q.Engine("TinyLog")
}

// StripeLog sets StripeLog engine.
func (q *CreateTableQuery) StripeLog() *CreateTableQuery {
	return q.Engine("StripeLog")
}

// Null sets Null engine.
func (q *CreateTableQuery) Null() *CreateTableQuery {
	return q.Engine("Null")
}

// Buffer sets Buffer engine.
func (q *CreateTableQuery) Buffer(database, table string, numLayers, minTime, maxTime, minRows, maxRows, minBytes, maxBytes int) *CreateTableQuery {
	return q.Engine(fmt.Sprintf("Buffer(%s, %s, %d, %d, %d, %d, %d, %d, %d)",
		database, table, numLayers, minTime, maxTime, minRows, maxRows, minBytes, maxBytes))
}

// Distributed sets Distributed engine.
func (q *CreateTableQuery) Distributed(cluster, database, table string, shardingKey ...string) *CreateTableQuery {
	if len(shardingKey) > 0 {
		return q.Engine(fmt.Sprintf("Distributed(%s, %s, %s, %s)", cluster, database, table, shardingKey[0]))
	}
	return q.Engine(fmt.Sprintf("Distributed(%s, %s, %s)", cluster, database, table))
}

// OrderBy sets ORDER BY (primary key for MergeTree).
func (q *CreateTableQuery) OrderBy(columns ...string) *CreateTableQuery {
	quoted := make([]string, len(columns))
	for i, col := range columns {
		quoted[i] = quoteIdentifier(col)
	}
	q.orderBy = QueryWithArgs{Query: strings.Join(quoted, ", ")}
	return q
}

// OrderByExpr sets ORDER BY with a raw expression.
func (q *CreateTableQuery) OrderByExpr(expr string, args ...any) *CreateTableQuery {
	q.orderBy = QueryWithArgs{Query: expr, Args: args}
	return q
}

// PartitionBy sets PARTITION BY.
func (q *CreateTableQuery) PartitionBy(expr string) *CreateTableQuery {
	q.partitionBy = QueryWithArgs{Query: expr}
	return q
}

// PartitionByExpr sets PARTITION BY with a raw expression.
func (q *CreateTableQuery) PartitionByExpr(expr string, args ...any) *CreateTableQuery {
	q.partitionBy = QueryWithArgs{Query: expr, Args: args}
	return q
}

// PrimaryKey sets PRIMARY KEY.
func (q *CreateTableQuery) PrimaryKey(columns ...string) *CreateTableQuery {
	quoted := make([]string, len(columns))
	for i, col := range columns {
		quoted[i] = quoteIdentifier(col)
	}
	q.primaryKey = QueryWithArgs{Query: strings.Join(quoted, ", ")}
	return q
}

// PrimaryKeyExpr sets PRIMARY KEY with a raw expression.
func (q *CreateTableQuery) PrimaryKeyExpr(expr string, args ...any) *CreateTableQuery {
	q.primaryKey = QueryWithArgs{Query: expr, Args: args}
	return q
}

// SampleBy sets SAMPLE BY.
func (q *CreateTableQuery) SampleBy(expr string) *CreateTableQuery {
	q.sampleBy = QueryWithArgs{Query: expr}
	return q
}

// SampleByExpr sets SAMPLE BY with a raw expression.
func (q *CreateTableQuery) SampleByExpr(expr string, args ...any) *CreateTableQuery {
	q.sampleBy = QueryWithArgs{Query: expr, Args: args}
	return q
}

// TTL sets the TTL expression.
func (q *CreateTableQuery) TTL(expr string) *CreateTableQuery {
	q.ttl = QueryWithArgs{Query: expr}
	return q
}

// TTLExpr sets the TTL with a raw expression.
func (q *CreateTableQuery) TTLExpr(expr string, args ...any) *CreateTableQuery {
	q.ttl = QueryWithArgs{Query: expr, Args: args}
	return q
}

// Comment sets the table comment.
func (q *CreateTableQuery) Comment(comment string) *CreateTableQuery {
	q.comment = comment
	return q
}

// Setting adds a SETTINGS clause.
func (q *CreateTableQuery) Setting(setting string) *CreateTableQuery {
	q.appendSetting(setting)
	return q
}

// SettingExpr adds a SETTINGS clause with arguments.
func (q *CreateTableQuery) SettingExpr(expr string, args ...any) *CreateTableQuery {
	q.appendSetting(expr, args...)
	return q
}

// As creates the table structure from another table.
func (q *CreateTableQuery) As(table string) *CreateTableQuery {
	q.asTable = table
	return q
}

// AsSelect creates the table from a SELECT query.
func (q *CreateTableQuery) AsSelect(selectQuery Builder) *CreateTableQuery {
	q.asSelect = selectQuery
	return q
}

// Build builds the CREATE TABLE query.
func (q *CreateTableQuery) Build() (string, []any, error) {
	if q.err != nil {
		return "", nil, q.err
	}

	if q.table.IsEmpty() {
		return "", nil, fmt.Errorf("table is required")
	}

	var sb strings.Builder
	var args []any

	// CREATE [TEMPORARY] TABLE
	sb.WriteString("CREATE ")
	if q.temporary {
		sb.WriteString("TEMPORARY ")
	}
	sb.WriteString("TABLE ")

	// IF NOT EXISTS
	if q.ifNotExists {
		sb.WriteString("IF NOT EXISTS ")
	}

	// Table name
	args = q.table.AppendTo(&sb, args)

	// ON CLUSTER
	args = q.buildOnClusterClause(&sb, args)

	// AS table (clone structure)
	if q.asTable != "" {
		sb.WriteString(" AS ")
		sb.WriteString(quoteIdentifier(q.asTable))
		return sb.String(), args, nil
	}

	// Columns
	if len(q.columns) > 0 {
		sb.WriteString(" (")
		for i, col := range q.columns {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(q.buildColumnDef(&col))
		}
		sb.WriteString(")")
	}

	// ENGINE
	if !q.engine.IsEmpty() {
		sb.WriteString(" ENGINE = ")
		args = q.engine.AppendTo(&sb, args)
	}

	// PARTITION BY
	if !q.partitionBy.IsEmpty() {
		sb.WriteString(" PARTITION BY ")
		args = q.partitionBy.AppendTo(&sb, args)
	}

	// PRIMARY KEY
	if !q.primaryKey.IsEmpty() {
		sb.WriteString(" PRIMARY KEY (")
		args = q.primaryKey.AppendTo(&sb, args)
		sb.WriteString(")")
	}

	// ORDER BY
	if !q.orderBy.IsEmpty() {
		sb.WriteString(" ORDER BY (")
		args = q.orderBy.AppendTo(&sb, args)
		sb.WriteString(")")
	}

	// SAMPLE BY
	if !q.sampleBy.IsEmpty() {
		sb.WriteString(" SAMPLE BY ")
		args = q.sampleBy.AppendTo(&sb, args)
	}

	// TTL
	if !q.ttl.IsEmpty() {
		sb.WriteString(" TTL ")
		args = q.ttl.AppendTo(&sb, args)
	}

	// SETTINGS
	args = q.buildSettingsClause(&sb, args)

	// COMMENT
	if q.comment != "" {
		sb.WriteString(" COMMENT '")
		sb.WriteString(escapeString(q.comment))
		sb.WriteString("'")
	}

	// AS SELECT
	if q.asSelect != nil {
		sb.WriteString(" AS ")
		sql, selectArgs, err := q.asSelect.Build()
		if err != nil {
			return "", nil, err
		}
		sb.WriteString(sql)
		args = append(args, selectArgs...)
	}

	return sb.String(), args, nil
}

// buildColumnDef builds a column definition string.
func (q *CreateTableQuery) buildColumnDef(col *columnDefinition) string {
	var sb strings.Builder

	// If the name contains a type, it's a raw expression
	if strings.Contains(col.name, " ") && col.typ == "" {
		return col.name
	}

	sb.WriteString(quoteIdentifier(col.name))
	sb.WriteString(" ")
	sb.WriteString(col.typ)

	if col.nullable {
		sb.WriteString(" NULL")
	}

	if col.defaultExpr != "" {
		sb.WriteString(" DEFAULT ")
		sb.WriteString(col.defaultExpr)
	}

	if col.materialized != "" {
		sb.WriteString(" MATERIALIZED ")
		sb.WriteString(col.materialized)
	}

	if col.alias != "" {
		sb.WriteString(" ALIAS ")
		sb.WriteString(col.alias)
	}

	if col.codec != "" {
		sb.WriteString(" CODEC(")
		sb.WriteString(col.codec)
		sb.WriteString(")")
	}

	if col.ttl != "" {
		sb.WriteString(" TTL ")
		sb.WriteString(col.ttl)
	}

	if col.comment != "" {
		sb.WriteString(" COMMENT '")
		sb.WriteString(escapeString(col.comment))
		sb.WriteString("'")
	}

	return sb.String()
}

// String returns the SQL query with arguments interpolated (for debugging).
func (q *CreateTableQuery) String() string {
	sql, args, err := q.Build()
	if err != nil {
		return fmt.Sprintf("ERROR: %v", err)
	}
	return interpolateArgs(sql, args)
}

// Exec executes the CREATE TABLE query.
func (q *CreateTableQuery) Exec(ctx context.Context) error {
	if q.executor == nil {
		return fmt.Errorf("no executor set")
	}
	sql, args, err := q.Build()
	if err != nil {
		return err
	}
	return q.executor.Exec(ctx, sql, args...)
}

// ColumnBuilder helps build column definitions.
type ColumnBuilder struct {
	parent *CreateTableQuery
	col    columnDefinition
}

// Nullable marks the column as nullable.
func (b *ColumnBuilder) Nullable() *ColumnBuilder {
	b.col.nullable = true
	return b
}

// Default sets the default expression.
func (b *ColumnBuilder) Default(expr string) *ColumnBuilder {
	b.col.defaultExpr = expr
	return b
}

// Materialized sets the materialized expression.
func (b *ColumnBuilder) Materialized(expr string) *ColumnBuilder {
	b.col.materialized = expr
	return b
}

// Alias sets the alias expression.
func (b *ColumnBuilder) Alias(expr string) *ColumnBuilder {
	b.col.alias = expr
	return b
}

// Codec sets the compression codec.
func (b *ColumnBuilder) Codec(codec string) *ColumnBuilder {
	b.col.codec = codec
	return b
}

// TTL sets the column TTL.
func (b *ColumnBuilder) TTL(expr string) *ColumnBuilder {
	b.col.ttl = expr
	return b
}

// Comment sets the column comment.
func (b *ColumnBuilder) Comment(comment string) *ColumnBuilder {
	b.col.comment = comment
	return b
}

// Add adds the column to the table and returns the CreateTableQuery.
func (b *ColumnBuilder) Add() *CreateTableQuery {
	b.parent.columns = append(b.parent.columns, b.col)
	return b.parent
}

// End is an alias for Add.
func (b *ColumnBuilder) End() *CreateTableQuery {
	return b.Add()
}
