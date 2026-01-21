package integration

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEdgeCase_NullHandling tests NULL value handling.
func TestEdgeCase_NullHandling(t *testing.T) {
	table := testTable("null_handling")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("nullable_str", "Nullable(String)").Add().
		Column("nullable_int", "Nullable(Int32)").Add().
		Column("nullable_array", "Array(Nullable(String))").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	t.Run("insert and read NULL", func(t *testing.T) {
		err := testClient.Insert(table).
			Columns("id", "nullable_str", "nullable_int", "nullable_array").
			Values(uint32(1), nil, nil, []any{nil, "a", nil}).
			Exec(testCtx)
		require.NoError(t, err)

		var str *string
		var num *int32
		row := testClient.Select("nullable_str", "nullable_int").From(table).Where("id = ?", uint32(1)).QueryRow(testCtx)
		err = row.Scan(&str, &num)
		require.NoError(t, err)
		assert.Nil(t, str)
		assert.Nil(t, num)
	})

	t.Run("WHERE with NULL", func(t *testing.T) {
		// Insert some non-NULL values
		err := testClient.Insert(table).
			Columns("id", "nullable_str", "nullable_int", "nullable_array").
			Values(uint32(2), "not_null", int32(42), []string{"a", "b"}).
			Exec(testCtx)
		require.NoError(t, err)

		// Query for NULL values
		var count uint64
		row := testClient.SelectExpr("count()").From(table).Where("nullable_str IS NULL").QueryRow(testCtx)
		err = row.Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, uint64(1), count)

		// Query for non-NULL values
		row = testClient.SelectExpr("count()").From(table).Where("nullable_str IS NOT NULL").QueryRow(testCtx)
		err = row.Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, uint64(1), count)
	})
}

// TestEdgeCase_EmptyValues tests empty collections and strings.
func TestEdgeCase_EmptyValues(t *testing.T) {
	table := testTable("empty_values")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("empty_str", "String").Add().
		Column("empty_array", "Array(String)").Add().
		Column("empty_map", "Map(String, Int32)").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert empty values
	err = testClient.Insert(table).
		Columns("id", "empty_str", "empty_array", "empty_map").
		Values(uint32(1), "", []string{}, map[string]int32{}).
		Exec(testCtx)
	require.NoError(t, err)

	var str string
	var arr []string
	var m map[string]int32
	row := testClient.Select("empty_str", "empty_array", "empty_map").From(table).Where("id = ?", uint32(1)).QueryRow(testCtx)
	err = row.Scan(&str, &arr, &m)
	require.NoError(t, err)

	assert.Equal(t, "", str)
	assert.Equal(t, []string{}, arr)
	assert.Equal(t, map[string]int32{}, m)
}

// TestEdgeCase_SpecialCharacters tests strings with special characters.
func TestEdgeCase_SpecialCharacters(t *testing.T) {
	table := testTable("special_chars")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("value", "String").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	testCases := []struct {
		name  string
		value string
	}{
		{"single quote", "it's a test"},
		{"double quote", `say "hello"`},
		{"backslash", `path\to\file`},
		{"newline", "line1\nline2"},
		{"tab", "col1\tcol2"},
		{"unicode", "\u4e2d\u6587\u65e5\u672c\u8a9e"},
		{"emoji", "\U0001F600\U0001F389\U0001F680"},
		{"mixed", "Hello 'world' \\ \n \u4e2d\u6587"},
		{"null byte", "before\x00after"},
		{"sql injection attempt", "'; DROP TABLE users; --"},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := testClient.Insert(table).
				Columns("id", "value").
				Values(uint32(i), tc.value).
				Exec(testCtx)
			require.NoError(t, err)

			var result string
			row := testClient.Select("value").From(table).Where("id = ?", uint32(i)).QueryRow(testCtx)
			err = row.Scan(&result)
			require.NoError(t, err)
			assert.Equal(t, tc.value, result)
		})
	}
}

// TestEdgeCase_LargeValues tests large data handling.
func TestEdgeCase_LargeValues(t *testing.T) {
	table := testTable("large_values")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("large_str", "String").Add().
		Column("large_array", "Array(Int32)").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	t.Run("large string", func(t *testing.T) {
		// 1MB string
		largeStr := make([]byte, 1024*1024)
		for i := range largeStr {
			largeStr[i] = byte('a' + (i % 26))
		}

		err := testClient.Insert(table).
			Columns("id", "large_str", "large_array").
			Values(uint32(1), string(largeStr), []int32{}).
			Exec(testCtx)
		require.NoError(t, err)

		var result string
		row := testClient.Select("large_str").From(table).Where("id = ?", uint32(1)).QueryRow(testCtx)
		err = row.Scan(&result)
		require.NoError(t, err)
		assert.Equal(t, len(largeStr), len(result))
	})

	t.Run("large array", func(t *testing.T) {
		// 10,000 element array
		largeArray := make([]int32, 10000)
		for i := range largeArray {
			largeArray[i] = int32(i)
		}

		err := testClient.Insert(table).
			Columns("id", "large_str", "large_array").
			Values(uint32(2), "", largeArray).
			Exec(testCtx)
		require.NoError(t, err)

		var result []int32
		row := testClient.Select("large_array").From(table).Where("id = ?", uint32(2)).QueryRow(testCtx)
		err = row.Scan(&result)
		require.NoError(t, err)
		assert.Equal(t, len(largeArray), len(result))
	})
}

// TestEdgeCase_ConcurrentAccess tests concurrent reads and writes.
func TestEdgeCase_ConcurrentAccess(t *testing.T) {
	table := testTable("concurrent")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt64").Add().
		Column("value", "Int32").Add().
		MergeTree().
		OrderBy("id").
		Exec(testCtx)
	require.NoError(t, err)

	const numGoroutines = 10
	const rowsPerGoroutine = 100

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Concurrent inserts
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < rowsPerGoroutine; i++ {
				id := uint64(goroutineID*rowsPerGoroutine + i)
				err := testClient.Insert(table).
					Columns("id", "value").
					Values(id, int32(goroutineID)).
					Exec(testCtx)
				if err != nil {
					errors <- err
					return
				}
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("concurrent insert error: %v", err)
	}

	// Verify total row count
	var count uint64
	row := testClient.SelectExpr("count()").From(table).QueryRow(testCtx)
	err = row.Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, uint64(numGoroutines*rowsPerGoroutine), count)

	// Concurrent reads
	wg = sync.WaitGroup{}
	readErrors := make(chan error, numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			var cnt uint64
			row := testClient.SelectExpr("count()").From(table).Where("value = ?", int32(goroutineID)).QueryRow(testCtx)
			err := row.Scan(&cnt)
			if err != nil {
				readErrors <- err
				return
			}
			if cnt != rowsPerGoroutine {
				readErrors <- err
			}
		}(g)
	}

	wg.Wait()
	close(readErrors)

	for err := range readErrors {
		t.Errorf("concurrent read error: %v", err)
	}
}

// TestEdgeCase_ErrorHandling tests error conditions.
func TestEdgeCase_ErrorHandling(t *testing.T) {
	t.Run("query non-existent table", func(t *testing.T) {
		_, err := testClient.Select().From("non_existent_table_xyz").Query(testCtx)
		assert.Error(t, err)
	})

	t.Run("insert type mismatch", func(t *testing.T) {
		table := testTable("type_mismatch")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Column("value", "Int32").Add().
			Engine("Memory").
			Exec(testCtx)
		require.NoError(t, err)

		// Try to insert string into Int32 column - should fail
		err = testClient.Insert(table).
			Columns("id", "value").
			Values(uint32(1), "not_a_number").
			Exec(testCtx)
		assert.Error(t, err)
	})

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		table := testTable("cancelled")
		defer dropTable(testCtx, table)

		// This might succeed or fail depending on timing
		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Engine("Memory").
			Exec(ctx)
		// Just check it doesn't panic - error is expected
		_ = err
	})

	t.Run("duplicate key in primary key", func(t *testing.T) {
		// ClickHouse MergeTree allows duplicates - they get merged later
		// This is not an error case, but good to verify behavior
		table := testTable("duplicate_pk")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Column("value", "String").Add().
			MergeTree().
			OrderBy("id").
			Exec(testCtx)
		require.NoError(t, err)

		// Insert duplicate IDs
		err = testClient.Insert(table).Columns("id", "value").Values(uint32(1), "first").Exec(testCtx)
		require.NoError(t, err)
		err = testClient.Insert(table).Columns("id", "value").Values(uint32(1), "second").Exec(testCtx)
		require.NoError(t, err)

		// Both should exist
		var count uint64
		row := testClient.SelectExpr("count()").From(table).Where("id = ?", uint32(1)).QueryRow(testCtx)
		err = row.Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, uint64(2), count)
	})
}

// TestEdgeCase_ExpressionHelpers tests the expression helper functions.
func TestEdgeCase_ExpressionHelpers(t *testing.T) {
	table := testTable("expr_helpers")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("status", "String").Add().
		Column("tags", "Array(String)").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert test data
	err = testClient.Insert(table).Columns("id", "status", "tags").Values(uint32(1), "active", []string{"go", "test"}).Exec(testCtx)
	require.NoError(t, err)
	err = testClient.Insert(table).Columns("id", "status", "tags").Values(uint32(2), "pending", []string{"rust", "test"}).Exec(testCtx)
	require.NoError(t, err)
	err = testClient.Insert(table).Columns("id", "status", "tags").Values(uint32(3), "inactive", []string{"python"}).Exec(testCtx)
	require.NoError(t, err)

	t.Run("In helper", func(t *testing.T) {
		var count uint64
		row := testClient.SelectExpr("count()").From(table).
			WhereIn("status", []string{"active", "pending"}).
			QueryRow(testCtx)
		err := row.Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, uint64(2), count)
	})

	t.Run("NotIn helper", func(t *testing.T) {
		var count uint64
		row := testClient.SelectExpr("count()").From(table).
			WhereNotIn("status", []string{"inactive"}).
			QueryRow(testCtx)
		err := row.Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, uint64(2), count)
	})

	t.Run("Array with hasAny", func(t *testing.T) {
		// Use raw SQL for array function since query.Array() is for INSERT values
		var count uint64
		row := testClient.SelectExpr("count()").From(table).
			Where("hasAny(tags, ['go', 'rust'])").
			QueryRow(testCtx)
		err := row.Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, uint64(2), count)
	})
}

// TestEdgeCase_Transactions tests that ClickHouse doesn't have traditional transactions
// but we can verify the behavior is as expected.
func TestEdgeCase_NoTransactions(t *testing.T) {
	table := testTable("no_transactions")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("value", "String").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert succeeds
	err = testClient.Insert(table).Columns("id", "value").Values(uint32(1), "first").Exec(testCtx)
	require.NoError(t, err)

	// Second insert also succeeds (no transaction to rollback)
	err = testClient.Insert(table).Columns("id", "value").Values(uint32(2), "second").Exec(testCtx)
	require.NoError(t, err)

	// Both rows exist
	assertRowCount(t, testCtx, table, 2)
}
