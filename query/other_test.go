package query

import (
	"testing"
)

func TestDropTable(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *DropQuery
		expected string
	}{
		{
			name: "drop table",
			build: func() *DropQuery {
				return DropTable("users")
			},
			expected: "DROP TABLE `users`",
		},
		{
			name: "drop table if exists",
			build: func() *DropQuery {
				return DropTable("users").IfExists()
			},
			expected: "DROP TABLE IF EXISTS `users`",
		},
		{
			name: "drop table on cluster",
			build: func() *DropQuery {
				return DropTable("users").IfExists().OnCluster("cluster1")
			},
			expected: "DROP TABLE IF EXISTS `users` ON CLUSTER `cluster1`",
		},
		{
			name: "drop table sync",
			build: func() *DropQuery {
				return DropTable("users").Sync()
			},
			expected: "DROP TABLE `users` SYNC",
		},
		{
			name: "drop view",
			build: func() *DropQuery {
				return DropView("my_view").IfExists()
			},
			expected: "DROP VIEW IF EXISTS `my_view`",
		},
		{
			name: "drop database",
			build: func() *DropQuery {
				return DropDatabase("my_db").IfExists()
			},
			expected: "DROP DATABASE IF EXISTS `my_db`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.build()
			sql, _, err := q.Build()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sql != tt.expected {
				t.Errorf("\nexpected: %s\ngot:      %s", tt.expected, sql)
			}
		})
	}
}

func TestTruncateTable(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *TruncateQuery
		expected string
	}{
		{
			name: "truncate table",
			build: func() *TruncateQuery {
				return Truncate("users")
			},
			expected: "TRUNCATE TABLE `users`",
		},
		{
			name: "truncate table if exists",
			build: func() *TruncateQuery {
				return Truncate("users").IfExists()
			},
			expected: "TRUNCATE TABLE IF EXISTS `users`",
		},
		{
			name: "truncate table on cluster",
			build: func() *TruncateQuery {
				return Truncate("users").OnCluster("cluster1")
			},
			expected: "TRUNCATE TABLE `users` ON CLUSTER `cluster1`",
		},
		{
			name: "truncate table sync",
			build: func() *TruncateQuery {
				return Truncate("users").Sync()
			},
			expected: "TRUNCATE TABLE `users` SYNC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.build()
			sql, _, err := q.Build()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sql != tt.expected {
				t.Errorf("\nexpected: %s\ngot:      %s", tt.expected, sql)
			}
		})
	}
}

func TestOptimizeTable(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *OptimizeQuery
		expected string
	}{
		{
			name: "optimize table",
			build: func() *OptimizeQuery {
				return Optimize("users")
			},
			expected: "OPTIMIZE TABLE `users`",
		},
		{
			name: "optimize table final",
			build: func() *OptimizeQuery {
				return Optimize("users").Final()
			},
			expected: "OPTIMIZE TABLE `users` FINAL",
		},
		{
			name: "optimize table partition",
			build: func() *OptimizeQuery {
				return Optimize("events").Partition("202301")
			},
			expected: "OPTIMIZE TABLE `events` PARTITION 202301",
		},
		{
			name: "optimize table deduplicate",
			build: func() *OptimizeQuery {
				return Optimize("users").Deduplicate()
			},
			expected: "OPTIMIZE TABLE `users` DEDUPLICATE",
		},
		{
			name: "optimize table deduplicate by columns",
			build: func() *OptimizeQuery {
				return Optimize("users").DeduplicateBy("id", "name")
			},
			expected: "OPTIMIZE TABLE `users` DEDUPLICATE BY `id`, `name`",
		},
		{
			name: "optimize table final deduplicate",
			build: func() *OptimizeQuery {
				return Optimize("users").Final().Deduplicate()
			},
			expected: "OPTIMIZE TABLE `users` FINAL DEDUPLICATE",
		},
		{
			name: "optimize table on cluster",
			build: func() *OptimizeQuery {
				return Optimize("users").OnCluster("cluster1").Final()
			},
			expected: "OPTIMIZE TABLE `users` ON CLUSTER `cluster1` FINAL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.build()
			sql, _, err := q.Build()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sql != tt.expected {
				t.Errorf("\nexpected: %s\ngot:      %s", tt.expected, sql)
			}
		})
	}
}

func TestCreateView(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *CreateViewQuery
		expected string
	}{
		{
			name: "create view",
			build: func() *CreateViewQuery {
				return CreateView("active_users").
					As(Select("id", "name").From("users").Where("status = ?", "active"))
			},
			expected: "CREATE VIEW `active_users` AS SELECT `id`, `name` FROM `users` WHERE status = ?",
		},
		{
			name: "create view if not exists",
			build: func() *CreateViewQuery {
				return CreateView("active_users").
					IfNotExists().
					As(Select("id", "name").From("users"))
			},
			expected: "CREATE VIEW IF NOT EXISTS `active_users` AS SELECT `id`, `name` FROM `users`",
		},
		{
			name: "create materialized view",
			build: func() *CreateViewQuery {
				return CreateMaterializedView("user_stats").
					To("user_stats_data").
					As(Select().ColumnExpr("user_id, count(*) as cnt").From("events").GroupBy("user_id"))
			},
			expected: "CREATE MATERIALIZED VIEW `user_stats` TO `user_stats_data` AS SELECT user_id, count(*) as cnt FROM `events` GROUP BY `user_id`",
		},
		{
			name: "create materialized view with engine",
			build: func() *CreateViewQuery {
				return CreateMaterializedView("user_stats").
					Engine("SummingMergeTree()").
					OrderBy("user_id").
					As(Select().ColumnExpr("user_id, count(*) as cnt").From("events").GroupBy("user_id"))
			},
			expected: "CREATE MATERIALIZED VIEW `user_stats` ENGINE = SummingMergeTree() ORDER BY (`user_id`) AS SELECT user_id, count(*) as cnt FROM `events` GROUP BY `user_id`",
		},
		{
			name: "create materialized view populate",
			build: func() *CreateViewQuery {
				return CreateMaterializedView("user_stats").
					To("user_stats_data").
					Populate().
					As(Select().ColumnExpr("user_id, count(*) as cnt").From("events").GroupBy("user_id"))
			},
			expected: "CREATE MATERIALIZED VIEW `user_stats` TO `user_stats_data` POPULATE AS SELECT user_id, count(*) as cnt FROM `events` GROUP BY `user_id`",
		},
		{
			name: "create materialized view on cluster",
			build: func() *CreateViewQuery {
				return CreateMaterializedView("user_stats").
					OnCluster("cluster1").
					To("user_stats_data").
					As(Select().ColumnExpr("user_id, count(*) as cnt").From("events").GroupBy("user_id"))
			},
			expected: "CREATE MATERIALIZED VIEW `user_stats` ON CLUSTER `cluster1` TO `user_stats_data` AS SELECT user_id, count(*) as cnt FROM `events` GROUP BY `user_id`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.build()
			sql, _, err := q.Build()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sql != tt.expected {
				t.Errorf("\nexpected: %s\ngot:      %s", tt.expected, sql)
			}
		})
	}
}

func TestExprHelpers(t *testing.T) {
	t.Run("In", func(t *testing.T) {
		in := In([]int{1, 2, 3})
		expr, args := in.Build()
		if expr != "(?, ?, ?)" {
			t.Errorf("expected (?, ?, ?), got %s", expr)
		}
		if len(args) != 3 {
			t.Errorf("expected 3 args, got %d", len(args))
		}
	})

	t.Run("Array", func(t *testing.T) {
		arr := Array([]string{"a", "b", "c"})
		expr, args := arr.Build()
		if expr != "[?, ?, ?]" {
			t.Errorf("expected [?, ?, ?], got %s", expr)
		}
		if len(args) != 3 {
			t.Errorf("expected 3 args, got %d", len(args))
		}
	})

	t.Run("Tuple", func(t *testing.T) {
		tuple := Tuple(1, "a", true)
		expr, args := tuple.Build()
		if expr != "(?, ?, ?)" {
			t.Errorf("expected (?, ?, ?), got %s", expr)
		}
		if len(args) != 3 {
			t.Errorf("expected 3 args, got %d", len(args))
		}
	})

	t.Run("Ident", func(t *testing.T) {
		id := Ident("column_name")
		if string(id) != "column_name" {
			t.Errorf("expected column_name, got %s", id)
		}
	})

	t.Run("Safe", func(t *testing.T) {
		s := Safe("now()")
		if string(s) != "now()" {
			t.Errorf("expected now(), got %s", s)
		}
	})

	t.Run("Raw", func(t *testing.T) {
		r := Raw("1 + 1")
		if string(r) != "1 + 1" {
			t.Errorf("expected 1 + 1, got %s", r)
		}
	})
}
