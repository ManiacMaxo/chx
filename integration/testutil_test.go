package integration

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
)

// tableCounter ensures unique table names across tests.
var tableCounter atomic.Uint64

// testTable generates a unique table name for a test.
func testTable(prefix string) string {
	return fmt.Sprintf("test_%s_%d", prefix, tableCounter.Add(1))
}

// dropTable drops a table, ignoring errors (for cleanup).
func dropTable(ctx context.Context, table string) {
	_ = testClient.DropTable(table).IfExists().Exec(ctx)
}

// mustExec executes a raw SQL statement and fails the test on error.
func mustExec(t *testing.T, ctx context.Context, sql string, args ...any) {
	t.Helper()
	if err := testClient.RawExec(ctx, sql, args...); err != nil {
		t.Fatalf("exec failed: %v\nSQL: %s", err, sql)
	}
}

// assertRowCount verifies the row count in a table.
func assertRowCount(t *testing.T, ctx context.Context, table string, expected uint64) {
	t.Helper()
	var count uint64
	row := testClient.Select("count()").From(table).QueryRow(ctx)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("failed to count rows: %v", err)
	}
	if count != expected {
		t.Errorf("expected %d rows, got %d", expected, count)
	}
}

// createMemoryTable creates a simple Memory table with the given columns.
// columns is a map of column_name -> clickhouse_type.
func createMemoryTable(t *testing.T, ctx context.Context, table string, columns map[string]string) {
	t.Helper()

	q := testClient.CreateTable(table)
	for name, typ := range columns {
		q = q.Column(name, typ).Add()
	}
	q = q.Engine("Memory")

	if err := q.Exec(ctx); err != nil {
		t.Fatalf("failed to create table %s: %v", table, err)
	}
}

// createMergeTreeTable creates a MergeTree table with the given columns and order by.
func createMergeTreeTable(t *testing.T, ctx context.Context, table string, columns map[string]string, orderBy ...string) {
	t.Helper()

	q := testClient.CreateTable(table)
	for name, typ := range columns {
		q = q.Column(name, typ).Add()
	}
	q = q.MergeTree().OrderBy(orderBy...)

	if err := q.Exec(ctx); err != nil {
		t.Fatalf("failed to create table %s: %v", table, err)
	}
}
