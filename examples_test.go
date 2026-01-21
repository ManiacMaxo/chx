package chx_test

import (
	"fmt"

	"github.com/ManiacMaxo/chx/query"
)

func ExampleSelect_basic() {
	q := query.Select("id", "name", "email").
		From("users").
		Where("status = ?", "active").
		Where("age >= ?", 18).
		OrderByDesc("created_at").
		Limit(10)

	fmt.Println(q.String())
	// Output: SELECT `id`, `name`, `email` FROM `users` WHERE status = 'active' AND age >= 18 ORDER BY `created_at` DESC LIMIT 10
}

func ExampleSelect_clickhouseFeatures() {
	q := query.Select("user_id", "event").
		ColumnExpr("count(*) as cnt").
		From("events").
		Final().                         // ReplacingMergeTree deduplication
		Sample(0.1).                     // Sample 10% of data
		Prewhere("project_id = ?", 123). // Filter before reading columns
		Where("timestamp > ?", "2024-01-01").
		GroupBy("user_id", "event").
		WithTotals(). // Include totals row
		Having("cnt > ?", 100).
		OrderByDesc("cnt").
		LimitBy(5, "user_id"). // Top 5 events per user
		Setting("max_threads = 4")

	sql, args, _ := q.Build()
	fmt.Println("SQL:", sql)
	fmt.Println("Args:", args)
}

func ExampleSelect_cte() {
	activeUsers := query.Select("id", "name").
		From("users").
		Where("status = ?", "active")

	q := query.Select("id", "name").
		With("active_users", activeUsers).
		From("active_users").
		OrderBy("name")

	fmt.Println(q.String())
	// Output: WITH active_users AS (SELECT `id`, `name` FROM `users` WHERE status = 'active') SELECT `id`, `name` FROM `active_users` ORDER BY `name`
}

func ExampleSelect_joins() {
	q := query.Select().
		ColumnExpr("u.id, u.name, count(o.id) as order_count").
		FromExpr("users AS u").
		LeftJoin("orders").As("o").On("o.user_id = u.id").End().
		GroupByExpr("u.id, u.name").
		Having("order_count > ?", 0)

	sql, _, _ := q.Build()
	fmt.Println(sql)
	// Output: SELECT u.id, u.name, count(o.id) as order_count FROM users AS u LEFT JOIN `orders` AS `o` ON o.user_id = u.id GROUP BY u.id, u.name HAVING order_count > ?
}

func ExampleSelect_union() {
	admins := query.Select("id", "name").From("admins")
	users := query.Select("id", "name").From("users")

	// Note: In ClickHouse, ORDER BY applies to the first query in UNION
	// Use subquery or wrap in another SELECT for global ordering
	q := admins.UnionAll(users)

	fmt.Println(q.String())
	// Output: SELECT `id`, `name` FROM `admins` UNION ALL SELECT `id`, `name` FROM `users`
}

func ExampleInsert_basic() {
	q := query.Insert("users").
		Columns("id", "name", "email").
		Values(1, "John Doe", "john@example.com").
		Values(2, "Jane Doe", "jane@example.com")

	fmt.Println(q.String())
	// Output: INSERT INTO `users` (`id`, `name`, `email`) VALUES (1, 'John Doe', 'john@example.com'), (2, 'Jane Doe', 'jane@example.com')
}

func ExampleInsert_fromSelect() {
	selectQuery := query.Select("id", "name", "email").
		From("old_users").
		Where("migrated = ?", false)

	q := query.Insert("users").
		Columns("id", "name", "email").
		Select(selectQuery)

	sql, _, _ := q.Build()
	fmt.Println(sql)
	// Output: INSERT INTO `users` (`id`, `name`, `email`) SELECT `id`, `name`, `email` FROM `old_users` WHERE migrated = ?
}

func ExampleCreateTable_mergeTree() {
	q := query.CreateTable("events").
		IfNotExists().
		Column("id", "UInt64").Add().
		Column("user_id", "UInt32").Add().
		Column("event", "LowCardinality(String)").Add().
		Column("timestamp", "DateTime64(3)").Default("now()").Add().
		Column("data", "String").Codec("ZSTD(1)").Add().
		ReplacingMergeTree("timestamp").
		PartitionBy("toYYYYMM(timestamp)").
		OrderBy("user_id", "timestamp").
		TTL("timestamp + INTERVAL 90 DAY").
		Setting("index_granularity = 8192")

	sql, _, _ := q.Build()
	fmt.Println(sql)
}

func ExampleCreateTable_replicated() {
	q := query.CreateTable("events").
		IfNotExists().
		OnCluster("production").
		Column("id", "UInt64").Add().
		Column("data", "String").Add().
		ReplicatedMergeTree("/clickhouse/tables/{shard}/events", "{replica}").
		OrderBy("id")

	sql, _, _ := q.Build()
	fmt.Println(sql)
}

func ExampleCreateMaterializedView() {
	selectQuery := query.Select().
		ColumnExpr("user_id, toDate(timestamp) as date, count(*) as cnt").
		From("events").
		GroupBy("user_id", "date")

	q := query.CreateMaterializedView("daily_events").
		IfNotExists().
		To("daily_events_data").
		As(selectQuery)

	sql, _, _ := q.Build()
	fmt.Println(sql)
	// Output: CREATE MATERIALIZED VIEW IF NOT EXISTS `daily_events` TO `daily_events_data` AS SELECT user_id, toDate(timestamp) as date, count(*) as cnt FROM `events` GROUP BY `user_id`, `date`
}

func ExampleAlter_addColumn() {
	q := query.Alter("users").
		AddColumn("phone", "Nullable(String)").After("email").End()

	fmt.Println(q.String())
	// Output: ALTER TABLE `users` ADD COLUMN `phone` Nullable(String) AFTER `email`
}

func ExampleAlter_partition() {
	q := query.Alter("events").
		DropPartition("202301").
		DropPartition("202302")

	fmt.Println(q.String())
	// Output: ALTER TABLE `events` DROP PARTITION 202301, DROP PARTITION 202302
}

func ExampleDelete() {
	q := query.Delete("users").
		Where("status = ?", "deleted").
		Where("deleted_at < ?", "2024-01-01")

	fmt.Println(q.String())
	// Output: DELETE FROM `users` WHERE status = 'deleted' AND deleted_at < '2024-01-01'
}

func ExampleUpdate() {
	q := query.Update("users").
		Set("status", "inactive").
		SetExpr("updated_at = now()").
		Where("last_login < ?", "2023-01-01")

	fmt.Println(q.String())
	// Output: ALTER TABLE `users` UPDATE `status` = 'inactive', updated_at = now() WHERE last_login < '2023-01-01'
}

func ExampleOptimize() {
	q := query.Optimize("events").
		Partition("202401").
		Final().
		Deduplicate()

	fmt.Println(q.String())
	// Output: OPTIMIZE TABLE `events` PARTITION 202401 FINAL DEDUPLICATE
}

func ExampleDropTable() {
	q := query.DropTable("old_events").
		IfExists().
		OnCluster("production").
		Sync()

	fmt.Println(q.String())
	// Output: DROP TABLE IF EXISTS `old_events` ON CLUSTER `production` SYNC
}

func ExampleTruncate() {
	q := query.Truncate("events").
		IfExists()

	fmt.Println(q.String())
	// Output: TRUNCATE TABLE IF EXISTS `events`
}

func ExampleIn() {
	q := query.Select("id", "name").
		From("users").
		WhereIn("id", []int{1, 2, 3, 4, 5})

	sql, args, _ := q.Build()
	fmt.Println("SQL:", sql)
	fmt.Println("Args:", args)
	// Output:
	// SQL: SELECT `id`, `name` FROM `users` WHERE `id` IN (?, ?, ?, ?, ?)
	// Args: [1 2 3 4 5]
}
