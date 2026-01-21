package chx

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/ManiacMaxo/chx/query"
)

// Client wraps a ClickHouse connection with a fluent query builder API.
type Client struct {
	conn driver.Conn
}

// New creates a new Client from an existing connection.
func New(conn driver.Conn) *Client {
	return &Client{conn}
}

// Open opens a new ClickHouse connection and returns a Client.
func Open(opt *clickhouse.Options) (*Client, error) {
	conn, err := clickhouse.Open(opt)
	if err != nil {
		return nil, fmt.Errorf("failed to open clickhouse connection: %w", err)
	}

	return New(conn), nil
}

// Conn returns the underlying connection.
func (c *Client) Conn() driver.Conn {
	return c.conn
}

// Close closes the underlying connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Ping pings the server.
func (c *Client) Ping(ctx context.Context) error {
	return c.conn.Ping(ctx)
}

// Query interface implementation for query builders
func (c *Client) Query(ctx context.Context, sql string, args ...any) (driver.Rows, error) {
	return c.conn.Query(ctx, sql, args...)
}

// QueryRow interface implementation for query builders
func (c *Client) QueryRow(ctx context.Context, sql string, args ...any) driver.Row {
	return c.conn.QueryRow(ctx, sql, args...)
}

// Exec interface implementation for query builders
func (c *Client) Exec(ctx context.Context, sql string, args ...any) error {
	return c.conn.Exec(ctx, sql, args...)
}

// PrepareBatch prepares a batch for bulk inserts.
func (c *Client) PrepareBatch(ctx context.Context, sql string) (driver.Batch, error) {
	return c.conn.PrepareBatch(ctx, sql)
}

// Select creates a new SELECT query builder.
func (c *Client) Select(columns ...string) *query.SelectQuery {
	return query.NewSelect(c).Columns(columns...)
}

// SelectExpr creates a new SELECT query builder with raw expressions.
func (c *Client) SelectExpr(expr string, args ...any) *query.SelectQuery {
	return query.NewSelect(c).ColumnExpr(expr, args...)
}

// Insert creates a new INSERT query builder.
func (c *Client) Insert(table string) *query.InsertQuery {
	return query.NewInsert(c).Into(table)
}

// Update creates a new UPDATE query builder (lightweight updates).
func (c *Client) Update(table string) *query.UpdateQuery {
	return query.NewUpdate(c).Table(table)
}

// Delete creates a new DELETE query builder (lightweight deletes).
func (c *Client) Delete(table string) *query.DeleteQuery {
	return query.NewDelete(c).From(table)
}

// CreateTable creates a new CREATE TABLE query builder.
func (c *Client) CreateTable(table string) *query.CreateTableQuery {
	return query.NewCreateTable(c).Table(table)
}

// CreateView creates a new CREATE VIEW query builder.
func (c *Client) CreateView(name string) *query.CreateViewQuery {
	return query.NewCreateView(c).View(name)
}

// CreateMaterializedView creates a new CREATE MATERIALIZED VIEW query builder.
func (c *Client) CreateMaterializedView(name string) *query.CreateViewQuery {
	return query.NewCreateView(c).View(name).Materialized()
}

// Alter creates a new ALTER TABLE query builder.
func (c *Client) Alter(table string) *query.AlterQuery {
	return query.NewAlter(c).Table(table)
}

// DropTable creates a new DROP TABLE query builder.
func (c *Client) DropTable(table string) *query.DropQuery {
	return query.NewDrop(c).Table(table)
}

// DropView creates a new DROP VIEW query builder.
func (c *Client) DropView(name string) *query.DropQuery {
	return query.NewDrop(c).View(name)
}

// DropDatabase creates a new DROP DATABASE query builder.
func (c *Client) DropDatabase(name string) *query.DropQuery {
	return query.NewDrop(c).Database(name)
}

// Truncate creates a new TRUNCATE TABLE query builder.
func (c *Client) Truncate(table string) *query.TruncateQuery {
	return query.NewTruncate(c).Table(table)
}

// Optimize creates a new OPTIMIZE TABLE query builder.
func (c *Client) Optimize(table string) *query.OptimizeQuery {
	return query.NewOptimize(c).Table(table)
}

// Raw executes a raw SQL query.
func (c *Client) Raw(ctx context.Context, sql string, args ...any) (driver.Rows, error) {
	return c.conn.Query(ctx, sql, args...)
}

// RawExec executes a raw SQL statement.
func (c *Client) RawExec(ctx context.Context, sql string, args ...any) error {
	return c.conn.Exec(ctx, sql, args...)
}
