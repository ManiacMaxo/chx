package query

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// InsertQuery represents an INSERT query builder.
type InsertQuery struct {
	baseQuery

	table       QueryWithArgs
	columns     []string
	values      [][]any
	selectQuery Builder
	format      string
}

// NewInsert creates a new INSERT query builder.
func NewInsert(executor Executor) *InsertQuery {
	return &InsertQuery{
		baseQuery: baseQuery{executor: executor},
	}
}

// Insert creates a new INSERT query for the given table.
func Insert(table string) *InsertQuery {
	return &InsertQuery{
		table: QueryWithArgs{Query: quoteIdentifier(table)},
	}
}

// Into sets the table to insert into.
func (q *InsertQuery) Into(table string) *InsertQuery {
	q.table = QueryWithArgs{Query: quoteIdentifier(table)}
	return q
}

// IntoExpr sets the table with a raw expression.
func (q *InsertQuery) IntoExpr(expr string, args ...any) *InsertQuery {
	q.table = QueryWithArgs{Query: expr, Args: args}
	return q
}

// Column adds columns for the INSERT.
func (q *InsertQuery) Column(columns ...string) *InsertQuery {
	q.columns = append(q.columns, columns...)
	return q
}

// Columns is an alias for Column.
func (q *InsertQuery) Columns(columns ...string) *InsertQuery {
	return q.Column(columns...)
}

// Values adds a row of values.
func (q *InsertQuery) Values(values ...any) *InsertQuery {
	q.values = append(q.values, values)
	return q
}

// Rows adds multiple rows of values.
func (q *InsertQuery) Rows(rows [][]any) *InsertQuery {
	q.values = append(q.values, rows...)
	return q
}

// Struct adds values from a struct.
func (q *InsertQuery) Struct(v any) *InsertQuery {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		q.setError(fmt.Errorf("Struct requires a struct, got %T", v))
		return q
	}

	typ := val.Type()
	row := make([]any, 0, val.NumField())

	// If columns not set, use struct field names
	if len(q.columns) == 0 {
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if !field.IsExported() {
				continue
			}
			// Check for ch tag
			tag := field.Tag.Get("ch")
			if tag == "-" {
				continue
			}
			colName := field.Name
			if tag != "" {
				parts := strings.Split(tag, ",")
				if parts[0] != "" {
					colName = parts[0]
				}
			}
			q.columns = append(q.columns, colName)
			row = append(row, val.Field(i).Interface())
		}
	} else {
		// Match columns to struct fields
		fieldMap := make(map[string]int)
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if !field.IsExported() {
				continue
			}
			tag := field.Tag.Get("ch")
			if tag == "-" {
				continue
			}
			name := field.Name
			if tag != "" {
				parts := strings.Split(tag, ",")
				if parts[0] != "" {
					name = parts[0]
				}
			}
			fieldMap[strings.ToLower(name)] = i
		}

		for _, col := range q.columns {
			if idx, ok := fieldMap[strings.ToLower(col)]; ok {
				row = append(row, val.Field(idx).Interface())
			} else {
				row = append(row, nil)
			}
		}
	}

	q.values = append(q.values, row)
	return q
}

// Structs adds values from a slice of structs.
func (q *InsertQuery) Structs(v any) *InsertQuery {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Slice {
		q.setError(fmt.Errorf("Structs requires a slice, got %T", v))
		return q
	}

	for i := 0; i < val.Len(); i++ {
		q.Struct(val.Index(i).Interface())
	}
	return q
}

// Map adds values from a map.
func (q *InsertQuery) Map(m map[string]any) *InsertQuery {
	if len(q.columns) == 0 {
		// Use map keys as columns
		for k := range m {
			q.columns = append(q.columns, k)
		}
	}

	row := make([]any, len(q.columns))
	for i, col := range q.columns {
		row[i] = m[col]
	}
	q.values = append(q.values, row)
	return q
}

// Select sets an INSERT ... SELECT query.
func (q *InsertQuery) Select(selectQuery Builder) *InsertQuery {
	q.selectQuery = selectQuery
	return q
}

// Format sets the input format.
func (q *InsertQuery) Format(format string) *InsertQuery {
	q.format = format
	return q
}

// Setting adds a SETTINGS clause.
func (q *InsertQuery) Setting(setting string) *InsertQuery {
	q.appendSetting(setting)
	return q
}

// SettingExpr adds a SETTINGS clause with arguments.
func (q *InsertQuery) SettingExpr(expr string, args ...any) *InsertQuery {
	q.appendSetting(expr, args...)
	return q
}

// OnCluster adds an ON CLUSTER clause.
func (q *InsertQuery) OnCluster(cluster string) *InsertQuery {
	q.onCluster = QueryWithArgs{Query: quoteIdentifier(cluster)}
	return q
}

// Build builds the INSERT query.
func (q *InsertQuery) Build() (string, []any, error) {
	if q.err != nil {
		return "", nil, q.err
	}

	if q.table.IsEmpty() {
		return "", nil, fmt.Errorf("table is required")
	}

	var sb strings.Builder
	var args []any

	sb.WriteString("INSERT INTO ")
	args = q.table.AppendTo(&sb, args)

	// ON CLUSTER
	args = q.buildOnClusterClause(&sb, args)

	// Columns
	if len(q.columns) > 0 {
		sb.WriteString(" (")
		for i, col := range q.columns {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(quoteIdentifier(col))
		}
		sb.WriteString(")")
	}

	// VALUES or SELECT
	if q.selectQuery != nil {
		sb.WriteString(" ")
		sql, selectArgs, err := q.selectQuery.Build()
		if err != nil {
			return "", nil, err
		}
		sb.WriteString(sql)
		args = append(args, selectArgs...)
	} else if len(q.values) > 0 {
		sb.WriteString(" VALUES ")
		for i, row := range q.values {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("(")
			for j, val := range row {
				if j > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString("?")
				args = append(args, val)
			}
			sb.WriteString(")")
		}
	}

	// FORMAT
	if q.format != "" {
		sb.WriteString(" FORMAT ")
		sb.WriteString(q.format)
	}

	// SETTINGS
	args = q.buildSettingsClause(&sb, args)

	return sb.String(), args, nil
}

// String returns the SQL query with arguments interpolated (for debugging).
func (q *InsertQuery) String() string {
	sql, args, err := q.Build()
	if err != nil {
		return fmt.Sprintf("ERROR: %v", err)
	}
	return interpolateArgs(sql, args)
}

// Exec executes the INSERT query.
func (q *InsertQuery) Exec(ctx context.Context) error {
	if q.executor == nil {
		return fmt.Errorf("no executor set")
	}
	sql, args, err := q.Build()
	if err != nil {
		return err
	}
	return q.executor.Exec(ctx, sql, args...)
}

// Batch returns a batch for bulk inserts.
func (q *InsertQuery) Batch(ctx context.Context) (driver.Batch, error) {
	if q.executor == nil {
		return nil, fmt.Errorf("no executor set")
	}

	// Build INSERT query without values for batch
	var sb strings.Builder
	var args []any

	sb.WriteString("INSERT INTO ")
	args = q.table.AppendTo(&sb, args)

	// ON CLUSTER
	args = q.buildOnClusterClause(&sb, args)

	// Columns
	if len(q.columns) > 0 {
		sb.WriteString(" (")
		for i, col := range q.columns {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(quoteIdentifier(col))
		}
		sb.WriteString(")")
	}

	// SETTINGS
	args = q.buildSettingsClause(&sb, args)

	sql := sb.String()

	// Get the batch interface from the connection
	// This requires the executor to implement PrepareBatch
	if batcher, ok := q.executor.(interface {
		PrepareBatch(ctx context.Context, query string) (driver.Batch, error)
	}); ok {
		return batcher.PrepareBatch(ctx, sql)
	}

	return nil, fmt.Errorf("executor does not support batch operations")
}
