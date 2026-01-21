package query

import (
	"context"
	"fmt"
	"strings"
)

// OptimizeQuery represents an OPTIMIZE query builder.
type OptimizeQuery struct {
	baseQuery

	table         QueryWithArgs
	partition     QueryWithArgs
	final         bool
	deduplicate   bool
	deduplicateBy []string
}

// NewOptimize creates a new OPTIMIZE query builder.
func NewOptimize(executor Executor) *OptimizeQuery {
	return &OptimizeQuery{
		baseQuery: baseQuery{executor: executor},
	}
}

// Optimize creates a new OPTIMIZE TABLE query.
func Optimize(table string) *OptimizeQuery {
	return &OptimizeQuery{
		table: QueryWithArgs{Query: quoteIdentifier(table)},
	}
}

// Table sets the table to optimize.
func (q *OptimizeQuery) Table(table string) *OptimizeQuery {
	q.table = QueryWithArgs{Query: quoteIdentifier(table)}
	return q
}

// TableExpr sets the table with a raw expression.
func (q *OptimizeQuery) TableExpr(expr string, args ...any) *OptimizeQuery {
	q.table = QueryWithArgs{Query: expr, Args: args}
	return q
}

// OnCluster adds ON CLUSTER clause.
func (q *OptimizeQuery) OnCluster(cluster string) *OptimizeQuery {
	q.onCluster = QueryWithArgs{Query: quoteIdentifier(cluster)}
	return q
}

// Partition sets the partition to optimize.
func (q *OptimizeQuery) Partition(partition string) *OptimizeQuery {
	q.partition = QueryWithArgs{Query: partition}
	return q
}

// PartitionExpr sets the partition with a raw expression.
func (q *OptimizeQuery) PartitionExpr(expr string, args ...any) *OptimizeQuery {
	q.partition = QueryWithArgs{Query: expr, Args: args}
	return q
}

// Final adds FINAL (force merge even if already optimized).
func (q *OptimizeQuery) Final() *OptimizeQuery {
	q.final = true
	return q
}

// Deduplicate adds DEDUPLICATE.
func (q *OptimizeQuery) Deduplicate() *OptimizeQuery {
	q.deduplicate = true
	return q
}

// DeduplicateBy adds DEDUPLICATE BY columns.
func (q *OptimizeQuery) DeduplicateBy(columns ...string) *OptimizeQuery {
	q.deduplicate = true
	q.deduplicateBy = columns
	return q
}

// DeduplicateByExpr adds DEDUPLICATE BY * EXCEPT.
func (q *OptimizeQuery) DeduplicateByExcept(columns ...string) *OptimizeQuery {
	q.deduplicate = true
	q.deduplicateBy = []string{"* EXCEPT (" + strings.Join(columns, ", ") + ")"}
	return q
}

// Setting adds a SETTINGS clause.
func (q *OptimizeQuery) Setting(setting string) *OptimizeQuery {
	q.appendSetting(setting)
	return q
}

// Build builds the OPTIMIZE query.
func (q *OptimizeQuery) Build() (string, []any, error) {
	if q.err != nil {
		return "", nil, q.err
	}

	if q.table.IsEmpty() {
		return "", nil, fmt.Errorf("table is required")
	}

	var sb strings.Builder
	var args []any

	sb.WriteString("OPTIMIZE TABLE ")
	args = q.table.AppendTo(&sb, args)

	// ON CLUSTER
	args = q.buildOnClusterClause(&sb, args)

	// PARTITION
	if !q.partition.IsEmpty() {
		sb.WriteString(" PARTITION ")
		args = q.partition.AppendTo(&sb, args)
	}

	// FINAL
	if q.final {
		sb.WriteString(" FINAL")
	}

	// DEDUPLICATE
	if q.deduplicate {
		sb.WriteString(" DEDUPLICATE")
		if len(q.deduplicateBy) > 0 {
			sb.WriteString(" BY ")
			for i, col := range q.deduplicateBy {
				if i > 0 {
					sb.WriteString(", ")
				}
				// Check if it's a raw expression (like * EXCEPT)
				if strings.HasPrefix(col, "*") {
					sb.WriteString(col)
				} else {
					sb.WriteString(quoteIdentifier(col))
				}
			}
		}
	}

	// SETTINGS
	args = q.buildSettingsClause(&sb, args)

	return sb.String(), args, nil
}

// String returns the SQL query with arguments interpolated (for debugging).
func (q *OptimizeQuery) String() string {
	sql, args, err := q.Build()
	if err != nil {
		return fmt.Sprintf("ERROR: %v", err)
	}
	return interpolateArgs(sql, args)
}

// Exec executes the OPTIMIZE query.
func (q *OptimizeQuery) Exec(ctx context.Context) error {
	if q.executor == nil {
		return fmt.Errorf("no executor set")
	}
	sql, args, err := q.Build()
	if err != nil {
		return err
	}
	return q.executor.Exec(ctx, sql, args...)
}
