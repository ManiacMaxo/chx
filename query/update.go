package query

import (
	"context"
	"fmt"
	"strings"
)

// UpdateQuery represents an UPDATE query builder (lightweight updates).
type UpdateQuery struct {
	baseQuery

	table QueryWithArgs
	sets  []QueryWithArgs
	where whereClause
}

// NewUpdate creates a new UPDATE query builder.
func NewUpdate(executor Executor) *UpdateQuery {
	return &UpdateQuery{
		baseQuery: baseQuery{executor: executor},
	}
}

// Update creates a new UPDATE query for the given table.
func Update(table string) *UpdateQuery {
	return &UpdateQuery{
		table: QueryWithArgs{Query: quoteIdentifier(table)},
	}
}

// Table sets the table to update.
func (q *UpdateQuery) Table(table string) *UpdateQuery {
	q.table = QueryWithArgs{Query: quoteIdentifier(table)}
	return q
}

// TableExpr sets the table with a raw expression.
func (q *UpdateQuery) TableExpr(expr string, args ...any) *UpdateQuery {
	q.table = QueryWithArgs{Query: expr, Args: args}
	return q
}

// Set sets a column to a value.
func (q *UpdateQuery) Set(column string, value any) *UpdateQuery {
	q.sets = append(q.sets, QueryWithArgs{
		Query: quoteIdentifier(column) + " = ?",
		Args:  []any{value},
	})
	return q
}

// SetExpr sets a column with a raw expression.
func (q *UpdateQuery) SetExpr(expr string, args ...any) *UpdateQuery {
	q.sets = append(q.sets, QueryWithArgs{Query: expr, Args: args})
	return q
}

// SetMap sets multiple columns from a map.
func (q *UpdateQuery) SetMap(values map[string]any) *UpdateQuery {
	for col, val := range values {
		q.Set(col, val)
	}
	return q
}

// Where adds a WHERE condition.
func (q *UpdateQuery) Where(expr string, args ...any) *UpdateQuery {
	q.where.And(expr, args...)
	return q
}

// WhereOr adds an OR WHERE condition.
func (q *UpdateQuery) WhereOr(expr string, args ...any) *UpdateQuery {
	q.where.Or(expr, args...)
	return q
}

// WhereGroup adds a grouped WHERE condition.
func (q *UpdateQuery) WhereGroup(fn func(*UpdateQuery) *UpdateQuery) *UpdateQuery {
	group := &UpdateQuery{}
	fn(group)
	if !group.where.IsEmpty() {
		q.where.addGroup("AND", &group.where)
	}
	return q
}

// WhereIn adds an IN condition.
func (q *UpdateQuery) WhereIn(column string, values any) *UpdateQuery {
	in := InExpr{Values: values}
	expr, inArgs := in.Build()
	q.where.And(quoteIdentifier(column)+" IN "+expr, inArgs...)
	return q
}

// WhereNotIn adds a NOT IN condition.
func (q *UpdateQuery) WhereNotIn(column string, values any) *UpdateQuery {
	in := InExpr{Values: values}
	expr, inArgs := in.Build()
	q.where.And(quoteIdentifier(column)+" NOT IN "+expr, inArgs...)
	return q
}

// OnCluster adds an ON CLUSTER clause.
func (q *UpdateQuery) OnCluster(cluster string) *UpdateQuery {
	q.onCluster = QueryWithArgs{Query: quoteIdentifier(cluster)}
	return q
}

// Setting adds a SETTINGS clause.
func (q *UpdateQuery) Setting(setting string) *UpdateQuery {
	q.appendSetting(setting)
	return q
}

// SettingExpr adds a SETTINGS clause with arguments.
func (q *UpdateQuery) SettingExpr(expr string, args ...any) *UpdateQuery {
	q.appendSetting(expr, args...)
	return q
}

// Build builds the UPDATE query.
func (q *UpdateQuery) Build() (string, []any, error) {
	if q.err != nil {
		return "", nil, q.err
	}

	if q.table.IsEmpty() {
		return "", nil, fmt.Errorf("table is required")
	}

	if len(q.sets) == 0 {
		return "", nil, fmt.Errorf("SET clause is required for UPDATE")
	}

	if q.where.IsEmpty() {
		return "", nil, fmt.Errorf("WHERE clause is required for UPDATE")
	}

	var sb strings.Builder
	var args []any

	sb.WriteString("ALTER TABLE ")
	args = q.table.AppendTo(&sb, args)

	// ON CLUSTER
	args = q.buildOnClusterClause(&sb, args)

	sb.WriteString(" UPDATE ")

	// SET clause
	for i, set := range q.sets {
		if i > 0 {
			sb.WriteString(", ")
		}
		args = set.AppendTo(&sb, args)
	}

	// WHERE
	sb.WriteString(" WHERE ")
	args = q.where.Build(&sb, args)

	// SETTINGS
	args = q.buildSettingsClause(&sb, args)

	return sb.String(), args, nil
}

// String returns the SQL query with arguments interpolated (for debugging).
func (q *UpdateQuery) String() string {
	sql, args, err := q.Build()
	if err != nil {
		return fmt.Sprintf("ERROR: %v", err)
	}
	return interpolateArgs(sql, args)
}

// Exec executes the UPDATE query.
func (q *UpdateQuery) Exec(ctx context.Context) error {
	if q.executor == nil {
		return fmt.Errorf("no executor set")
	}
	sql, args, err := q.Build()
	if err != nil {
		return err
	}
	return q.executor.Exec(ctx, sql, args...)
}
