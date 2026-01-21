package query

import (
	"strings"
)

// JoinType represents the type of JOIN.
type JoinType int

const (
	JoinTypeInner JoinType = iota
	JoinTypeLeft
	JoinTypeRight
	JoinTypeFull
	JoinTypeCross
)

// JoinStrictness represents JOIN strictness (ANY, ALL, ASOF).
type JoinStrictness int

const (
	JoinStrictnessDefault JoinStrictness = iota
	JoinStrictnessAny
	JoinStrictnessAll
	JoinStrictnessAsof
)

// JoinKind represents JOIN kind (SEMI, ANTI).
type JoinKind int

const (
	JoinKindDefault JoinKind = iota
	JoinKindSemi
	JoinKindAnti
	JoinKindOuter
)

// joinClause represents a JOIN clause.
type joinClause struct {
	global     bool
	strictness JoinStrictness
	kind       JoinKind
	joinType   JoinType
	table      QueryWithArgs
	alias      string
	on         whereClause
	using      []string
}

// JoinBuilder provides a fluent interface for building JOIN clauses.
type JoinBuilder struct {
	parent *SelectQuery
	join   *joinClause
}

// newJoinBuilder creates a new JOIN builder.
func newJoinBuilder(parent *SelectQuery, joinType JoinType, table string) *JoinBuilder {
	return &JoinBuilder{
		parent: parent,
		join: &joinClause{
			joinType: joinType,
			table:    QueryWithArgs{Query: quoteIdentifier(table)},
		},
	}
}

// Global sets the GLOBAL modifier for distributed queries.
func (b *JoinBuilder) Global() *JoinBuilder {
	b.join.global = true
	return b
}

// Any sets the ANY strictness.
func (b *JoinBuilder) Any() *JoinBuilder {
	b.join.strictness = JoinStrictnessAny
	return b
}

// All sets the ALL strictness.
func (b *JoinBuilder) All() *JoinBuilder {
	b.join.strictness = JoinStrictnessAll
	return b
}

// Asof sets the ASOF strictness.
func (b *JoinBuilder) Asof() *JoinBuilder {
	b.join.strictness = JoinStrictnessAsof
	return b
}

// Semi sets the SEMI kind.
func (b *JoinBuilder) Semi() *JoinBuilder {
	b.join.kind = JoinKindSemi
	return b
}

// Anti sets the ANTI kind.
func (b *JoinBuilder) Anti() *JoinBuilder {
	b.join.kind = JoinKindAnti
	return b
}

// Outer sets the OUTER kind.
func (b *JoinBuilder) Outer() *JoinBuilder {
	b.join.kind = JoinKindOuter
	return b
}

// As sets the alias for the joined table.
func (b *JoinBuilder) As(alias string) *JoinBuilder {
	b.join.alias = alias
	return b
}

// On adds an ON condition.
func (b *JoinBuilder) On(expr string, args ...any) *JoinBuilder {
	b.join.on.And(expr, args...)
	return b
}

// OnOr adds an OR ON condition.
func (b *JoinBuilder) OnOr(expr string, args ...any) *JoinBuilder {
	b.join.on.Or(expr, args...)
	return b
}

// Using sets the USING columns.
func (b *JoinBuilder) Using(columns ...string) *JoinBuilder {
	b.join.using = columns
	return b
}

// End returns to the parent SelectQuery.
func (b *JoinBuilder) End() *SelectQuery {
	b.parent.joins = append(b.parent.joins, *b.join)
	return b.parent
}

// SelectQuery returns to the parent SelectQuery (alias for End).
func (b *JoinBuilder) SelectQuery() *SelectQuery {
	return b.End()
}

// Build builds the JOIN clause.
func (j *joinClause) Build(sb *strings.Builder, args []any) []any {
	// GLOBAL
	if j.global {
		sb.WriteString("GLOBAL ")
	}

	// Strictness: ANY, ALL, ASOF
	switch j.strictness {
	case JoinStrictnessAny:
		sb.WriteString("ANY ")
	case JoinStrictnessAll:
		sb.WriteString("ALL ")
	case JoinStrictnessAsof:
		sb.WriteString("ASOF ")
	}

	// Join type: INNER, LEFT, RIGHT, FULL, CROSS
	switch j.joinType {
	case JoinTypeInner:
		sb.WriteString("INNER ")
	case JoinTypeLeft:
		sb.WriteString("LEFT ")
	case JoinTypeRight:
		sb.WriteString("RIGHT ")
	case JoinTypeFull:
		sb.WriteString("FULL ")
	case JoinTypeCross:
		sb.WriteString("CROSS ")
	}

	// Kind: OUTER, SEMI, ANTI
	switch j.kind {
	case JoinKindOuter:
		sb.WriteString("OUTER ")
	case JoinKindSemi:
		sb.WriteString("SEMI ")
	case JoinKindAnti:
		sb.WriteString("ANTI ")
	}

	sb.WriteString("JOIN ")
	args = j.table.AppendTo(sb, args)

	// Alias
	if j.alias != "" {
		sb.WriteString(" AS ")
		sb.WriteString(quoteIdentifier(j.alias))
	}

	// ON clause
	if !j.on.IsEmpty() {
		sb.WriteString(" ON ")
		args = j.on.Build(sb, args)
	}

	// USING clause
	if len(j.using) > 0 {
		sb.WriteString(" USING (")
		for i, col := range j.using {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(quoteIdentifier(col))
		}
		sb.WriteString(")")
	}

	return args
}

// arrayJoinClause represents an ARRAY JOIN clause.
type arrayJoinClause struct {
	left   bool
	column QueryWithArgs
}

// Build builds the ARRAY JOIN clause.
func (j *arrayJoinClause) Build(sb *strings.Builder, args []any) []any {
	if j.left {
		sb.WriteString("LEFT ")
	}
	sb.WriteString("ARRAY JOIN ")
	return j.column.AppendTo(sb, args)
}
