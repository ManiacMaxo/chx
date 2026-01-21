package query

import (
	"strings"
)

// whereClause represents a WHERE or PREWHERE clause.
type whereClause struct {
	conditions []whereCondition
}

// whereCondition represents a single condition in a WHERE clause.
type whereCondition struct {
	separator string // "AND" or "OR"
	expr      QueryWithArgs
	group     *whereClause // For grouped conditions
}

// IsEmpty returns true if the where clause has no conditions.
func (w *whereClause) IsEmpty() bool {
	return len(w.conditions) == 0
}

// addCondition adds a condition with the given separator.
func (w *whereClause) addCondition(separator string, expr string, args ...any) {
	w.conditions = append(w.conditions, whereCondition{
		separator: separator,
		expr:      QueryWithArgs{Query: expr, Args: args},
	})
}

// addGroup adds a grouped condition.
func (w *whereClause) addGroup(separator string, group *whereClause) {
	w.conditions = append(w.conditions, whereCondition{
		separator: separator,
		group:     group,
	})
}

// And adds an AND condition.
func (w *whereClause) And(expr string, args ...any) {
	w.addCondition("AND", expr, args...)
}

// Or adds an OR condition.
func (w *whereClause) Or(expr string, args ...any) {
	w.addCondition("OR", expr, args...)
}

// Build builds the WHERE clause (without the WHERE keyword).
func (w *whereClause) Build(sb *strings.Builder, args []any) []any {
	for i, cond := range w.conditions {
		if i > 0 {
			sb.WriteString(" ")
			sb.WriteString(cond.separator)
			sb.WriteString(" ")
		}

		if cond.group != nil {
			sb.WriteString("(")
			args = cond.group.Build(sb, args)
			sb.WriteString(")")
		} else {
			args = cond.expr.AppendTo(sb, args)
		}
	}
	return args
}

// Clone creates a copy of the where clause.
func (w *whereClause) Clone() *whereClause {
	if w == nil {
		return nil
	}
	clone := &whereClause{
		conditions: make([]whereCondition, len(w.conditions)),
	}
	for i, cond := range w.conditions {
		clone.conditions[i] = whereCondition{
			separator: cond.separator,
			expr:      cond.expr,
		}
		if cond.group != nil {
			clone.conditions[i].group = cond.group.Clone()
		}
	}
	return clone
}

// WhereBuilder provides a fluent interface for building WHERE clauses.
type WhereBuilder struct {
	clause *whereClause
}

// NewWhereBuilder creates a new WHERE clause builder.
func NewWhereBuilder() *WhereBuilder {
	return &WhereBuilder{
		clause: &whereClause{},
	}
}

// Where adds an AND condition.
func (b *WhereBuilder) Where(expr string, args ...any) *WhereBuilder {
	b.clause.And(expr, args...)
	return b
}

// WhereOr adds an OR condition.
func (b *WhereBuilder) WhereOr(expr string, args ...any) *WhereBuilder {
	b.clause.Or(expr, args...)
	return b
}

// WhereIn adds an IN condition.
func (b *WhereBuilder) WhereIn(column string, values any) *WhereBuilder {
	in := InExpr{Values: values}
	expr, inArgs := in.Build()
	b.clause.And(quoteIdentifier(column)+" IN "+expr, inArgs...)
	return b
}

// WhereNotIn adds a NOT IN condition.
func (b *WhereBuilder) WhereNotIn(column string, values any) *WhereBuilder {
	in := InExpr{Values: values}
	expr, inArgs := in.Build()
	b.clause.And(quoteIdentifier(column)+" NOT IN "+expr, inArgs...)
	return b
}

// WhereNull adds an IS NULL condition.
func (b *WhereBuilder) WhereNull(column string) *WhereBuilder {
	b.clause.And(quoteIdentifier(column) + " IS NULL")
	return b
}

// WhereNotNull adds an IS NOT NULL condition.
func (b *WhereBuilder) WhereNotNull(column string) *WhereBuilder {
	b.clause.And(quoteIdentifier(column) + " IS NOT NULL")
	return b
}

// WhereBetween adds a BETWEEN condition.
func (b *WhereBuilder) WhereBetween(column string, from, to any) *WhereBuilder {
	b.clause.And(quoteIdentifier(column)+" BETWEEN ? AND ?", from, to)
	return b
}

// WhereLike adds a LIKE condition.
func (b *WhereBuilder) WhereLike(column string, pattern string) *WhereBuilder {
	b.clause.And(quoteIdentifier(column)+" LIKE ?", pattern)
	return b
}

// WhereILike adds an ILIKE condition (case-insensitive LIKE).
func (b *WhereBuilder) WhereILike(column string, pattern string) *WhereBuilder {
	b.clause.And(quoteIdentifier(column)+" ILIKE ?", pattern)
	return b
}

// WhereGroup adds a grouped condition.
func (b *WhereBuilder) WhereGroup(fn func(*WhereBuilder) *WhereBuilder) *WhereBuilder {
	group := NewWhereBuilder()
	fn(group)
	if !group.clause.IsEmpty() {
		b.clause.addGroup("AND", group.clause)
	}
	return b
}

// WhereOrGroup adds a grouped OR condition.
func (b *WhereBuilder) WhereOrGroup(fn func(*WhereBuilder) *WhereBuilder) *WhereBuilder {
	group := NewWhereBuilder()
	fn(group)
	if !group.clause.IsEmpty() {
		b.clause.addGroup("OR", group.clause)
	}
	return b
}

// WhereExists adds an EXISTS subquery condition.
func (b *WhereBuilder) WhereExists(subquery Builder) *WhereBuilder {
	sql, args, err := subquery.Build()
	if err != nil {
		return b
	}
	b.clause.And("EXISTS ("+sql+")", args...)
	return b
}

// WhereNotExists adds a NOT EXISTS subquery condition.
func (b *WhereBuilder) WhereNotExists(subquery Builder) *WhereBuilder {
	sql, args, err := subquery.Build()
	if err != nil {
		return b
	}
	b.clause.And("NOT EXISTS ("+sql+")", args...)
	return b
}

// Build builds the WHERE clause.
func (b *WhereBuilder) Build() (string, []any) {
	if b.clause.IsEmpty() {
		return "", nil
	}

	var sb strings.Builder
	args := b.clause.Build(&sb, nil)
	return sb.String(), args
}

// IsEmpty returns true if the WHERE clause is empty.
func (b *WhereBuilder) IsEmpty() bool {
	return b.clause.IsEmpty()
}
