// Package query provides a fluent query builder for ClickHouse.
package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Builder is the interface that all query builders implement.
type Builder interface {
	// Build returns the SQL query string and arguments.
	Build() (string, []any, error)
	// String returns the SQL query with arguments interpolated (for debugging).
	String() string
}

// Executor provides query execution capabilities.
type Executor interface {
	Query(ctx context.Context, query string, args ...any) (driver.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) driver.Row
	Exec(ctx context.Context, query string, args ...any) error
}

// QueryWithArgs holds a query fragment with its arguments.
type QueryWithArgs struct {
	Query string
	Args  []any
}

// IsEmpty returns true if the query is empty.
func (q QueryWithArgs) IsEmpty() bool {
	return q.Query == ""
}

// AppendTo appends this query fragment to a string builder and args slice.
func (q QueryWithArgs) AppendTo(sb *strings.Builder, args []any) []any {
	sb.WriteString(q.Query)
	return append(args, q.Args...)
}

// baseQuery contains shared fields for all query types.
type baseQuery struct {
	executor  Executor
	err       error
	with      []withClause
	settings  []QueryWithArgs
	onCluster QueryWithArgs
}

// withClause represents a CTE (Common Table Expression).
type withClause struct {
	name      string
	recursive bool
	query     Builder
	expr      QueryWithArgs
}

// setError sets an error if one hasn't been set already.
func (q *baseQuery) setError(err error) {
	if q.err == nil {
		q.err = err
	}
}

// appendWith adds a CTE to the query.
func (q *baseQuery) appendWith(name string, recursive bool, query Builder) {
	q.with = append(q.with, withClause{
		name:      name,
		recursive: recursive,
		query:     query,
	})
}

// appendWithExpr adds a CTE expression to the query.
func (q *baseQuery) appendWithExpr(expr string, args ...any) {
	q.with = append(q.with, withClause{
		expr: QueryWithArgs{Query: expr, Args: args},
	})
}

// appendSetting adds a SETTINGS clause.
func (q *baseQuery) appendSetting(setting string, args ...any) {
	q.settings = append(q.settings, QueryWithArgs{Query: setting, Args: args})
}

// buildWithClause builds the WITH clause.
func (q *baseQuery) buildWithClause(sb *strings.Builder, args []any) ([]any, error) {
	if len(q.with) == 0 {
		return args, nil
	}

	sb.WriteString("WITH ")
	for i, w := range q.with {
		if i > 0 {
			sb.WriteString(", ")
		}

		if w.query != nil {
			if w.recursive {
				sb.WriteString("RECURSIVE ")
			}
			sb.WriteString(w.name)
			sb.WriteString(" AS (")
			sql, queryArgs, err := w.query.Build()
			if err != nil {
				return nil, err
			}
			sb.WriteString(sql)
			sb.WriteString(")")
			args = append(args, queryArgs...)
		} else {
			args = w.expr.AppendTo(sb, args)
		}
	}
	sb.WriteString(" ")

	return args, nil
}

// buildSettingsClause builds the SETTINGS clause.
func (q *baseQuery) buildSettingsClause(sb *strings.Builder, args []any) []any {
	if len(q.settings) == 0 {
		return args
	}

	sb.WriteString(" SETTINGS ")
	for i, s := range q.settings {
		if i > 0 {
			sb.WriteString(", ")
		}
		args = s.AppendTo(sb, args)
	}

	return args
}

// buildOnClusterClause builds the ON CLUSTER clause.
func (q *baseQuery) buildOnClusterClause(sb *strings.Builder, args []any) []any {
	if q.onCluster.IsEmpty() {
		return args
	}

	sb.WriteString(" ON CLUSTER ")
	return q.onCluster.AppendTo(sb, args)
}

// quoteIdentifier quotes a ClickHouse identifier.
func quoteIdentifier(name string) string {
	// If already quoted or contains special chars that suggest it's an expression, return as-is
	if strings.HasPrefix(name, "`") || strings.HasPrefix(name, "\"") {
		return name
	}
	// Check if it looks like an expression (contains operators, parens, etc.)
	if strings.ContainsAny(name, "()+-*/=<>!,. ") {
		return name
	}
	// Handle qualified names (db.table.column)
	parts := strings.Split(name, ".")
	for i, part := range parts {
		if part == "*" {
			continue
		}
		parts[i] = "`" + strings.ReplaceAll(part, "`", "``") + "`"
	}
	return strings.Join(parts, ".")
}

// escapeString escapes a string value for ClickHouse.
func escapeString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return s
}

// formatValue formats a value for SQL.
func formatValue(v any) string {
	switch val := v.(type) {
	case nil:
		return "NULL"
	case string:
		return "'" + escapeString(val) + "'"
	case bool:
		if val {
			return "1"
		}
		return "0"
	case Identifier:
		return quoteIdentifier(string(val))
	case SafeString:
		return string(val)
	case RawExpr:
		return string(val)
	case fmt.Stringer:
		return "'" + escapeString(val.String()) + "'"
	default:
		return fmt.Sprintf("%v", val)
	}
}

// interpolateArgs replaces ? placeholders with formatted values.
// This is only for String() debugging output, not for actual query execution.
func interpolateArgs(query string, args []any) string {
	if len(args) == 0 {
		return query
	}

	result := query
	for _, arg := range args {
		idx := strings.Index(result, "?")
		if idx == -1 {
			break
		}
		result = result[:idx] + formatValue(arg) + result[idx+1:]
	}
	return result
}
