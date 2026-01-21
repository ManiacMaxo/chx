package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: ClickHouse UPDATE and DELETE are "lightweight" mutations that work
// asynchronously via ALTER TABLE. They require MergeTree family engines.

// TestUpdate_Basic tests basic UPDATE operations.
func TestUpdate_Basic(t *testing.T) {
	table := testTable("update_basic")
	defer dropTable(testCtx, table)

	// Must use MergeTree for mutations
	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("name", "String").Add().
		Column("status", "String").Add().
		MergeTree().
		OrderBy("id").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert test data
	for i := 1; i <= 5; i++ {
		err := testClient.Insert(table).
			Columns("id", "name", "status").
			Values(uint32(i), "user"+string(rune('0'+i)), "active").
			Exec(testCtx)
		require.NoError(t, err)
	}

	t.Run("update single column", func(t *testing.T) {
		err := testClient.Update(table).
			Set("status", "inactive").
			Where("id = ?", uint32(1)).
			Exec(testCtx)
		require.NoError(t, err)

		// Wait for mutation to complete
		time.Sleep(500 * time.Millisecond)

		// Verify
		var status string
		row := testClient.Select("status").From(table).Where("id = ?", uint32(1)).QueryRow(testCtx)
		err = row.Scan(&status)
		require.NoError(t, err)
		assert.Equal(t, "inactive", status)
	})

	t.Run("update multiple columns", func(t *testing.T) {
		err := testClient.Update(table).
			Set("name", "updated_user").
			Set("status", "pending").
			Where("id = ?", uint32(2)).
			Exec(testCtx)
		require.NoError(t, err)

		time.Sleep(500 * time.Millisecond)

		var name, status string
		row := testClient.Select("name", "status").From(table).Where("id = ?", uint32(2)).QueryRow(testCtx)
		err = row.Scan(&name, &status)
		require.NoError(t, err)
		assert.Equal(t, "updated_user", name)
		assert.Equal(t, "pending", status)
	})

	t.Run("update with multiple conditions", func(t *testing.T) {
		err := testClient.Update(table).
			Set("status", "bulk_updated").
			Where("id >= ?", uint32(3)).
			Where("status = ?", "active").
			Exec(testCtx)
		require.NoError(t, err)

		time.Sleep(500 * time.Millisecond)

		// Count bulk_updated rows
		var count uint64
		row := testClient.SelectExpr("count()").From(table).Where("status = ?", "bulk_updated").QueryRow(testCtx)
		err = row.Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, uint64(3), count) // ids 3, 4, 5
	})
}

// TestDelete_Basic tests basic DELETE operations.
func TestDelete_Basic(t *testing.T) {
	table := testTable("delete_basic")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("category", "String").Add().
		MergeTree().
		OrderBy("id").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert test data
	data := []struct {
		id       uint32
		category string
	}{
		{1, "A"}, {2, "A"}, {3, "B"}, {4, "B"}, {5, "C"},
	}
	for _, d := range data {
		err := testClient.Insert(table).Columns("id", "category").Values(d.id, d.category).Exec(testCtx)
		require.NoError(t, err)
	}

	t.Run("delete single row", func(t *testing.T) {
		err := testClient.Delete(table).
			Where("id = ?", uint32(1)).
			Exec(testCtx)
		require.NoError(t, err)

		time.Sleep(500 * time.Millisecond)

		// Verify row is gone
		var count uint64
		row := testClient.SelectExpr("count()").From(table).Where("id = ?", uint32(1)).QueryRow(testCtx)
		err = row.Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), count)
	})

	t.Run("delete by category", func(t *testing.T) {
		err := testClient.Delete(table).
			Where("category = ?", "B").
			Exec(testCtx)
		require.NoError(t, err)

		time.Sleep(500 * time.Millisecond)

		// Verify category B is gone
		var count uint64
		row := testClient.SelectExpr("count()").From(table).Where("category = ?", "B").QueryRow(testCtx)
		err = row.Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), count)
	})

	t.Run("remaining rows", func(t *testing.T) {
		// Should have: id=2 (A) and id=5 (C)
		var count uint64
		row := testClient.SelectExpr("count()").From(table).QueryRow(testCtx)
		err := row.Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, uint64(2), count)
	})
}

// TestTruncate tests TRUNCATE TABLE.
func TestTruncate(t *testing.T) {
	table := testTable("truncate")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("data", "String").Add().
		MergeTree().
		OrderBy("id").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert data
	for i := 1; i <= 100; i++ {
		err := testClient.Insert(table).Columns("id", "data").Values(uint32(i), "data").Exec(testCtx)
		require.NoError(t, err)
	}

	assertRowCount(t, testCtx, table, 100)

	// Truncate
	err = testClient.Truncate(table).Exec(testCtx)
	require.NoError(t, err)

	assertRowCount(t, testCtx, table, 0)

	// Table should still exist and be usable
	err = testClient.Insert(table).Columns("id", "data").Values(uint32(1), "after_truncate").Exec(testCtx)
	require.NoError(t, err)

	assertRowCount(t, testCtx, table, 1)
}

// TestOptimize tests OPTIMIZE TABLE.
func TestOptimize(t *testing.T) {
	table := testTable("optimize")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("data", "String").Add().
		MergeTree().
		OrderBy("id").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert data in multiple batches to create multiple parts
	for batch := 0; batch < 5; batch++ {
		for i := 1; i <= 10; i++ {
			id := uint32(batch*10 + i)
			err := testClient.Insert(table).Columns("id", "data").Values(id, "data").Exec(testCtx)
			require.NoError(t, err)
		}
	}

	// Optimize to merge parts
	err = testClient.Optimize(table).Final().Exec(testCtx)
	require.NoError(t, err)

	// Data should still be intact
	assertRowCount(t, testCtx, table, 50)
}
