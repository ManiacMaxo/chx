package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// SelectQuery represents a SELECT query builder.
type SelectQuery struct {
	baseQuery

	// Column selection
	columns    []QueryWithArgs
	distinct   bool
	distinctOn []QueryWithArgs
	all        bool

	// Column modifiers
	columnModifiers []columnModifier

	// FROM clause
	from       []QueryWithArgs
	final      bool
	sampleExpr QueryWithArgs
	sampleSeed QueryWithArgs

	// ARRAY JOIN
	arrayJoins []arrayJoinClause

	// JOINs
	joins []joinClause

	// Filtering
	prewhere whereClause
	where    whereClause

	// Grouping
	groupBy    []QueryWithArgs
	withRollup bool
	withCube   bool
	withTotals bool
	having     whereClause

	// Window functions
	windows []windowDef

	// QUALIFY
	qualify QueryWithArgs

	// ORDER BY
	orderBy []orderByClause

	// Pagination
	limit    *int
	offset   *int
	limitBy  *limitByClause
	withTies bool

	// Set operations
	unions []unionClause

	// Output
	intoOutfile   QueryWithArgs
	compression   string
	compressLevel int
	format        string
}

// columnModifier represents APPLY, EXCEPT, REPLACE modifiers.
type columnModifier struct {
	modType string // "APPLY", "EXCEPT", "REPLACE"
	expr    string
}

// windowDef represents a WINDOW clause definition.
type windowDef struct {
	name        string
	partitionBy []QueryWithArgs
	orderBy     []orderByClause
	frame       string
}

// orderByClause represents an ORDER BY clause.
type orderByClause struct {
	column    QueryWithArgs
	direction OrderDirection
	nulls     NullsPosition
	collate   string
	withFill  *withFillClause
}

// withFillClause represents WITH FILL options.
type withFillClause struct {
	from        QueryWithArgs
	to          QueryWithArgs
	step        QueryWithArgs
	interpolate []QueryWithArgs
}

// limitByClause represents a LIMIT BY clause.
type limitByClause struct {
	n       int
	offset  int
	columns []string
}

// unionClause represents a UNION clause.
type unionClause struct {
	all      bool
	distinct bool
	query    Builder
}

// NewSelect creates a new SELECT query builder.
func NewSelect(executor Executor) *SelectQuery {
	return &SelectQuery{
		baseQuery: baseQuery{executor: executor},
	}
}

// Select creates a new SELECT query with columns.
func Select(columns ...string) *SelectQuery {
	q := &SelectQuery{}
	return q.Columns(columns...)
}

// With adds a CTE (Common Table Expression).
func (q *SelectQuery) With(name string, subquery Builder) *SelectQuery {
	q.appendWith(name, false, subquery)
	return q
}

// WithRecursive adds a recursive CTE.
func (q *SelectQuery) WithRecursive(name string, subquery Builder) *SelectQuery {
	q.appendWith(name, true, subquery)
	return q
}

// WithExpr adds a CTE with a raw expression.
func (q *SelectQuery) WithExpr(expr string, args ...any) *SelectQuery {
	q.appendWithExpr(expr, args...)
	return q
}

// Column adds columns to select (safely quoted).
func (q *SelectQuery) Column(columns ...string) *SelectQuery {
	for _, col := range columns {
		q.columns = append(q.columns, QueryWithArgs{Query: quoteIdentifier(col)})
	}
	return q
}

// Columns is an alias for Column.
func (q *SelectQuery) Columns(columns ...string) *SelectQuery {
	return q.Column(columns...)
}

// ColumnExpr adds a raw column expression.
func (q *SelectQuery) ColumnExpr(expr string, args ...any) *SelectQuery {
	q.columns = append(q.columns, QueryWithArgs{Query: expr, Args: args})
	return q
}

// ExcludeColumn adds columns to exclude (for use with *).
func (q *SelectQuery) ExcludeColumn(columns ...string) *SelectQuery {
	for _, col := range columns {
		q.columnModifiers = append(q.columnModifiers, columnModifier{
			modType: "EXCEPT",
			expr:    quoteIdentifier(col),
		})
	}
	return q
}

// Except is an alias for ExcludeColumn.
func (q *SelectQuery) Except(columns ...string) *SelectQuery {
	return q.ExcludeColumn(columns...)
}

// ColumnApply adds an APPLY modifier for column expressions.
func (q *SelectQuery) ColumnApply(fn string) *SelectQuery {
	q.columnModifiers = append(q.columnModifiers, columnModifier{
		modType: "APPLY",
		expr:    fn,
	})
	return q
}

// Replace adds a REPLACE modifier for column expressions.
func (q *SelectQuery) Replace(expr string, as string) *SelectQuery {
	q.columnModifiers = append(q.columnModifiers, columnModifier{
		modType: "REPLACE",
		expr:    fmt.Sprintf("%s AS %s", expr, quoteIdentifier(as)),
	})
	return q
}

// Distinct adds DISTINCT.
func (q *SelectQuery) Distinct() *SelectQuery {
	q.distinct = true
	return q
}

// DistinctOn adds DISTINCT ON columns.
func (q *SelectQuery) DistinctOn(columns ...string) *SelectQuery {
	q.distinct = true
	for _, col := range columns {
		q.distinctOn = append(q.distinctOn, QueryWithArgs{Query: quoteIdentifier(col)})
	}
	return q
}

// All adds ALL modifier.
func (q *SelectQuery) All() *SelectQuery {
	q.all = true
	return q
}

// From adds tables to the FROM clause (safely quoted).
func (q *SelectQuery) From(tables ...string) *SelectQuery {
	for _, table := range tables {
		q.from = append(q.from, QueryWithArgs{Query: quoteIdentifier(table)})
	}
	return q
}

// FromExpr adds a raw FROM expression.
func (q *SelectQuery) FromExpr(expr string, args ...any) *SelectQuery {
	q.from = append(q.from, QueryWithArgs{Query: expr, Args: args})
	return q
}

// FromSubquery adds a subquery as a FROM source.
func (q *SelectQuery) FromSubquery(subquery Builder, alias string) *SelectQuery {
	sql, args, err := subquery.Build()
	if err != nil {
		q.setError(err)
		return q
	}
	q.from = append(q.from, QueryWithArgs{
		Query: fmt.Sprintf("(%s) AS %s", sql, quoteIdentifier(alias)),
		Args:  args,
	})
	return q
}

// Final adds the FINAL modifier (for ReplacingMergeTree, etc.).
func (q *SelectQuery) Final() *SelectQuery {
	q.final = true
	return q
}

// Sample adds a SAMPLE clause.
func (q *SelectQuery) Sample(n float64) *SelectQuery {
	q.sampleExpr = QueryWithArgs{Query: fmt.Sprintf("%v", n)}
	return q
}

// SampleRows adds a SAMPLE clause for a specific number of rows.
func (q *SelectQuery) SampleRows(n int64) *SelectQuery {
	q.sampleExpr = QueryWithArgs{Query: fmt.Sprintf("%d", n)}
	return q
}

// SampleRatio adds a SAMPLE k/m clause.
func (q *SelectQuery) SampleRatio(k, m int64) *SelectQuery {
	q.sampleExpr = QueryWithArgs{Query: fmt.Sprintf("%d/%d", k, m)}
	return q
}

// SampleOffset adds OFFSET to the SAMPLE clause.
func (q *SelectQuery) SampleOffset(offset float64) *SelectQuery {
	q.sampleExpr.Query += fmt.Sprintf(" OFFSET %v", offset)
	return q
}

// SampleSeed adds SEED to the SAMPLE clause.
func (q *SelectQuery) SampleSeed(seed int64) *SelectQuery {
	q.sampleSeed = QueryWithArgs{Query: fmt.Sprintf("%d", seed)}
	return q
}

// ArrayJoin adds an ARRAY JOIN clause.
func (q *SelectQuery) ArrayJoin(column string) *SelectQuery {
	q.arrayJoins = append(q.arrayJoins, arrayJoinClause{
		left:   false,
		column: QueryWithArgs{Query: quoteIdentifier(column)},
	})
	return q
}

// ArrayJoinExpr adds an ARRAY JOIN clause with a raw expression.
func (q *SelectQuery) ArrayJoinExpr(expr string, args ...any) *SelectQuery {
	q.arrayJoins = append(q.arrayJoins, arrayJoinClause{
		left:   false,
		column: QueryWithArgs{Query: expr, Args: args},
	})
	return q
}

// LeftArrayJoin adds a LEFT ARRAY JOIN clause.
func (q *SelectQuery) LeftArrayJoin(column string) *SelectQuery {
	q.arrayJoins = append(q.arrayJoins, arrayJoinClause{
		left:   true,
		column: QueryWithArgs{Query: quoteIdentifier(column)},
	})
	return q
}

// Join starts building an INNER JOIN.
func (q *SelectQuery) Join(table string) *JoinBuilder {
	return newJoinBuilder(q, JoinTypeInner, table)
}

// JoinExpr adds a raw JOIN expression.
func (q *SelectQuery) JoinExpr(expr string, args ...any) *SelectQuery {
	q.joins = append(q.joins, joinClause{
		table: QueryWithArgs{Query: expr, Args: args},
	})
	return q
}

// InnerJoin starts building an INNER JOIN.
func (q *SelectQuery) InnerJoin(table string) *JoinBuilder {
	return newJoinBuilder(q, JoinTypeInner, table)
}

// LeftJoin starts building a LEFT JOIN.
func (q *SelectQuery) LeftJoin(table string) *JoinBuilder {
	return newJoinBuilder(q, JoinTypeLeft, table)
}

// RightJoin starts building a RIGHT JOIN.
func (q *SelectQuery) RightJoin(table string) *JoinBuilder {
	return newJoinBuilder(q, JoinTypeRight, table)
}

// FullJoin starts building a FULL JOIN.
func (q *SelectQuery) FullJoin(table string) *JoinBuilder {
	return newJoinBuilder(q, JoinTypeFull, table)
}

// CrossJoin adds a CROSS JOIN.
func (q *SelectQuery) CrossJoin(table string) *SelectQuery {
	q.joins = append(q.joins, joinClause{
		joinType: JoinTypeCross,
		table:    QueryWithArgs{Query: quoteIdentifier(table)},
	})
	return q
}

// GlobalJoin starts building a GLOBAL JOIN.
func (q *SelectQuery) GlobalJoin(table string) *JoinBuilder {
	jb := newJoinBuilder(q, JoinTypeInner, table)
	jb.join.global = true
	return jb
}

// AnyJoin starts building an ANY JOIN.
func (q *SelectQuery) AnyJoin(table string) *JoinBuilder {
	jb := newJoinBuilder(q, JoinTypeInner, table)
	jb.join.strictness = JoinStrictnessAny
	return jb
}

// AllJoin starts building an ALL JOIN.
func (q *SelectQuery) AllJoin(table string) *JoinBuilder {
	jb := newJoinBuilder(q, JoinTypeInner, table)
	jb.join.strictness = JoinStrictnessAll
	return jb
}

// AsofJoin starts building an ASOF JOIN.
func (q *SelectQuery) AsofJoin(table string) *JoinBuilder {
	jb := newJoinBuilder(q, JoinTypeInner, table)
	jb.join.strictness = JoinStrictnessAsof
	return jb
}

// SemiJoin starts building a SEMI JOIN.
func (q *SelectQuery) SemiJoin(table string) *JoinBuilder {
	jb := newJoinBuilder(q, JoinTypeLeft, table)
	jb.join.kind = JoinKindSemi
	return jb
}

// AntiJoin starts building an ANTI JOIN.
func (q *SelectQuery) AntiJoin(table string) *JoinBuilder {
	jb := newJoinBuilder(q, JoinTypeLeft, table)
	jb.join.kind = JoinKindAnti
	return jb
}

// Prewhere adds a PREWHERE condition (ClickHouse-specific).
func (q *SelectQuery) Prewhere(expr string, args ...any) *SelectQuery {
	q.prewhere.And(expr, args...)
	return q
}

// PrewhereOr adds an OR PREWHERE condition.
func (q *SelectQuery) PrewhereOr(expr string, args ...any) *SelectQuery {
	q.prewhere.Or(expr, args...)
	return q
}

// PrewhereGroup adds a grouped PREWHERE condition.
func (q *SelectQuery) PrewhereGroup(fn func(*SelectQuery) *SelectQuery) *SelectQuery {
	group := &SelectQuery{}
	fn(group)
	if !group.prewhere.IsEmpty() {
		q.prewhere.addGroup("AND", &group.prewhere)
	}
	return q
}

// Where adds a WHERE condition.
func (q *SelectQuery) Where(expr string, args ...any) *SelectQuery {
	q.where.And(expr, args...)
	return q
}

// WhereOr adds an OR WHERE condition.
func (q *SelectQuery) WhereOr(expr string, args ...any) *SelectQuery {
	q.where.Or(expr, args...)
	return q
}

// WhereGroup adds a grouped WHERE condition.
func (q *SelectQuery) WhereGroup(fn func(*SelectQuery) *SelectQuery) *SelectQuery {
	group := &SelectQuery{}
	fn(group)
	if !group.where.IsEmpty() {
		q.where.addGroup("AND", &group.where)
	}
	return q
}

// WhereOrGroup adds a grouped OR WHERE condition.
func (q *SelectQuery) WhereOrGroup(fn func(*SelectQuery) *SelectQuery) *SelectQuery {
	group := &SelectQuery{}
	fn(group)
	if !group.where.IsEmpty() {
		q.where.addGroup("OR", &group.where)
	}
	return q
}

// WhereIn adds an IN condition.
func (q *SelectQuery) WhereIn(column string, values any) *SelectQuery {
	in := InExpr{Values: values}
	expr, inArgs := in.Build()
	q.where.And(quoteIdentifier(column)+" IN "+expr, inArgs...)
	return q
}

// WhereNotIn adds a NOT IN condition.
func (q *SelectQuery) WhereNotIn(column string, values any) *SelectQuery {
	in := InExpr{Values: values}
	expr, inArgs := in.Build()
	q.where.And(quoteIdentifier(column)+" NOT IN "+expr, inArgs...)
	return q
}

// GroupBy adds columns to GROUP BY (safely quoted).
func (q *SelectQuery) GroupBy(columns ...string) *SelectQuery {
	for _, col := range columns {
		q.groupBy = append(q.groupBy, QueryWithArgs{Query: quoteIdentifier(col)})
	}
	return q
}

// GroupByExpr adds a raw GROUP BY expression.
func (q *SelectQuery) GroupByExpr(expr string, args ...any) *SelectQuery {
	q.groupBy = append(q.groupBy, QueryWithArgs{Query: expr, Args: args})
	return q
}

// WithRollup adds WITH ROLLUP modifier.
func (q *SelectQuery) WithRollup() *SelectQuery {
	q.withRollup = true
	return q
}

// WithCube adds WITH CUBE modifier.
func (q *SelectQuery) WithCube() *SelectQuery {
	q.withCube = true
	return q
}

// WithTotals adds WITH TOTALS modifier.
func (q *SelectQuery) WithTotals() *SelectQuery {
	q.withTotals = true
	return q
}

// Having adds a HAVING condition.
func (q *SelectQuery) Having(expr string, args ...any) *SelectQuery {
	q.having.And(expr, args...)
	return q
}

// HavingOr adds an OR HAVING condition.
func (q *SelectQuery) HavingOr(expr string, args ...any) *SelectQuery {
	q.having.Or(expr, args...)
	return q
}

// Window adds a named window definition.
func (q *SelectQuery) Window(name string) *WindowBuilder {
	return &WindowBuilder{
		parent: q,
		def: windowDef{
			name: name,
		},
	}
}

// Qualify adds a QUALIFY clause.
func (q *SelectQuery) Qualify(expr string, args ...any) *SelectQuery {
	q.qualify = QueryWithArgs{Query: expr, Args: args}
	return q
}

// OrderBy adds columns to ORDER BY (ASC by default).
func (q *SelectQuery) OrderBy(columns ...string) *SelectQuery {
	for _, col := range columns {
		q.orderBy = append(q.orderBy, orderByClause{
			column:    QueryWithArgs{Query: quoteIdentifier(col)},
			direction: Asc,
		})
	}
	return q
}

// OrderByDesc adds columns to ORDER BY DESC.
func (q *SelectQuery) OrderByDesc(columns ...string) *SelectQuery {
	for _, col := range columns {
		q.orderBy = append(q.orderBy, orderByClause{
			column:    QueryWithArgs{Query: quoteIdentifier(col)},
			direction: Desc,
		})
	}
	return q
}

// OrderByExpr adds a raw ORDER BY expression.
func (q *SelectQuery) OrderByExpr(expr string, args ...any) *SelectQuery {
	q.orderBy = append(q.orderBy, orderByClause{
		column: QueryWithArgs{Query: expr, Args: args},
	})
	return q
}

// OrderByWithFill starts building an ORDER BY WITH FILL clause.
func (q *SelectQuery) OrderByWithFill(column string) *OrderByFillBuilder {
	clause := &orderByClause{
		column:   QueryWithArgs{Query: quoteIdentifier(column)},
		withFill: &withFillClause{},
	}
	return &OrderByFillBuilder{
		parent: q,
		clause: clause,
	}
}

// Limit sets the LIMIT.
func (q *SelectQuery) Limit(n int) *SelectQuery {
	q.limit = &n
	return q
}

// Offset sets the OFFSET.
func (q *SelectQuery) Offset(n int) *SelectQuery {
	q.offset = &n
	return q
}

// WithTies adds WITH TIES to LIMIT.
func (q *SelectQuery) WithTies() *SelectQuery {
	q.withTies = true
	return q
}

// LimitBy adds a LIMIT BY clause (ClickHouse-specific).
func (q *SelectQuery) LimitBy(n int, columns ...string) *SelectQuery {
	q.limitBy = &limitByClause{
		n:       n,
		columns: columns,
	}
	return q
}

// LimitByOffset adds a LIMIT BY with offset clause.
func (q *SelectQuery) LimitByOffset(n, offset int, columns ...string) *SelectQuery {
	q.limitBy = &limitByClause{
		n:       n,
		offset:  offset,
		columns: columns,
	}
	return q
}

// Union adds a UNION.
func (q *SelectQuery) Union(query Builder) *SelectQuery {
	q.unions = append(q.unions, unionClause{query: query})
	return q
}

// UnionAll adds a UNION ALL.
func (q *SelectQuery) UnionAll(query Builder) *SelectQuery {
	q.unions = append(q.unions, unionClause{all: true, query: query})
	return q
}

// UnionDistinct adds a UNION DISTINCT.
func (q *SelectQuery) UnionDistinct(query Builder) *SelectQuery {
	q.unions = append(q.unions, unionClause{distinct: true, query: query})
	return q
}

// Intersect adds an INTERSECT.
func (q *SelectQuery) Intersect(query Builder) *SelectQuery {
	q.unions = append(q.unions, unionClause{query: query})
	return q
}

// ExceptQuery adds an EXCEPT (named to avoid collision with Except column modifier).
func (q *SelectQuery) ExceptQuery(query Builder) *SelectQuery {
	q.unions = append(q.unions, unionClause{query: query})
	return q
}

// Setting adds a SETTINGS clause.
func (q *SelectQuery) Setting(setting string) *SelectQuery {
	q.appendSetting(setting)
	return q
}

// SettingExpr adds a SETTINGS clause with arguments.
func (q *SelectQuery) SettingExpr(expr string, args ...any) *SelectQuery {
	q.appendSetting(expr, args...)
	return q
}

// Settings adds multiple settings.
func (q *SelectQuery) Settings(settings map[string]any) *SelectQuery {
	for k, v := range settings {
		q.appendSetting(fmt.Sprintf("%s = ?", k), v)
	}
	return q
}

// IntoOutfile adds an INTO OUTFILE clause.
func (q *SelectQuery) IntoOutfile(filename string) *SelectQuery {
	q.intoOutfile = QueryWithArgs{Query: "'" + escapeString(filename) + "'"}
	return q
}

// Compression adds COMPRESSION to INTO OUTFILE.
func (q *SelectQuery) Compression(typ string, level int) *SelectQuery {
	q.compression = typ
	q.compressLevel = level
	return q
}

// Format adds a FORMAT clause.
func (q *SelectQuery) Format(format string) *SelectQuery {
	q.format = format
	return q
}

// Clone creates a copy of the query.
func (q *SelectQuery) Clone() *SelectQuery {
	clone := *q
	clone.columns = append([]QueryWithArgs{}, q.columns...)
	clone.from = append([]QueryWithArgs{}, q.from...)
	clone.joins = append([]joinClause{}, q.joins...)
	clone.groupBy = append([]QueryWithArgs{}, q.groupBy...)
	clone.orderBy = append([]orderByClause{}, q.orderBy...)
	clone.with = append([]withClause{}, q.with...)
	clone.settings = append([]QueryWithArgs{}, q.settings...)
	return &clone
}

// Apply applies a function to the query.
func (q *SelectQuery) Apply(fn func(*SelectQuery) *SelectQuery) *SelectQuery {
	return fn(q)
}

// Build builds the SELECT query.
func (q *SelectQuery) Build() (string, []any, error) {
	if q.err != nil {
		return "", nil, q.err
	}

	var sb strings.Builder
	var args []any
	var err error

	// WITH clause
	args, err = q.buildWithClause(&sb, args)
	if err != nil {
		return "", nil, err
	}

	// SELECT
	sb.WriteString("SELECT ")

	// DISTINCT
	if q.distinct {
		sb.WriteString("DISTINCT ")
		if len(q.distinctOn) > 0 {
			sb.WriteString("ON (")
			for i, col := range q.distinctOn {
				if i > 0 {
					sb.WriteString(", ")
				}
				args = col.AppendTo(&sb, args)
			}
			sb.WriteString(") ")
		}
	}

	// ALL
	if q.all {
		sb.WriteString("ALL ")
	}

	// Columns
	if len(q.columns) == 0 {
		sb.WriteString("*")
	} else {
		for i, col := range q.columns {
			if i > 0 {
				sb.WriteString(", ")
			}
			args = col.AppendTo(&sb, args)
		}
	}

	// Column modifiers (APPLY, EXCEPT, REPLACE)
	for _, mod := range q.columnModifiers {
		sb.WriteString(" ")
		sb.WriteString(mod.modType)
		sb.WriteString("(")
		sb.WriteString(mod.expr)
		sb.WriteString(")")
	}

	// FROM clause
	if len(q.from) > 0 {
		sb.WriteString(" FROM ")
		for i, from := range q.from {
			if i > 0 {
				sb.WriteString(", ")
			}
			args = from.AppendTo(&sb, args)
		}
	}

	// FINAL
	if q.final {
		sb.WriteString(" FINAL")
	}

	// SAMPLE
	if !q.sampleExpr.IsEmpty() {
		sb.WriteString(" SAMPLE ")
		args = q.sampleExpr.AppendTo(&sb, args)
		if !q.sampleSeed.IsEmpty() {
			sb.WriteString(" SEED ")
			args = q.sampleSeed.AppendTo(&sb, args)
		}
	}

	// ARRAY JOIN
	for _, aj := range q.arrayJoins {
		sb.WriteString(" ")
		args = aj.Build(&sb, args)
	}

	// JOINs
	for _, j := range q.joins {
		sb.WriteString(" ")
		args = j.Build(&sb, args)
	}

	// PREWHERE
	if !q.prewhere.IsEmpty() {
		sb.WriteString(" PREWHERE ")
		args = q.prewhere.Build(&sb, args)
	}

	// WHERE
	if !q.where.IsEmpty() {
		sb.WriteString(" WHERE ")
		args = q.where.Build(&sb, args)
	}

	// GROUP BY
	if len(q.groupBy) > 0 {
		sb.WriteString(" GROUP BY ")
		for i, g := range q.groupBy {
			if i > 0 {
				sb.WriteString(", ")
			}
			args = g.AppendTo(&sb, args)
		}
		if q.withRollup {
			sb.WriteString(" WITH ROLLUP")
		}
		if q.withCube {
			sb.WriteString(" WITH CUBE")
		}
		if q.withTotals {
			sb.WriteString(" WITH TOTALS")
		}
	}

	// HAVING
	if !q.having.IsEmpty() {
		sb.WriteString(" HAVING ")
		args = q.having.Build(&sb, args)
	}

	// WINDOW
	if len(q.windows) > 0 {
		sb.WriteString(" WINDOW ")
		for i, w := range q.windows {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(w.name)
			sb.WriteString(" AS (")
			args = q.buildWindowDef(&sb, args, &w)
			sb.WriteString(")")
		}
	}

	// QUALIFY
	if !q.qualify.IsEmpty() {
		sb.WriteString(" QUALIFY ")
		args = q.qualify.AppendTo(&sb, args)
	}

	// ORDER BY
	if len(q.orderBy) > 0 {
		sb.WriteString(" ORDER BY ")
		for i, o := range q.orderBy {
			if i > 0 {
				sb.WriteString(", ")
			}
			args = o.column.AppendTo(&sb, args)
			if o.direction == Desc {
				sb.WriteString(" DESC")
			}
			switch o.nulls {
			case NullsFirst:
				sb.WriteString(" NULLS FIRST")
			case NullsLast:
				sb.WriteString(" NULLS LAST")
			}
			if o.collate != "" {
				sb.WriteString(" COLLATE ")
				sb.WriteString(o.collate)
			}
			if o.withFill != nil {
				sb.WriteString(" WITH FILL")
				if !o.withFill.from.IsEmpty() {
					sb.WriteString(" FROM ")
					args = o.withFill.from.AppendTo(&sb, args)
				}
				if !o.withFill.to.IsEmpty() {
					sb.WriteString(" TO ")
					args = o.withFill.to.AppendTo(&sb, args)
				}
				if !o.withFill.step.IsEmpty() {
					sb.WriteString(" STEP ")
					args = o.withFill.step.AppendTo(&sb, args)
				}
				if len(o.withFill.interpolate) > 0 {
					sb.WriteString(" INTERPOLATE (")
					for j, interp := range o.withFill.interpolate {
						if j > 0 {
							sb.WriteString(", ")
						}
						args = interp.AppendTo(&sb, args)
					}
					sb.WriteString(")")
				}
			}
		}
	}

	// LIMIT BY
	if q.limitBy != nil {
		sb.WriteString(" LIMIT ")
		if q.limitBy.offset > 0 {
			sb.WriteString(fmt.Sprintf("%d, ", q.limitBy.offset))
		}
		sb.WriteString(fmt.Sprintf("%d BY ", q.limitBy.n))
		for i, col := range q.limitBy.columns {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(quoteIdentifier(col))
		}
	}

	// LIMIT
	if q.limit != nil {
		sb.WriteString(" LIMIT ")
		if q.offset != nil && *q.offset > 0 {
			sb.WriteString(fmt.Sprintf("%d, ", *q.offset))
		}
		sb.WriteString(fmt.Sprintf("%d", *q.limit))
		if q.withTies {
			sb.WriteString(" WITH TIES")
		}
	} else if q.offset != nil && *q.offset > 0 {
		sb.WriteString(fmt.Sprintf(" OFFSET %d", *q.offset))
	}

	// UNION / INTERSECT / EXCEPT
	for _, u := range q.unions {
		if u.all {
			sb.WriteString(" UNION ALL ")
		} else if u.distinct {
			sb.WriteString(" UNION DISTINCT ")
		} else {
			sb.WriteString(" UNION ")
		}
		sql, unionArgs, err := u.query.Build()
		if err != nil {
			return "", nil, err
		}
		sb.WriteString(sql)
		args = append(args, unionArgs...)
	}

	// INTO OUTFILE
	if !q.intoOutfile.IsEmpty() {
		sb.WriteString(" INTO OUTFILE ")
		args = q.intoOutfile.AppendTo(&sb, args)
		if q.compression != "" {
			sb.WriteString(" COMPRESSION ")
			sb.WriteString(q.compression)
			if q.compressLevel > 0 {
				sb.WriteString(fmt.Sprintf(" LEVEL %d", q.compressLevel))
			}
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

// buildWindowDef builds a window definition.
func (q *SelectQuery) buildWindowDef(sb *strings.Builder, args []any, w *windowDef) []any {
	if len(w.partitionBy) > 0 {
		sb.WriteString("PARTITION BY ")
		for i, p := range w.partitionBy {
			if i > 0 {
				sb.WriteString(", ")
			}
			args = p.AppendTo(sb, args)
		}
	}
	if len(w.orderBy) > 0 {
		if len(w.partitionBy) > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString("ORDER BY ")
		for i, o := range w.orderBy {
			if i > 0 {
				sb.WriteString(", ")
			}
			args = o.column.AppendTo(sb, args)
			if o.direction == Desc {
				sb.WriteString(" DESC")
			}
		}
	}
	if w.frame != "" {
		sb.WriteString(" ")
		sb.WriteString(w.frame)
	}
	return args
}

// String returns the SQL query with arguments interpolated (for debugging).
func (q *SelectQuery) String() string {
	sql, args, err := q.Build()
	if err != nil {
		return fmt.Sprintf("ERROR: %v", err)
	}
	return interpolateArgs(sql, args)
}

// Query executes the query and returns rows.
func (q *SelectQuery) Query(ctx context.Context) (driver.Rows, error) {
	if q.executor == nil {
		return nil, fmt.Errorf("no executor set")
	}
	sql, args, err := q.Build()
	if err != nil {
		return nil, err
	}
	return q.executor.Query(ctx, sql, args...)
}

// QueryRow executes the query and returns a single row.
func (q *SelectQuery) QueryRow(ctx context.Context) driver.Row {
	if q.executor == nil {
		return nil
	}
	sql, args, err := q.Build()
	if err != nil {
		return nil
	}
	return q.executor.QueryRow(ctx, sql, args...)
}

// Scan executes the query and scans the result into dest.
func (q *SelectQuery) Scan(ctx context.Context, dest ...any) error {
	row := q.QueryRow(ctx)
	if row == nil {
		return fmt.Errorf("no executor set")
	}
	return row.Scan(dest...)
}

// Count returns the count of rows.
func (q *SelectQuery) Count(ctx context.Context) (uint64, error) {
	countQuery := q.Clone()
	countQuery.columns = []QueryWithArgs{{Query: "count(*)"}}
	countQuery.orderBy = nil
	countQuery.limit = nil
	countQuery.offset = nil

	var count uint64
	err := countQuery.Scan(ctx, &count)
	return count, err
}

// WindowBuilder builds a WINDOW clause.
type WindowBuilder struct {
	parent *SelectQuery
	def    windowDef
}

// PartitionBy adds PARTITION BY columns.
func (b *WindowBuilder) PartitionBy(columns ...string) *WindowBuilder {
	for _, col := range columns {
		b.def.partitionBy = append(b.def.partitionBy, QueryWithArgs{Query: quoteIdentifier(col)})
	}
	return b
}

// OrderBy adds ORDER BY columns.
func (b *WindowBuilder) OrderBy(columns ...string) *WindowBuilder {
	for _, col := range columns {
		b.def.orderBy = append(b.def.orderBy, orderByClause{
			column: QueryWithArgs{Query: quoteIdentifier(col)},
		})
	}
	return b
}

// Frame sets the window frame.
func (b *WindowBuilder) Frame(frame string) *WindowBuilder {
	b.def.frame = frame
	return b
}

// End returns to the parent SelectQuery.
func (b *WindowBuilder) End() *SelectQuery {
	b.parent.windows = append(b.parent.windows, b.def)
	return b.parent
}

// OrderByFillBuilder builds an ORDER BY WITH FILL clause.
type OrderByFillBuilder struct {
	parent *SelectQuery
	clause *orderByClause
}

// Desc sets descending order.
func (b *OrderByFillBuilder) Desc() *OrderByFillBuilder {
	b.clause.direction = Desc
	return b
}

// From sets the FROM value for WITH FILL.
func (b *OrderByFillBuilder) From(value any) *OrderByFillBuilder {
	b.clause.withFill.from = QueryWithArgs{Query: "?", Args: []any{value}}
	return b
}

// To sets the TO value for WITH FILL.
func (b *OrderByFillBuilder) To(value any) *OrderByFillBuilder {
	b.clause.withFill.to = QueryWithArgs{Query: "?", Args: []any{value}}
	return b
}

// Step sets the STEP value for WITH FILL.
func (b *OrderByFillBuilder) Step(value any) *OrderByFillBuilder {
	b.clause.withFill.step = QueryWithArgs{Query: "?", Args: []any{value}}
	return b
}

// Interpolate adds INTERPOLATE expressions.
func (b *OrderByFillBuilder) Interpolate(exprs ...string) *OrderByFillBuilder {
	for _, expr := range exprs {
		b.clause.withFill.interpolate = append(b.clause.withFill.interpolate, QueryWithArgs{Query: expr})
	}
	return b
}

// End returns to the parent SelectQuery.
func (b *OrderByFillBuilder) End() *SelectQuery {
	b.parent.orderBy = append(b.parent.orderBy, *b.clause)
	return b.parent
}
