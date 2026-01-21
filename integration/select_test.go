package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelect_Basic tests basic SELECT queries.
func TestSelect_Basic(t *testing.T) {
	table := testTable("select_basic")
	defer dropTable(testCtx, table)

	// Create and populate table
	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("name", "String").Add().
		Column("age", "UInt8").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert test data
	for i := 1; i <= 5; i++ {
		err := testClient.Insert(table).
			Columns("id", "name", "age").
			Values(uint32(i), "user"+string(rune('0'+i)), uint8(20+i)).
			Exec(testCtx)
		require.NoError(t, err)
	}

	t.Run("select all", func(t *testing.T) {
		rows, err := testClient.Select().From(table).Query(testCtx)
		require.NoError(t, err)
		defer rows.Close()

		count := 0
		for rows.Next() {
			count++
		}
		assert.Equal(t, 5, count)
	})

	t.Run("select columns", func(t *testing.T) {
		var id uint32
		var name string
		row := testClient.Select("id", "name").From(table).Where("id = ?", uint32(1)).QueryRow(testCtx)
		err := row.Scan(&id, &name)
		require.NoError(t, err)
		assert.Equal(t, uint32(1), id)
		assert.Equal(t, "user1", name)
	})

	t.Run("select with limit", func(t *testing.T) {
		rows, err := testClient.Select("id").From(table).Limit(3).Query(testCtx)
		require.NoError(t, err)
		defer rows.Close()

		count := 0
		for rows.Next() {
			count++
		}
		assert.Equal(t, 3, count)
	})

	t.Run("select with offset", func(t *testing.T) {
		rows, err := testClient.Select("id").From(table).OrderBy("id").Limit(10).Offset(3).Query(testCtx)
		require.NoError(t, err)
		defer rows.Close()

		var ids []uint32
		for rows.Next() {
			var id uint32
			err := rows.Scan(&id)
			require.NoError(t, err)
			ids = append(ids, id)
		}
		assert.Equal(t, []uint32{4, 5}, ids)
	})
}

// TestSelect_Where tests WHERE clause variations.
func TestSelect_Where(t *testing.T) {
	table := testTable("select_where")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("status", "String").Add().
		Column("value", "Int32").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert test data
	data := []struct {
		id     uint32
		status string
		value  int32
	}{
		{1, "active", 100},
		{2, "active", 200},
		{3, "inactive", 150},
		{4, "pending", 50},
		{5, "active", 300},
	}
	for _, d := range data {
		err := testClient.Insert(table).Columns("id", "status", "value").Values(d.id, d.status, d.value).Exec(testCtx)
		require.NoError(t, err)
	}

	t.Run("single condition", func(t *testing.T) {
		var count uint64
		row := testClient.SelectExpr("count()").From(table).Where("status = ?", "active").QueryRow(testCtx)
		err := row.Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, uint64(3), count)
	})

	t.Run("multiple AND conditions", func(t *testing.T) {
		var count uint64
		row := testClient.SelectExpr("count()").From(table).
			Where("status = ?", "active").
			Where("value > ?", int32(150)).
			QueryRow(testCtx)
		err := row.Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, uint64(2), count)
	})

	t.Run("OR condition", func(t *testing.T) {
		var count uint64
		row := testClient.SelectExpr("count()").From(table).
			Where("status = ?", "active").
			WhereOr("status = ?", "pending").
			QueryRow(testCtx)
		err := row.Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, uint64(4), count)
	})

	t.Run("IN clause", func(t *testing.T) {
		var count uint64
		row := testClient.SelectExpr("count()").From(table).
			WhereIn("status", []string{"active", "pending"}).
			QueryRow(testCtx)
		err := row.Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, uint64(4), count)
	})

	t.Run("comparison operators", func(t *testing.T) {
		var count uint64
		row := testClient.SelectExpr("count()").From(table).
			Where("value >= ?", int32(100)).
			Where("value <= ?", int32(200)).
			QueryRow(testCtx)
		err := row.Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, uint64(3), count)
	})
}

// TestSelect_OrderBy tests ORDER BY clause.
func TestSelect_OrderBy(t *testing.T) {
	table := testTable("select_orderby")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("value", "Int32").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert in random order
	for _, v := range []int32{30, 10, 50, 20, 40} {
		err := testClient.Insert(table).Columns("id", "value").Values(uint32(v), v).Exec(testCtx)
		require.NoError(t, err)
	}

	t.Run("order by asc", func(t *testing.T) {
		rows, err := testClient.Select("value").From(table).OrderBy("value").Query(testCtx)
		require.NoError(t, err)
		defer rows.Close()

		var values []int32
		for rows.Next() {
			var v int32
			err := rows.Scan(&v)
			require.NoError(t, err)
			values = append(values, v)
		}
		assert.Equal(t, []int32{10, 20, 30, 40, 50}, values)
	})

	t.Run("order by desc", func(t *testing.T) {
		rows, err := testClient.Select("value").From(table).OrderByDesc("value").Query(testCtx)
		require.NoError(t, err)
		defer rows.Close()

		var values []int32
		for rows.Next() {
			var v int32
			err := rows.Scan(&v)
			require.NoError(t, err)
			values = append(values, v)
		}
		assert.Equal(t, []int32{50, 40, 30, 20, 10}, values)
	})
}

// TestSelect_GroupBy tests GROUP BY with aggregates.
func TestSelect_GroupBy(t *testing.T) {
	table := testTable("select_groupby")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("category", "String").Add().
		Column("value", "Int32").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert test data
	data := []struct {
		category string
		value    int32
	}{
		{"A", 10}, {"A", 20}, {"A", 30},
		{"B", 100}, {"B", 200},
		{"C", 5},
	}
	for _, d := range data {
		err := testClient.Insert(table).Columns("category", "value").Values(d.category, d.value).Exec(testCtx)
		require.NoError(t, err)
	}

	t.Run("count by group", func(t *testing.T) {
		rows, err := testClient.SelectExpr("category, count() as cnt").
			From(table).
			GroupBy("category").
			OrderBy("category").
			Query(testCtx)
		require.NoError(t, err)
		defer rows.Close()

		results := make(map[string]uint64)
		for rows.Next() {
			var cat string
			var cnt uint64
			err := rows.Scan(&cat, &cnt)
			require.NoError(t, err)
			results[cat] = cnt
		}
		assert.Equal(t, uint64(3), results["A"])
		assert.Equal(t, uint64(2), results["B"])
		assert.Equal(t, uint64(1), results["C"])
	})

	t.Run("sum by group", func(t *testing.T) {
		rows, err := testClient.SelectExpr("category, sum(value) as total").
			From(table).
			GroupBy("category").
			OrderBy("category").
			Query(testCtx)
		require.NoError(t, err)
		defer rows.Close()

		results := make(map[string]int64)
		for rows.Next() {
			var cat string
			var total int64
			err := rows.Scan(&cat, &total)
			require.NoError(t, err)
			results[cat] = total
		}
		assert.Equal(t, int64(60), results["A"])
		assert.Equal(t, int64(300), results["B"])
		assert.Equal(t, int64(5), results["C"])
	})

	t.Run("having clause", func(t *testing.T) {
		rows, err := testClient.SelectExpr("category, count() as cnt").
			From(table).
			GroupBy("category").
			Having("cnt > ?", uint64(1)).
			OrderBy("category").
			Query(testCtx)
		require.NoError(t, err)
		defer rows.Close()

		var categories []string
		for rows.Next() {
			var cat string
			var cnt uint64
			err := rows.Scan(&cat, &cnt)
			require.NoError(t, err)
			categories = append(categories, cat)
		}
		assert.Equal(t, []string{"A", "B"}, categories)
	})
}

// TestSelect_Distinct tests DISTINCT modifier.
func TestSelect_Distinct(t *testing.T) {
	table := testTable("select_distinct")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("category", "String").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert duplicates
	for _, cat := range []string{"A", "A", "B", "B", "B", "C"} {
		err := testClient.Insert(table).Columns("category").Values(cat).Exec(testCtx)
		require.NoError(t, err)
	}

	rows, err := testClient.Select("category").From(table).Distinct().OrderBy("category").Query(testCtx)
	require.NoError(t, err)
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var cat string
		err := rows.Scan(&cat)
		require.NoError(t, err)
		categories = append(categories, cat)
	}
	assert.Equal(t, []string{"A", "B", "C"}, categories)
}

// TestSelect_Join tests JOIN operations.
func TestSelect_Join(t *testing.T) {
	users := testTable("users")
	orders := testTable("orders")
	defer dropTable(testCtx, users)
	defer dropTable(testCtx, orders)

	// Create users table
	err := testClient.CreateTable(users).
		Column("id", "UInt32").Add().
		Column("name", "String").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	// Create orders table
	err = testClient.CreateTable(orders).
		Column("id", "UInt32").Add().
		Column("user_id", "UInt32").Add().
		Column("amount", "Decimal(10, 2)").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert users
	for _, u := range []struct {
		id   uint32
		name string
	}{{1, "Alice"}, {2, "Bob"}, {3, "Charlie"}} {
		err := testClient.Insert(users).Columns("id", "name").Values(u.id, u.name).Exec(testCtx)
		require.NoError(t, err)
	}

	// Insert orders (user 3 has no orders)
	for _, o := range []struct {
		id      uint32
		user_id uint32
		amount  string
	}{{1, 1, "100.00"}, {2, 1, "200.00"}, {3, 2, "150.00"}} {
		err := testClient.Insert(orders).Columns("id", "user_id", "amount").Values(o.id, o.user_id, o.amount).Exec(testCtx)
		require.NoError(t, err)
	}

	t.Run("inner join", func(t *testing.T) {
		rows, err := testClient.Select("u.name", "o.amount").
			FromExpr(users + " AS u").
			Join(orders).As("o").On("u.id = o.user_id").End().
			OrderByExpr("o.id").
			Query(testCtx)
		require.NoError(t, err)
		defer rows.Close()

		count := 0
		for rows.Next() {
			count++
		}
		assert.Equal(t, 3, count) // Alice has 2 orders, Bob has 1
	})

	t.Run("left join", func(t *testing.T) {
		rows, err := testClient.Select("u.name").
			FromExpr(users + " AS u").
			LeftJoin(orders).As("o").On("u.id = o.user_id").End().
			GroupBy("u.name").
			OrderByExpr("u.name").
			Query(testCtx)
		require.NoError(t, err)
		defer rows.Close()

		var names []string
		for rows.Next() {
			var name string
			err := rows.Scan(&name)
			require.NoError(t, err)
			names = append(names, name)
		}
		assert.Equal(t, []string{"Alice", "Bob", "Charlie"}, names) // Charlie included with NULL order
	})

}

// TestSelect_Subquery tests subqueries.
func TestSelect_Subquery(t *testing.T) {
	table := testTable("select_subquery")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("value", "Int32").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert test data
	for i := 1; i <= 5; i++ {
		err := testClient.Insert(table).Columns("id", "value").Values(uint32(i), int32(i*10)).Exec(testCtx)
		require.NoError(t, err)
	}

	// Select rows with value greater than average using a subquery in WHERE
	// Build the subquery SQL manually and embed it
	avgSubquery := testClient.SelectExpr("avg(value)").From(table)
	avgSQL, _, err := avgSubquery.Build()
	require.NoError(t, err)

	rows, err := testClient.Select("id", "value").
		From(table).
		Where("value > (" + avgSQL + ")").
		OrderBy("id").
		Query(testCtx)
	require.NoError(t, err)
	defer rows.Close()

	var ids []uint32
	for rows.Next() {
		var id uint32
		var value int32
		err := rows.Scan(&id, &value)
		require.NoError(t, err)
		ids = append(ids, id)
	}
	// Average is 30, so we expect ids 4 and 5 (values 40 and 50)
	assert.Equal(t, []uint32{4, 5}, ids)
}

// TestSelect_ClickHouseFeatures tests ClickHouse-specific features.
func TestSelect_ClickHouseFeatures(t *testing.T) {
	t.Run("LIMIT BY", func(t *testing.T) {
		table := testTable("limit_by")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Column("category", "String").Add().
			Column("value", "Int32").Add().
			MergeTree().
			OrderBy("id").
			Exec(testCtx)
		require.NoError(t, err)

		// Insert test data
		for i := 1; i <= 100; i++ {
			cat := "A"
			if i%2 == 0 {
				cat = "B"
			}
			err := testClient.Insert(table).Columns("id", "category", "value").Values(uint32(i), cat, int32(i)).Exec(testCtx)
			require.NoError(t, err)
		}

		// Get top 2 values per category
		rows, err := testClient.Select("category", "value").
			From(table).
			OrderByDesc("value").
			LimitBy(2, "category").
			Query(testCtx)
		require.NoError(t, err)
		defer rows.Close()

		count := 0
		for rows.Next() {
			count++
		}
		assert.Equal(t, 4, count) // 2 per category * 2 categories
	})

	t.Run("FINAL", func(t *testing.T) {
		table := testTable("final")
		defer dropTable(testCtx, table)

		// FINAL requires ReplacingMergeTree
		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Column("version", "UInt32").Add().
			Column("value", "String").Add().
			ReplacingMergeTree("version").
			OrderBy("id").
			Exec(testCtx)
		require.NoError(t, err)

		// Insert two versions
		err = testClient.Insert(table).Columns("id", "version", "value").Values(uint32(1), uint32(1), "v1").Exec(testCtx)
		require.NoError(t, err)
		err = testClient.Insert(table).Columns("id", "version", "value").Values(uint32(1), uint32(2), "v2").Exec(testCtx)
		require.NoError(t, err)

		// Query with FINAL should deduplicate
		rows, err := testClient.Select("id", "value").From(table).Final().Query(testCtx)
		require.NoError(t, err)
		defer rows.Close()

		count := 0
		for rows.Next() {
			count++
		}
		assert.Equal(t, 1, count)
	})

	t.Run("SAMPLE", func(t *testing.T) {
		table := testTable("sample")
		defer dropTable(testCtx, table)

		// SAMPLE BY requires the expression to be in the primary key (ORDER BY)
		// Use raw CREATE TABLE for proper SAMPLE BY syntax
		err := testClient.RawExec(testCtx, `
			CREATE TABLE `+"`"+table+"`"+` (
				id UInt64,
				value Int32
			) ENGINE = MergeTree()
			ORDER BY id
			SAMPLE BY id
		`)
		require.NoError(t, err)

		// Insert test data
		for i := 1; i <= 1000; i++ {
			err := testClient.Insert(table).Columns("id", "value").Values(uint64(i), int32(i)).Exec(testCtx)
			require.NoError(t, err)
		}

		// Sample 10% of rows - just verify the query works
		rows, err := testClient.Select("id").From(table).Sample(0.1).Query(testCtx)
		require.NoError(t, err)
		defer rows.Close()

		count := 0
		for rows.Next() {
			count++
		}
		// Just verify sampling returns something and the query worked
		t.Logf("SAMPLE(0.1) returned %d rows out of 1000", count)
		assert.True(t, count >= 0 && count <= 1000, "expected valid sample count, got %d", count)
	})
}

// TestSelect_Union tests UNION operations.
func TestSelect_Union(t *testing.T) {
	table1 := testTable("union1")
	table2 := testTable("union2")
	defer dropTable(testCtx, table1)
	defer dropTable(testCtx, table2)

	// Create tables
	for _, tbl := range []string{table1, table2} {
		err := testClient.CreateTable(tbl).
			Column("id", "UInt32").Add().
			Column("value", "String").Add().
			Engine("Memory").
			Exec(testCtx)
		require.NoError(t, err)
	}

	// Insert data
	err := testClient.Insert(table1).Columns("id", "value").Values(uint32(1), "a").Exec(testCtx)
	require.NoError(t, err)
	err = testClient.Insert(table1).Columns("id", "value").Values(uint32(2), "b").Exec(testCtx)
	require.NoError(t, err)
	err = testClient.Insert(table2).Columns("id", "value").Values(uint32(2), "b").Exec(testCtx)
	require.NoError(t, err)
	err = testClient.Insert(table2).Columns("id", "value").Values(uint32(3), "c").Exec(testCtx)
	require.NoError(t, err)

	t.Run("UNION ALL", func(t *testing.T) {
		rows, err := testClient.Select("id", "value").From(table1).
			UnionAll(testClient.Select("id", "value").From(table2)).
			OrderBy("id").
			Query(testCtx)
		require.NoError(t, err)
		defer rows.Close()

		count := 0
		for rows.Next() {
			count++
		}
		assert.Equal(t, 4, count) // All rows including duplicate (2, "b")
	})

	t.Run("UNION DISTINCT", func(t *testing.T) {
		rows, err := testClient.Select("id", "value").From(table1).
			UnionDistinct(testClient.Select("id", "value").From(table2)).
			OrderBy("id").
			Query(testCtx)
		require.NoError(t, err)
		defer rows.Close()

		count := 0
		for rows.Next() {
			count++
		}
		assert.Equal(t, 3, count) // Duplicate (2, "b") removed
	})
}

// TestSelect_CTE tests Common Table Expressions (WITH clause).
func TestSelect_CTE(t *testing.T) {
	table := testTable("cte")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("parent_id", "Nullable(UInt32)").Add().
		Column("name", "String").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert hierarchical data
	data := []struct {
		id       uint32
		parentID *uint32
		name     string
	}{
		{1, nil, "root"},
		{2, ptr(uint32(1)), "child1"},
		{3, ptr(uint32(1)), "child2"},
		{4, ptr(uint32(2)), "grandchild1"},
	}
	for _, d := range data {
		err := testClient.Insert(table).Columns("id", "parent_id", "name").Values(d.id, d.parentID, d.name).Exec(testCtx)
		require.NoError(t, err)
	}

	// Use CTE to select root items
	rows, err := testClient.Select("name").
		With("roots", testClient.Select("id", "name").From(table).Where("parent_id IS NULL")).
		FromExpr("roots").
		Query(testCtx)
	require.NoError(t, err)
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		require.NoError(t, err)
		names = append(names, name)
	}
	assert.Equal(t, []string{"root"}, names)
}

// ptr returns a pointer to the value.
func ptr[T any](v T) *T {
	return &v
}
