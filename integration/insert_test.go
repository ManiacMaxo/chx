package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInsert_Values tests INSERT with VALUES.
func TestInsert_Values(t *testing.T) {
	table := testTable("insert_values")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("name", "String").Add().
		Column("value", "Int32").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	t.Run("single row", func(t *testing.T) {
		err := testClient.Insert(table).
			Columns("id", "name", "value").
			Values(uint32(1), "test", int32(100)).
			Exec(testCtx)
		require.NoError(t, err)

		assertRowCount(t, testCtx, table, 1)
	})

	t.Run("multiple rows", func(t *testing.T) {
		err := testClient.Insert(table).
			Columns("id", "name", "value").
			Values(uint32(2), "row2", int32(200)).
			Values(uint32(3), "row3", int32(300)).
			Values(uint32(4), "row4", int32(400)).
			Exec(testCtx)
		require.NoError(t, err)

		assertRowCount(t, testCtx, table, 4) // 1 from previous test + 3 new
	})
}

// TestInsert_Struct tests INSERT from structs.
func TestInsert_Struct(t *testing.T) {
	table := testTable("insert_struct")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("name", "String").Add().
		Column("email", "String").Add().
		Column("age", "UInt8").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	type User struct {
		ID    uint32 `ch:"id"`
		Name  string `ch:"name"`
		Email string `ch:"email"`
		Age   uint8  `ch:"age"`
	}

	t.Run("single struct", func(t *testing.T) {
		user := User{ID: 1, Name: "Alice", Email: "alice@example.com", Age: 30}
		err := testClient.Insert(table).Struct(&user).Exec(testCtx)
		require.NoError(t, err)

		// Verify
		var name string
		row := testClient.Select("name").From(table).Where("id = ?", uint32(1)).QueryRow(testCtx)
		err = row.Scan(&name)
		require.NoError(t, err)
		assert.Equal(t, "Alice", name)
	})

	t.Run("multiple structs", func(t *testing.T) {
		users := []User{
			{ID: 2, Name: "Bob", Email: "bob@example.com", Age: 25},
			{ID: 3, Name: "Charlie", Email: "charlie@example.com", Age: 35},
		}
		for _, u := range users {
			err := testClient.Insert(table).Struct(&u).Exec(testCtx)
			require.NoError(t, err)
		}

		assertRowCount(t, testCtx, table, 3)
	})
}

// TestInsert_Map tests INSERT from maps.
func TestInsert_Map(t *testing.T) {
	table := testTable("insert_map")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("name", "String").Add().
		Column("value", "Int32").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	data := map[string]any{
		"id":    uint32(1),
		"name":  "from_map",
		"value": int32(999),
	}
	err = testClient.Insert(table).Map(data).Exec(testCtx)
	require.NoError(t, err)

	// Verify
	var name string
	var value int32
	row := testClient.Select("name", "value").From(table).Where("id = ?", uint32(1)).QueryRow(testCtx)
	err = row.Scan(&name, &value)
	require.NoError(t, err)
	assert.Equal(t, "from_map", name)
	assert.Equal(t, int32(999), value)
}

// TestInsert_Select tests INSERT ... SELECT.
func TestInsert_Select(t *testing.T) {
	source := testTable("insert_select_src")
	dest := testTable("insert_select_dst")
	defer dropTable(testCtx, source)
	defer dropTable(testCtx, dest)

	// Create source table
	err := testClient.CreateTable(source).
		Column("id", "UInt32").Add().
		Column("name", "String").Add().
		Column("value", "Int32").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	// Create destination table with same schema
	err = testClient.CreateTable(dest).
		Column("id", "UInt32").Add().
		Column("name", "String").Add().
		Column("value", "Int32").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	// Populate source
	for i := 1; i <= 5; i++ {
		err := testClient.Insert(source).
			Columns("id", "name", "value").
			Values(uint32(i), "item"+string(rune('0'+i)), int32(i*10)).
			Exec(testCtx)
		require.NoError(t, err)
	}

	// INSERT ... SELECT with filter
	selectQuery := testClient.Select("id", "name", "value").From(source).Where("value > ?", int32(20))
	err = testClient.Insert(dest).
		Columns("id", "name", "value").
		Select(selectQuery).
		Exec(testCtx)
	require.NoError(t, err)

	// Should have 3 rows (value > 20 means 30, 40, 50)
	assertRowCount(t, testCtx, dest, 3)
}

// TestInsert_Defaults tests INSERT with default values.
func TestInsert_Defaults(t *testing.T) {
	table := testTable("insert_defaults")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("name", "String").Default("'unknown'").Add().
		Column("created_at", "DateTime").Default("now()").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert only id, let defaults fill in
	err = testClient.Insert(table).
		Columns("id").
		Values(uint32(1)).
		Exec(testCtx)
	require.NoError(t, err)

	// Verify defaults were applied
	var name string
	row := testClient.Select("name").From(table).Where("id = ?", uint32(1)).QueryRow(testCtx)
	err = row.Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "unknown", name)
}

// TestInsert_NullableColumns tests INSERT with nullable columns.
func TestInsert_NullableColumns(t *testing.T) {
	table := testTable("insert_nullable")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("optional_name", "Nullable(String)").Add().
		Column("optional_value", "Nullable(Int32)").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert with NULL values
	err = testClient.Insert(table).
		Columns("id", "optional_name", "optional_value").
		Values(uint32(1), nil, nil).
		Exec(testCtx)
	require.NoError(t, err)

	// Insert with non-NULL values
	err = testClient.Insert(table).
		Columns("id", "optional_name", "optional_value").
		Values(uint32(2), "has_value", int32(42)).
		Exec(testCtx)
	require.NoError(t, err)

	// Verify NULL
	var name1 *string
	row := testClient.Select("optional_name").From(table).Where("id = ?", uint32(1)).QueryRow(testCtx)
	err = row.Scan(&name1)
	require.NoError(t, err)
	assert.Nil(t, name1)

	// Verify non-NULL
	var name2 *string
	row = testClient.Select("optional_name").From(table).Where("id = ?", uint32(2)).QueryRow(testCtx)
	err = row.Scan(&name2)
	require.NoError(t, err)
	require.NotNil(t, name2)
	assert.Equal(t, "has_value", *name2)
}

// TestInsert_ComplexTypes tests INSERT with arrays, maps, tuples.
func TestInsert_ComplexTypes(t *testing.T) {
	table := testTable("insert_complex")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("tags", "Array(String)").Add().
		Column("metadata", "Map(String, String)").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	tags := []string{"go", "clickhouse", "test"}
	metadata := map[string]string{"env": "test", "version": "1.0"}

	err = testClient.Insert(table).
		Columns("id", "tags", "metadata").
		Values(uint32(1), tags, metadata).
		Exec(testCtx)
	require.NoError(t, err)

	// Verify
	var resultTags []string
	var resultMeta map[string]string
	row := testClient.Select("tags", "metadata").From(table).Where("id = ?", uint32(1)).QueryRow(testCtx)
	err = row.Scan(&resultTags, &resultMeta)
	require.NoError(t, err)
	assert.Equal(t, tags, resultTags)
	assert.Equal(t, metadata, resultMeta)
}
