package query

import (
	"context"
	"fmt"
	"strings"
)

// DeleteQuery represents a DELETE query builder (lightweight deletes).
type DeleteQuery struct {
	baseQuery

	table QueryWithArgs
	where whereClause
}

// NewDelete creates a new DELETE query builder.
func NewDelete(executor Executor) *DeleteQuery {
	return &DeleteQuery{
		baseQuery: baseQuery{executor: executor},
	}
}

// Delete creates a new DELETE query for the given table.
func Delete(table string) *DeleteQuery {
	return &DeleteQuery{
		table: QueryWithArgs{Query: quoteIdentifier(table)},
	}
}

// From sets the table to delete from.
func (q *DeleteQuery) From(table string) *DeleteQuery {
	q.table = QueryWithArgs{Query: quoteIdentifier(table)}
	return q
}

// FromExpr sets the table with a raw expression.
func (q *DeleteQuery) FromExpr(expr string, args ...any) *DeleteQuery {
	q.table = QueryWithArgs{Query: expr, Args: args}
	return q
}

// Where adds a WHERE condition.
func (q *DeleteQuery) Where(expr string, args ...any) *DeleteQuery {
	q.where.And(expr, args...)
	return q
}

// WhereOr adds an OR WHERE condition.
func (q *DeleteQuery) WhereOr(expr string, args ...any) *DeleteQuery {
	q.where.Or(expr, args...)
	return q
}

// WhereGroup adds a grouped WHERE condition.
func (q *DeleteQuery) WhereGroup(fn func(*DeleteQuery) *DeleteQuery) *DeleteQuery {
	group := &DeleteQuery{}
	fn(group)
	if !group.where.IsEmpty() {
		q.where.addGroup("AND", &group.where)
	}
	return q
}

// WhereIn adds an IN condition.
func (q *DeleteQuery) WhereIn(column string, values any) *DeleteQuery {
	in := InExpr{Values: values}
	expr, inArgs := in.Build()
	q.where.And(quoteIdentifier(column)+" IN "+expr, inArgs...)
	return q
}

// WhereNotIn adds a NOT IN condition.
func (q *DeleteQuery) WhereNotIn(column string, values any) *DeleteQuery {
	in := InExpr{Values: values}
	expr, inArgs := in.Build()
	q.where.And(quoteIdentifier(column)+" NOT IN "+expr, inArgs...)
	return q
}

// OnCluster adds an ON CLUSTER clause.
func (q *DeleteQuery) OnCluster(cluster string) *DeleteQuery {
	q.onCluster = QueryWithArgs{Query: quoteIdentifier(cluster)}
	return q
}

// Setting adds a SETTINGS clause.
func (q *DeleteQuery) Setting(setting string) *DeleteQuery {
	q.appendSetting(setting)
	return q
}

// SettingExpr adds a SETTINGS clause with arguments.
func (q *DeleteQuery) SettingExpr(expr string, args ...any) *DeleteQuery {
	q.appendSetting(expr, args...)
	return q
}

// Build builds the DELETE query.
func (q *DeleteQuery) Build() (string, []any, error) {
	if q.err != nil {
		return "", nil, q.err
	}

	if q.table.IsEmpty() {
		return "", nil, fmt.Errorf("table is required")
	}

	if q.where.IsEmpty() {
		return "", nil, fmt.Errorf("WHERE clause is required for DELETE")
	}

	var sb strings.Builder
	var args []any

	sb.WriteString("DELETE FROM ")
	args = q.table.AppendTo(&sb, args)

	// ON CLUSTER
	args = q.buildOnClusterClause(&sb, args)

	// WHERE
	sb.WriteString(" WHERE ")
	args = q.where.Build(&sb, args)

	// SETTINGS
	args = q.buildSettingsClause(&sb, args)

	return sb.String(), args, nil
}

// String returns the SQL query with arguments interpolated (for debugging).
func (q *DeleteQuery) String() string {
	sql, args, err := q.Build()
	if err != nil {
		return fmt.Sprintf("ERROR: %v", err)
	}
	return interpolateArgs(sql, args)
}

// Exec executes the DELETE query.
func (q *DeleteQuery) Exec(ctx context.Context) error {
	if q.executor == nil {
		return fmt.Errorf("no executor set")
	}
	sql, args, err := q.Build()
	if err != nil {
		return err
	}
	return q.executor.Exec(ctx, sql, args...)
}
