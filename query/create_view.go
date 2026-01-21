package query

import (
	"context"
	"fmt"
	"strings"
)

// CreateViewQuery represents a CREATE VIEW query builder.
type CreateViewQuery struct {
	baseQuery

	view         QueryWithArgs
	materialized bool
	ifNotExists  bool
	populate     bool
	toTable      QueryWithArgs
	engine       QueryWithArgs
	orderBy      QueryWithArgs
	partitionBy  QueryWithArgs
	primaryKey   QueryWithArgs
	selectQuery  Builder
}

// NewCreateView creates a new CREATE VIEW query builder.
func NewCreateView(executor Executor) *CreateViewQuery {
	return &CreateViewQuery{
		baseQuery: baseQuery{executor: executor},
	}
}

// CreateView creates a new CREATE VIEW query.
func CreateView(name string) *CreateViewQuery {
	return &CreateViewQuery{
		view: QueryWithArgs{Query: quoteIdentifier(name)},
	}
}

// CreateMaterializedView creates a new CREATE MATERIALIZED VIEW query.
func CreateMaterializedView(name string) *CreateViewQuery {
	return &CreateViewQuery{
		view:         QueryWithArgs{Query: quoteIdentifier(name)},
		materialized: true,
	}
}

// View sets the view name.
func (q *CreateViewQuery) View(name string) *CreateViewQuery {
	q.view = QueryWithArgs{Query: quoteIdentifier(name)}
	return q
}

// ViewExpr sets the view name with a raw expression.
func (q *CreateViewQuery) ViewExpr(expr string, args ...any) *CreateViewQuery {
	q.view = QueryWithArgs{Query: expr, Args: args}
	return q
}

// Materialized makes this a materialized view.
func (q *CreateViewQuery) Materialized() *CreateViewQuery {
	q.materialized = true
	return q
}

// IfNotExists adds IF NOT EXISTS.
func (q *CreateViewQuery) IfNotExists() *CreateViewQuery {
	q.ifNotExists = true
	return q
}

// OnCluster adds ON CLUSTER clause.
func (q *CreateViewQuery) OnCluster(cluster string) *CreateViewQuery {
	q.onCluster = QueryWithArgs{Query: quoteIdentifier(cluster)}
	return q
}

// OnClusterExpr adds ON CLUSTER with a raw expression.
func (q *CreateViewQuery) OnClusterExpr(expr string, args ...any) *CreateViewQuery {
	q.onCluster = QueryWithArgs{Query: expr, Args: args}
	return q
}

// To sets the target table for materialized views.
func (q *CreateViewQuery) To(table string) *CreateViewQuery {
	q.toTable = QueryWithArgs{Query: quoteIdentifier(table)}
	return q
}

// ToExpr sets the target table with a raw expression.
func (q *CreateViewQuery) ToExpr(expr string, args ...any) *CreateViewQuery {
	q.toTable = QueryWithArgs{Query: expr, Args: args}
	return q
}

// Populate adds POPULATE (materialize existing data).
func (q *CreateViewQuery) Populate() *CreateViewQuery {
	q.populate = true
	return q
}

// Engine sets the table engine (for materialized views without TO).
func (q *CreateViewQuery) Engine(engine string) *CreateViewQuery {
	q.engine = QueryWithArgs{Query: engine}
	return q
}

// EngineExpr sets the table engine with a raw expression.
func (q *CreateViewQuery) EngineExpr(expr string, args ...any) *CreateViewQuery {
	q.engine = QueryWithArgs{Query: expr, Args: args}
	return q
}

// OrderBy sets ORDER BY (for materialized views without TO).
func (q *CreateViewQuery) OrderBy(columns ...string) *CreateViewQuery {
	quoted := make([]string, len(columns))
	for i, col := range columns {
		quoted[i] = quoteIdentifier(col)
	}
	q.orderBy = QueryWithArgs{Query: strings.Join(quoted, ", ")}
	return q
}

// OrderByExpr sets ORDER BY with a raw expression.
func (q *CreateViewQuery) OrderByExpr(expr string, args ...any) *CreateViewQuery {
	q.orderBy = QueryWithArgs{Query: expr, Args: args}
	return q
}

// PartitionBy sets PARTITION BY.
func (q *CreateViewQuery) PartitionBy(expr string) *CreateViewQuery {
	q.partitionBy = QueryWithArgs{Query: expr}
	return q
}

// PartitionByExpr sets PARTITION BY with a raw expression.
func (q *CreateViewQuery) PartitionByExpr(expr string, args ...any) *CreateViewQuery {
	q.partitionBy = QueryWithArgs{Query: expr, Args: args}
	return q
}

// PrimaryKey sets PRIMARY KEY.
func (q *CreateViewQuery) PrimaryKey(columns ...string) *CreateViewQuery {
	quoted := make([]string, len(columns))
	for i, col := range columns {
		quoted[i] = quoteIdentifier(col)
	}
	q.primaryKey = QueryWithArgs{Query: strings.Join(quoted, ", ")}
	return q
}

// As sets the SELECT query for the view.
func (q *CreateViewQuery) As(selectQuery Builder) *CreateViewQuery {
	q.selectQuery = selectQuery
	return q
}

// Setting adds a SETTINGS clause.
func (q *CreateViewQuery) Setting(setting string) *CreateViewQuery {
	q.appendSetting(setting)
	return q
}

// SettingExpr adds a SETTINGS clause with arguments.
func (q *CreateViewQuery) SettingExpr(expr string, args ...any) *CreateViewQuery {
	q.appendSetting(expr, args...)
	return q
}

// Build builds the CREATE VIEW query.
func (q *CreateViewQuery) Build() (string, []any, error) {
	if q.err != nil {
		return "", nil, q.err
	}

	if q.view.IsEmpty() {
		return "", nil, fmt.Errorf("view name is required")
	}

	if q.selectQuery == nil {
		return "", nil, fmt.Errorf("AS SELECT is required")
	}

	var sb strings.Builder
	var args []any

	// CREATE [MATERIALIZED] VIEW
	sb.WriteString("CREATE ")
	if q.materialized {
		sb.WriteString("MATERIALIZED ")
	}
	sb.WriteString("VIEW ")

	// IF NOT EXISTS
	if q.ifNotExists {
		sb.WriteString("IF NOT EXISTS ")
	}

	// View name
	args = q.view.AppendTo(&sb, args)

	// ON CLUSTER
	args = q.buildOnClusterClause(&sb, args)

	// TO table (for materialized views)
	if !q.toTable.IsEmpty() {
		sb.WriteString(" TO ")
		args = q.toTable.AppendTo(&sb, args)
	}

	// ENGINE (for materialized views without TO)
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

	// POPULATE
	if q.populate {
		sb.WriteString(" POPULATE")
	}

	// SETTINGS
	args = q.buildSettingsClause(&sb, args)

	// AS SELECT
	sb.WriteString(" AS ")
	sql, selectArgs, err := q.selectQuery.Build()
	if err != nil {
		return "", nil, err
	}
	sb.WriteString(sql)
	args = append(args, selectArgs...)

	return sb.String(), args, nil
}

// String returns the SQL query with arguments interpolated (for debugging).
func (q *CreateViewQuery) String() string {
	sql, args, err := q.Build()
	if err != nil {
		return fmt.Sprintf("ERROR: %v", err)
	}
	return interpolateArgs(sql, args)
}

// Exec executes the CREATE VIEW query.
func (q *CreateViewQuery) Exec(ctx context.Context) error {
	if q.executor == nil {
		return fmt.Errorf("no executor set")
	}
	sql, args, err := q.Build()
	if err != nil {
		return err
	}
	return q.executor.Exec(ctx, sql, args...)
}
