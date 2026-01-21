package query

import (
	"context"
	"fmt"
	"strings"
)

// AlterQuery represents an ALTER TABLE query builder.
type AlterQuery struct {
	baseQuery

	table       QueryWithArgs
	alterations []QueryWithArgs
}

// NewAlter creates a new ALTER TABLE query builder.
func NewAlter(executor Executor) *AlterQuery {
	return &AlterQuery{
		baseQuery: baseQuery{executor: executor},
	}
}

// Alter creates a new ALTER TABLE query.
func Alter(table string) *AlterQuery {
	return &AlterQuery{
		table: QueryWithArgs{Query: quoteIdentifier(table)},
	}
}

// Table sets the table to alter.
func (q *AlterQuery) Table(table string) *AlterQuery {
	q.table = QueryWithArgs{Query: quoteIdentifier(table)}
	return q
}

// TableExpr sets the table with a raw expression.
func (q *AlterQuery) TableExpr(expr string, args ...any) *AlterQuery {
	q.table = QueryWithArgs{Query: expr, Args: args}
	return q
}

// OnCluster adds an ON CLUSTER clause.
func (q *AlterQuery) OnCluster(cluster string) *AlterQuery {
	q.onCluster = QueryWithArgs{Query: quoteIdentifier(cluster)}
	return q
}

// addAlteration adds an alteration to the query.
func (q *AlterQuery) addAlteration(alteration string, args ...any) {
	q.alterations = append(q.alterations, QueryWithArgs{Query: alteration, Args: args})
}

// AddColumn adds a column to the table.
func (q *AlterQuery) AddColumn(name, typ string) *AlterColumnBuilder {
	return &AlterColumnBuilder{
		parent: q,
		action: "ADD COLUMN",
		name:   name,
		typ:    typ,
	}
}

// DropColumn drops a column from the table.
func (q *AlterQuery) DropColumn(name string) *AlterQuery {
	q.addAlteration("DROP COLUMN " + quoteIdentifier(name))
	return q
}

// RenameColumn renames a column.
func (q *AlterQuery) RenameColumn(from, to string) *AlterQuery {
	q.addAlteration(fmt.Sprintf("RENAME COLUMN %s TO %s", quoteIdentifier(from), quoteIdentifier(to)))
	return q
}

// ModifyColumn modifies a column type.
func (q *AlterQuery) ModifyColumn(name, typ string) *AlterQuery {
	q.addAlteration(fmt.Sprintf("MODIFY COLUMN %s %s", quoteIdentifier(name), typ))
	return q
}

// ClearColumn clears a column in a partition.
func (q *AlterQuery) ClearColumn(name string) *AlterQuery {
	q.addAlteration("CLEAR COLUMN " + quoteIdentifier(name))
	return q
}

// ClearColumnInPartition clears a column in a specific partition.
func (q *AlterQuery) ClearColumnInPartition(name, partition string) *AlterQuery {
	q.addAlteration(fmt.Sprintf("CLEAR COLUMN %s IN PARTITION %s", quoteIdentifier(name), partition))
	return q
}

// CommentColumn sets a column comment.
func (q *AlterQuery) CommentColumn(name, comment string) *AlterQuery {
	q.addAlteration(fmt.Sprintf("COMMENT COLUMN %s '%s'", quoteIdentifier(name), escapeString(comment)))
	return q
}

// AddIndex adds an index.
func (q *AlterQuery) AddIndex(name, expr, typ string, granularity ...int) *AlterQuery {
	alteration := fmt.Sprintf("ADD INDEX %s %s TYPE %s", quoteIdentifier(name), expr, typ)
	if len(granularity) > 0 {
		alteration += fmt.Sprintf(" GRANULARITY %d", granularity[0])
	}
	q.addAlteration(alteration)
	return q
}

// DropIndex drops an index.
func (q *AlterQuery) DropIndex(name string) *AlterQuery {
	q.addAlteration("DROP INDEX " + quoteIdentifier(name))
	return q
}

// MaterializeIndex materializes an index.
func (q *AlterQuery) MaterializeIndex(name string) *AlterQuery {
	q.addAlteration("MATERIALIZE INDEX " + quoteIdentifier(name))
	return q
}

// ClearIndex clears an index in a partition.
func (q *AlterQuery) ClearIndex(name, partition string) *AlterQuery {
	q.addAlteration(fmt.Sprintf("CLEAR INDEX %s IN PARTITION %s", quoteIdentifier(name), partition))
	return q
}

// AddConstraint adds a constraint.
func (q *AlterQuery) AddConstraint(name, expr string) *AlterQuery {
	q.addAlteration(fmt.Sprintf("ADD CONSTRAINT %s CHECK %s", quoteIdentifier(name), expr))
	return q
}

// DropConstraint drops a constraint.
func (q *AlterQuery) DropConstraint(name string) *AlterQuery {
	q.addAlteration("DROP CONSTRAINT " + quoteIdentifier(name))
	return q
}

// AddProjection adds a projection.
func (q *AlterQuery) AddProjection(name, selectExpr string) *AlterQuery {
	q.addAlteration(fmt.Sprintf("ADD PROJECTION %s (%s)", quoteIdentifier(name), selectExpr))
	return q
}

// DropProjection drops a projection.
func (q *AlterQuery) DropProjection(name string) *AlterQuery {
	q.addAlteration("DROP PROJECTION " + quoteIdentifier(name))
	return q
}

// MaterializeProjection materializes a projection.
func (q *AlterQuery) MaterializeProjection(name string) *AlterQuery {
	q.addAlteration("MATERIALIZE PROJECTION " + quoteIdentifier(name))
	return q
}

// ClearProjection clears a projection in a partition.
func (q *AlterQuery) ClearProjection(name, partition string) *AlterQuery {
	q.addAlteration(fmt.Sprintf("CLEAR PROJECTION %s IN PARTITION %s", quoteIdentifier(name), partition))
	return q
}

// DropPartition drops a partition.
func (q *AlterQuery) DropPartition(partition string) *AlterQuery {
	q.addAlteration("DROP PARTITION " + partition)
	return q
}

// DetachPartition detaches a partition.
func (q *AlterQuery) DetachPartition(partition string) *AlterQuery {
	q.addAlteration("DETACH PARTITION " + partition)
	return q
}

// AttachPartition attaches a partition.
func (q *AlterQuery) AttachPartition(partition string) *AlterQuery {
	q.addAlteration("ATTACH PARTITION " + partition)
	return q
}

// MovePartition moves a partition to another table.
func (q *AlterQuery) MovePartition(partition, toTable string) *AlterQuery {
	q.addAlteration(fmt.Sprintf("MOVE PARTITION %s TO TABLE %s", partition, quoteIdentifier(toTable)))
	return q
}

// ReplacePartition replaces a partition from another table.
func (q *AlterQuery) ReplacePartition(partition, fromTable string) *AlterQuery {
	q.addAlteration(fmt.Sprintf("REPLACE PARTITION %s FROM %s", partition, quoteIdentifier(fromTable)))
	return q
}

// FreezePartition freezes a partition (for backups).
func (q *AlterQuery) FreezePartition(partition string) *AlterQuery {
	if partition == "" {
		q.addAlteration("FREEZE")
	} else {
		q.addAlteration("FREEZE PARTITION " + partition)
	}
	return q
}

// UnfreezePartition unfreezes a partition.
func (q *AlterQuery) UnfreezePartition(partition string) *AlterQuery {
	if partition == "" {
		q.addAlteration("UNFREEZE")
	} else {
		q.addAlteration("UNFREEZE PARTITION " + partition)
	}
	return q
}

// FetchPartition fetches a partition from a replica.
func (q *AlterQuery) FetchPartition(partition, fromPath string) *AlterQuery {
	q.addAlteration(fmt.Sprintf("FETCH PARTITION %s FROM '%s'", partition, escapeString(fromPath)))
	return q
}

// ModifyTTL modifies the table TTL.
func (q *AlterQuery) ModifyTTL(expr string) *AlterQuery {
	q.addAlteration("MODIFY TTL " + expr)
	return q
}

// RemoveTTL removes the table TTL.
func (q *AlterQuery) RemoveTTL() *AlterQuery {
	q.addAlteration("REMOVE TTL")
	return q
}

// ModifyOrderBy modifies the ORDER BY expression.
func (q *AlterQuery) ModifyOrderBy(expr string) *AlterQuery {
	q.addAlteration("MODIFY ORDER BY " + expr)
	return q
}

// ModifySampleBy modifies the SAMPLE BY expression.
func (q *AlterQuery) ModifySampleBy(expr string) *AlterQuery {
	q.addAlteration("MODIFY SAMPLE BY " + expr)
	return q
}

// RemoveSampleBy removes the SAMPLE BY expression.
func (q *AlterQuery) RemoveSampleBy() *AlterQuery {
	q.addAlteration("REMOVE SAMPLE BY")
	return q
}

// ModifyComment modifies the table comment.
func (q *AlterQuery) ModifyComment(comment string) *AlterQuery {
	q.addAlteration(fmt.Sprintf("MODIFY COMMENT '%s'", escapeString(comment)))
	return q
}

// ModifySetting modifies a table setting.
func (q *AlterQuery) ModifySetting(name string, value any) *AlterQuery {
	q.addAlteration(fmt.Sprintf("MODIFY SETTING %s = ?", name), value)
	return q
}

// ResetSetting resets a table setting to default.
func (q *AlterQuery) ResetSetting(name string) *AlterQuery {
	q.addAlteration("RESET SETTING " + name)
	return q
}

// Delete adds a DELETE mutation.
func (q *AlterQuery) Delete(where string, args ...any) *AlterQuery {
	q.addAlteration("DELETE WHERE "+where, args...)
	return q
}

// UpdateMutation adds an UPDATE mutation.
func (q *AlterQuery) UpdateMutation(setExpr, where string, args ...any) *AlterQuery {
	q.addAlteration(fmt.Sprintf("UPDATE %s WHERE %s", setExpr, where), args...)
	return q
}

// Setting adds a query SETTINGS clause.
func (q *AlterQuery) Setting(setting string) *AlterQuery {
	q.appendSetting(setting)
	return q
}

// Build builds the ALTER TABLE query.
func (q *AlterQuery) Build() (string, []any, error) {
	if q.err != nil {
		return "", nil, q.err
	}

	if q.table.IsEmpty() {
		return "", nil, fmt.Errorf("table is required")
	}

	if len(q.alterations) == 0 {
		return "", nil, fmt.Errorf("at least one alteration is required")
	}

	var sb strings.Builder
	var args []any

	sb.WriteString("ALTER TABLE ")
	args = q.table.AppendTo(&sb, args)

	// ON CLUSTER
	args = q.buildOnClusterClause(&sb, args)

	// Alterations
	for i, alt := range q.alterations {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(" ")
		args = alt.AppendTo(&sb, args)
	}

	// SETTINGS
	args = q.buildSettingsClause(&sb, args)

	return sb.String(), args, nil
}

// String returns the SQL query with arguments interpolated (for debugging).
func (q *AlterQuery) String() string {
	sql, args, err := q.Build()
	if err != nil {
		return fmt.Sprintf("ERROR: %v", err)
	}
	return interpolateArgs(sql, args)
}

// Exec executes the ALTER TABLE query.
func (q *AlterQuery) Exec(ctx context.Context) error {
	if q.executor == nil {
		return fmt.Errorf("no executor set")
	}
	sql, args, err := q.Build()
	if err != nil {
		return err
	}
	return q.executor.Exec(ctx, sql, args...)
}

// AlterColumnBuilder helps build column alterations.
type AlterColumnBuilder struct {
	parent       *AlterQuery
	action       string
	name         string
	typ          string
	after        string
	first        bool
	ifNotExists  bool
	ifExists     bool
	defaultExpr  string
	materialized string
	codec        string
	ttl          string
	comment      string
}

// IfNotExists adds IF NOT EXISTS to ADD COLUMN.
func (b *AlterColumnBuilder) IfNotExists() *AlterColumnBuilder {
	b.ifNotExists = true
	return b
}

// IfExists adds IF EXISTS to DROP/MODIFY COLUMN.
func (b *AlterColumnBuilder) IfExists() *AlterColumnBuilder {
	b.ifExists = true
	return b
}

// After positions the column after another column.
func (b *AlterColumnBuilder) After(column string) *AlterColumnBuilder {
	b.after = column
	return b
}

// First positions the column first.
func (b *AlterColumnBuilder) First() *AlterColumnBuilder {
	b.first = true
	return b
}

// Default sets the default expression.
func (b *AlterColumnBuilder) Default(expr string) *AlterColumnBuilder {
	b.defaultExpr = expr
	return b
}

// Materialized sets the materialized expression.
func (b *AlterColumnBuilder) Materialized(expr string) *AlterColumnBuilder {
	b.materialized = expr
	return b
}

// Codec sets the compression codec.
func (b *AlterColumnBuilder) Codec(codec string) *AlterColumnBuilder {
	b.codec = codec
	return b
}

// TTL sets the column TTL.
func (b *AlterColumnBuilder) TTL(expr string) *AlterColumnBuilder {
	b.ttl = expr
	return b
}

// Comment sets the column comment.
func (b *AlterColumnBuilder) Comment(comment string) *AlterColumnBuilder {
	b.comment = comment
	return b
}

// End adds the column alteration to the query.
func (b *AlterColumnBuilder) End() *AlterQuery {
	var sb strings.Builder

	sb.WriteString(b.action)

	if b.ifNotExists {
		sb.WriteString(" IF NOT EXISTS")
	}
	if b.ifExists {
		sb.WriteString(" IF EXISTS")
	}

	sb.WriteString(" ")
	sb.WriteString(quoteIdentifier(b.name))
	sb.WriteString(" ")
	sb.WriteString(b.typ)

	if b.defaultExpr != "" {
		sb.WriteString(" DEFAULT ")
		sb.WriteString(b.defaultExpr)
	}

	if b.materialized != "" {
		sb.WriteString(" MATERIALIZED ")
		sb.WriteString(b.materialized)
	}

	if b.codec != "" {
		sb.WriteString(" CODEC(")
		sb.WriteString(b.codec)
		sb.WriteString(")")
	}

	if b.ttl != "" {
		sb.WriteString(" TTL ")
		sb.WriteString(b.ttl)
	}

	if b.comment != "" {
		sb.WriteString(" COMMENT '")
		sb.WriteString(escapeString(b.comment))
		sb.WriteString("'")
	}

	if b.after != "" {
		sb.WriteString(" AFTER ")
		sb.WriteString(quoteIdentifier(b.after))
	} else if b.first {
		sb.WriteString(" FIRST")
	}

	b.parent.addAlteration(sb.String())
	return b.parent
}
