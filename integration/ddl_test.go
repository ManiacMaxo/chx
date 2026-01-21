package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateTable_Engines tests CREATE TABLE with various engines.
func TestCreateTable_Engines(t *testing.T) {
	t.Run("MergeTree", func(t *testing.T) {
		table := testTable("engine_mergetree")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt64").Add().
			Column("timestamp", "DateTime").Add().
			Column("value", "Float64").Add().
			MergeTree().
			OrderBy("id").
			Exec(testCtx)
		require.NoError(t, err)

		// Insert and read back
		err = testClient.Insert(table).
			Columns("id", "timestamp", "value").
			Values(uint64(1), time.Now(), float64(3.14)).
			Exec(testCtx)
		require.NoError(t, err)

		assertRowCount(t, testCtx, table, 1)
	})

	t.Run("ReplacingMergeTree", func(t *testing.T) {
		table := testTable("engine_replacing")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt64").Add().
			Column("version", "UInt64").Add().
			Column("data", "String").Add().
			ReplacingMergeTree("version").
			OrderBy("id").
			Exec(testCtx)
		require.NoError(t, err)

		// Insert two versions of same id
		err = testClient.Insert(table).Columns("id", "version", "data").Values(uint64(1), uint64(1), "v1").Exec(testCtx)
		require.NoError(t, err)
		err = testClient.Insert(table).Columns("id", "version", "data").Values(uint64(1), uint64(2), "v2").Exec(testCtx)
		require.NoError(t, err)

		// Optimize to trigger replacement
		err = testClient.Optimize(table).Final().Exec(testCtx)
		require.NoError(t, err)

		// Should only have latest version
		var data string
		row := testClient.Select("data").From(table).Final().Where("id = ?", uint64(1)).QueryRow(testCtx)
		err = row.Scan(&data)
		require.NoError(t, err)
		assert.Equal(t, "v2", data)
	})

	t.Run("SummingMergeTree", func(t *testing.T) {
		table := testTable("engine_summing")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("key", "String").Add().
			Column("value", "UInt64").Add().
			SummingMergeTree("value").
			OrderBy("key").
			Exec(testCtx)
		require.NoError(t, err)

		// Insert multiple rows with same key
		for i := 0; i < 5; i++ {
			err = testClient.Insert(table).Columns("key", "value").Values("a", uint64(10)).Exec(testCtx)
			require.NoError(t, err)
		}

		// Optimize to trigger summing
		err = testClient.Optimize(table).Final().Exec(testCtx)
		require.NoError(t, err)

		// Should have sum of values
		var value uint64
		row := testClient.Select("value").From(table).Final().Where("key = ?", "a").QueryRow(testCtx)
		err = row.Scan(&value)
		require.NoError(t, err)
		assert.Equal(t, uint64(50), value)
	})

	t.Run("AggregatingMergeTree", func(t *testing.T) {
		table := testTable("engine_aggregating")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("key", "String").Add().
			Column("value", "AggregateFunction(sum, UInt64)").Add().
			AggregatingMergeTree().
			OrderBy("key").
			Exec(testCtx)
		require.NoError(t, err)

		// Table exists and is usable
		assertRowCount(t, testCtx, table, 0)
	})

	t.Run("CollapsingMergeTree", func(t *testing.T) {
		table := testTable("engine_collapsing")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt64").Add().
			Column("data", "String").Add().
			Column("sign", "Int8").Add().
			CollapsingMergeTree("sign").
			OrderBy("id").
			Exec(testCtx)
		require.NoError(t, err)

		// Insert row with sign=1, then "delete" with sign=-1
		err = testClient.Insert(table).Columns("id", "data", "sign").Values(uint64(1), "original", int8(1)).Exec(testCtx)
		require.NoError(t, err)
		err = testClient.Insert(table).Columns("id", "data", "sign").Values(uint64(1), "original", int8(-1)).Exec(testCtx)
		require.NoError(t, err)

		// Optimize to collapse
		err = testClient.Optimize(table).Final().Exec(testCtx)
		require.NoError(t, err)

		// Row should be collapsed (gone)
		var count uint64
		row := testClient.SelectExpr("count()").From(table).Final().QueryRow(testCtx)
		err = row.Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), count)
	})

	t.Run("Memory", func(t *testing.T) {
		table := testTable("engine_memory")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Column("data", "String").Add().
			Memory().
			Exec(testCtx)
		require.NoError(t, err)

		err = testClient.Insert(table).Columns("id", "data").Values(uint32(1), "test").Exec(testCtx)
		require.NoError(t, err)

		assertRowCount(t, testCtx, table, 1)
	})

	t.Run("Log", func(t *testing.T) {
		table := testTable("engine_log")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Column("data", "String").Add().
			Log().
			Exec(testCtx)
		require.NoError(t, err)

		err = testClient.Insert(table).Columns("id", "data").Values(uint32(1), "test").Exec(testCtx)
		require.NoError(t, err)

		assertRowCount(t, testCtx, table, 1)
	})

	t.Run("TinyLog", func(t *testing.T) {
		table := testTable("engine_tinylog")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Column("data", "String").Add().
			TinyLog().
			Exec(testCtx)
		require.NoError(t, err)

		err = testClient.Insert(table).Columns("id", "data").Values(uint32(1), "test").Exec(testCtx)
		require.NoError(t, err)

		assertRowCount(t, testCtx, table, 1)
	})
}

// TestCreateTable_Options tests CREATE TABLE with various options.
func TestCreateTable_Options(t *testing.T) {
	t.Run("IfNotExists", func(t *testing.T) {
		table := testTable("ifnotexists")
		defer dropTable(testCtx, table)

		// Create table
		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Engine("Memory").
			Exec(testCtx)
		require.NoError(t, err)

		// Create again with IfNotExists - should not error
		err = testClient.CreateTable(table).
			IfNotExists().
			Column("id", "UInt32").Add().
			Engine("Memory").
			Exec(testCtx)
		require.NoError(t, err)
	})

	t.Run("PartitionBy", func(t *testing.T) {
		table := testTable("partitionby")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt64").Add().
			Column("date", "Date").Add().
			Column("value", "Float64").Add().
			MergeTree().
			PartitionBy("toYYYYMM(date)").
			OrderBy("id").
			Exec(testCtx)
		require.NoError(t, err)

		// Insert data for different months
		err = testClient.Insert(table).Columns("id", "date", "value").
			Values(uint64(1), time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), float64(1.0)).
			Exec(testCtx)
		require.NoError(t, err)

		err = testClient.Insert(table).Columns("id", "date", "value").
			Values(uint64(2), time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC), float64(2.0)).
			Exec(testCtx)
		require.NoError(t, err)

		assertRowCount(t, testCtx, table, 2)
	})

	t.Run("TTL", func(t *testing.T) {
		table := testTable("ttl")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt64").Add().
			Column("created_at", "DateTime").Add().
			Column("data", "String").Add().
			MergeTree().
			OrderBy("id").
			TTL("created_at + INTERVAL 1 DAY").
			Exec(testCtx)
		require.NoError(t, err)

		// Insert data
		err = testClient.Insert(table).Columns("id", "created_at", "data").
			Values(uint64(1), time.Now(), "test").
			Exec(testCtx)
		require.NoError(t, err)

		assertRowCount(t, testCtx, table, 1)
	})

	t.Run("Comment", func(t *testing.T) {
		table := testTable("comment")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Engine("Memory").
			Comment("This is a test table").
			Exec(testCtx)
		require.NoError(t, err)

		// Table should exist
		assertRowCount(t, testCtx, table, 0)
	})

	t.Run("ColumnOptions", func(t *testing.T) {
		table := testTable("coloptions")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt64").Add().
			Column("name", "String").Default("'default_name'").Add().
			Column("computed", "String").Materialized("concat(name, '_suffix')").Add().
			Column("data", "String").Codec("LZ4").Add().
			MergeTree().
			OrderBy("id").
			Exec(testCtx)
		require.NoError(t, err)

		// Insert with default
		err = testClient.Insert(table).Columns("id").Values(uint64(1)).Exec(testCtx)
		require.NoError(t, err)

		var name string
		row := testClient.Select("name").From(table).Where("id = ?", uint64(1)).QueryRow(testCtx)
		err = row.Scan(&name)
		require.NoError(t, err)
		assert.Equal(t, "default_name", name)
	})
}

// TestAlterTable tests ALTER TABLE operations.
func TestAlterTable(t *testing.T) {
	table := testTable("alter")
	defer dropTable(testCtx, table)

	// Create initial table
	err := testClient.CreateTable(table).
		Column("id", "UInt64").Add().
		Column("name", "String").Add().
		MergeTree().
		OrderBy("id").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert initial data
	err = testClient.Insert(table).Columns("id", "name").Values(uint64(1), "test").Exec(testCtx)
	require.NoError(t, err)

	t.Run("AddColumn", func(t *testing.T) {
		err := testClient.Alter(table).
			AddColumn("new_col", "String").Default("'new_default'").End().
			Exec(testCtx)
		require.NoError(t, err)

		// Verify column exists with default
		var newCol string
		row := testClient.Select("new_col").From(table).Where("id = ?", uint64(1)).QueryRow(testCtx)
		err = row.Scan(&newCol)
		require.NoError(t, err)
		assert.Equal(t, "new_default", newCol)
	})

	t.Run("ModifyColumn", func(t *testing.T) {
		// ModifyColumn returns *AlterQuery directly, so we use the raw MODIFY COLUMN syntax
		err := testClient.Alter(table).
			ModifyColumn("new_col", "String DEFAULT 'modified_default'").
			Exec(testCtx)
		require.NoError(t, err)

		// Insert new row to verify modified default
		err = testClient.Insert(table).Columns("id", "name").Values(uint64(2), "test2").Exec(testCtx)
		require.NoError(t, err)

		var newCol string
		row := testClient.Select("new_col").From(table).Where("id = ?", uint64(2)).QueryRow(testCtx)
		err = row.Scan(&newCol)
		require.NoError(t, err)
		assert.Equal(t, "modified_default", newCol)
	})

	t.Run("DropColumn", func(t *testing.T) {
		err := testClient.Alter(table).
			DropColumn("new_col").
			Exec(testCtx)
		require.NoError(t, err)

		// Verify column is gone by selecting only existing columns
		var id uint64
		row := testClient.Select("id").From(table).Limit(1).QueryRow(testCtx)
		err = row.Scan(&id)
		require.NoError(t, err)
	})
}

// TestDropTable tests DROP TABLE operations.
func TestDropTable(t *testing.T) {
	t.Run("DropExisting", func(t *testing.T) {
		table := testTable("drop_existing")

		// Create table
		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Engine("Memory").
			Exec(testCtx)
		require.NoError(t, err)

		// Drop it
		err = testClient.DropTable(table).Exec(testCtx)
		require.NoError(t, err)

		// Verify it's gone - creating it again should work
		err = testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Engine("Memory").
			Exec(testCtx)
		require.NoError(t, err)

		// Cleanup
		dropTable(testCtx, table)
	})

	t.Run("DropIfExists", func(t *testing.T) {
		table := testTable("drop_nonexistent")

		// Drop non-existent table with IF EXISTS - should not error
		err := testClient.DropTable(table).IfExists().Exec(testCtx)
		require.NoError(t, err)
	})
}

// TestCreateView tests CREATE VIEW and CREATE MATERIALIZED VIEW.
func TestCreateView(t *testing.T) {
	sourceTable := testTable("view_source")
	defer dropTable(testCtx, sourceTable)

	// Create source table
	err := testClient.CreateTable(sourceTable).
		Column("id", "UInt64").Add().
		Column("value", "Int32").Add().
		MergeTree().
		OrderBy("id").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert data
	for i := 1; i <= 10; i++ {
		err = testClient.Insert(sourceTable).Columns("id", "value").Values(uint64(i), int32(i*10)).Exec(testCtx)
		require.NoError(t, err)
	}

	t.Run("View", func(t *testing.T) {
		viewName := testTable("regular_view")
		defer testClient.DropView(viewName).IfExists().Exec(testCtx)

		selectQuery := testClient.Select("id", "value").From(sourceTable).Where("value > ?", int32(50))
		err := testClient.CreateView(viewName).
			As(selectQuery).
			Exec(testCtx)
		require.NoError(t, err)

		// Query the view
		var count uint64
		row := testClient.SelectExpr("count()").FromExpr("`" + viewName + "`").QueryRow(testCtx)
		err = row.Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, uint64(5), count) // values 60, 70, 80, 90, 100
	})

	t.Run("MaterializedView", func(t *testing.T) {
		mvName := testTable("mat_view")
		mvTarget := testTable("mat_view_target")
		defer testClient.DropView(mvName).IfExists().Exec(testCtx)
		defer dropTable(testCtx, mvTarget)

		// Create target table for materialized view
		err := testClient.CreateTable(mvTarget).
			Column("id", "UInt64").Add().
			Column("doubled_value", "Int32").Add().
			MergeTree().
			OrderBy("id").
			Exec(testCtx)
		require.NoError(t, err)

		// Create materialized view
		selectQuery := testClient.SelectExpr("id, value * 2 as doubled_value").From(sourceTable)
		err = testClient.CreateMaterializedView(mvName).
			To(mvTarget).
			As(selectQuery).
			Exec(testCtx)
		require.NoError(t, err)

		// Insert new data to source
		err = testClient.Insert(sourceTable).Columns("id", "value").Values(uint64(100), int32(50)).Exec(testCtx)
		require.NoError(t, err)

		// Check if MV populated target
		var doubled int32
		row := testClient.Select("doubled_value").From(mvTarget).Where("id = ?", uint64(100)).QueryRow(testCtx)
		err = row.Scan(&doubled)
		require.NoError(t, err)
		assert.Equal(t, int32(100), doubled) // 50 * 2
	})
}
