package query

import (
	"fmt"
	"reflect"
	"strings"
)

// Identifier represents a safely quoted identifier.
type Identifier string

// Ident creates a quoted identifier.
func Ident(name string) Identifier {
	return Identifier(name)
}

// SafeString represents a string that should not be escaped.
type SafeString string

// Safe marks a string as safe (no escaping).
func Safe(s string) SafeString {
	return SafeString(s)
}

// RawExpr represents a raw SQL expression.
type RawExpr string

// Raw creates a raw SQL expression (use with caution).
func Raw(sql string) RawExpr {
	return RawExpr(sql)
}

// InExpr represents an IN clause expression.
type InExpr struct {
	Values any
}

// In creates an IN clause from a slice.
func In[T any](values []T) InExpr {
	return InExpr{Values: values}
}

// Build builds the IN expression.
func (e InExpr) Build() (string, []any) {
	v := reflect.ValueOf(e.Values)
	if v.Kind() != reflect.Slice {
		return "?", []any{e.Values}
	}

	if v.Len() == 0 {
		return "()", nil
	}

	placeholders := make([]string, v.Len())
	args := make([]any, v.Len())
	for i := 0; i < v.Len(); i++ {
		placeholders[i] = "?"
		args[i] = v.Index(i).Interface()
	}

	return "(" + strings.Join(placeholders, ", ") + ")", args
}

// NotInExpr represents a NOT IN clause expression.
type NotInExpr struct {
	Values any
}

// NotIn creates a NOT IN clause from a slice.
func NotIn[T any](values []T) NotInExpr {
	return NotInExpr{Values: values}
}

// Build builds the NOT IN expression.
func (e NotInExpr) Build() (string, []any) {
	in := InExpr{Values: e.Values}
	return in.Build()
}

// ArrayExpr represents a ClickHouse array literal.
type ArrayExpr struct {
	Values any
}

// Array creates a ClickHouse array literal.
func Array[T any](values []T) ArrayExpr {
	return ArrayExpr{Values: values}
}

// Build builds the array expression.
func (e ArrayExpr) Build() (string, []any) {
	v := reflect.ValueOf(e.Values)
	if v.Kind() != reflect.Slice {
		return "[?]", []any{e.Values}
	}

	if v.Len() == 0 {
		return "[]", nil
	}

	placeholders := make([]string, v.Len())
	args := make([]any, v.Len())
	for i := 0; i < v.Len(); i++ {
		placeholders[i] = "?"
		args[i] = v.Index(i).Interface()
	}

	return "[" + strings.Join(placeholders, ", ") + "]", args
}

// TupleExpr represents a ClickHouse tuple literal.
type TupleExpr struct {
	Values []any
}

// Tuple creates a ClickHouse tuple literal.
func Tuple(values ...any) TupleExpr {
	return TupleExpr{Values: values}
}

// Build builds the tuple expression.
func (e TupleExpr) Build() (string, []any) {
	if len(e.Values) == 0 {
		return "()", nil
	}

	placeholders := make([]string, len(e.Values))
	for i := range e.Values {
		placeholders[i] = "?"
	}

	return "(" + strings.Join(placeholders, ", ") + ")", e.Values
}

// NamedArg represents a named parameter.
type NamedArg struct {
	Name  string
	Value any
}

// Named creates a named parameter.
func Named(name string, value any) NamedArg {
	return NamedArg{Name: name, Value: value}
}

// SubqueryExpr wraps a Builder for use as a subquery.
type SubqueryExpr struct {
	Query Builder
}

// Subquery wraps a query builder for use in expressions.
func Subquery(query Builder) SubqueryExpr {
	return SubqueryExpr{Query: query}
}

// Build builds the subquery expression.
func (e SubqueryExpr) Build() (string, []any, error) {
	if e.Query == nil {
		return "()", nil, nil
	}
	sql, args, err := e.Query.Build()
	if err != nil {
		return "", nil, err
	}
	return "(" + sql + ")", args, nil
}

// OrderDirection represents an ORDER BY direction.
type OrderDirection int

const (
	// Asc represents ascending order.
	Asc OrderDirection = iota
	// Desc represents descending order.
	Desc
)

// NullsPosition represents NULLS FIRST/LAST.
type NullsPosition int

const (
	// NullsDefault uses default nulls ordering.
	NullsDefault NullsPosition = iota
	// NullsFirst puts nulls first.
	NullsFirst
	// NullsLast puts nulls last.
	NullsLast
)

// OrderColumn represents an ORDER BY column with direction.
type OrderColumn struct {
	Column    string
	Direction OrderDirection
	Nulls     NullsPosition
	Collate   string
}

// Columns creates a list of quoted column names.
func Columns(names ...string) string {
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = quoteIdentifier(name)
	}
	return strings.Join(quoted, ", ")
}

// ExprBuilder helps build complex expressions.
type ExprBuilder struct {
	parts []string
	args  []any
}

// NewExpr creates a new expression builder.
func NewExpr(expr string, args ...any) *ExprBuilder {
	return &ExprBuilder{
		parts: []string{expr},
		args:  args,
	}
}

// And adds an AND condition.
func (e *ExprBuilder) And(expr string, args ...any) *ExprBuilder {
	e.parts = append(e.parts, expr)
	e.args = append(e.args, args...)
	return e
}

// Or adds an OR condition.
func (e *ExprBuilder) Or(expr string, args ...any) *ExprBuilder {
	e.parts = append(e.parts, expr)
	e.args = append(e.args, args...)
	return e
}

// Build builds the expression.
func (e *ExprBuilder) Build(separator string) (string, []any) {
	return strings.Join(e.parts, separator), e.args
}

// Expr creates a QueryWithArgs from an expression string and arguments.
func Expr(expr string, args ...any) QueryWithArgs {
	return QueryWithArgs{Query: expr, Args: args}
}

// Placeholder constants for convenience
const (
	PlaceholderQuestion = "?"
)

// FormatPlaceholders creates n placeholders separated by commas.
func FormatPlaceholders(n int) string {
	if n <= 0 {
		return ""
	}
	placeholders := make([]string, n)
	for i := range placeholders {
		placeholders[i] = "?"
	}
	return strings.Join(placeholders, ", ")
}

// AliasExpr represents an aliased expression.
type AliasExpr struct {
	Expr  string
	Alias string
	Args  []any
}

// As creates an aliased expression.
func As(expr string, alias string, args ...any) AliasExpr {
	return AliasExpr{Expr: expr, Alias: alias, Args: args}
}

// Build builds the aliased expression.
func (e AliasExpr) Build() (string, []any) {
	return fmt.Sprintf("%s AS %s", e.Expr, quoteIdentifier(e.Alias)), e.Args
}
