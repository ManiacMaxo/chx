package query

import (
	"testing"
)

func TestSelectBasic(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *SelectQuery
		expected string
	}{
		{
			name: "simple select all",
			build: func() *SelectQuery {
				return Select().From("users")
			},
			expected: "SELECT * FROM `users`",
		},
		{
			name: "select columns",
			build: func() *SelectQuery {
				return Select("id", "name", "email").From("users")
			},
			expected: "SELECT `id`, `name`, `email` FROM `users`",
		},
		{
			name: "select with where",
			build: func() *SelectQuery {
				return Select("id", "name").From("users").Where("status = ?", "active")
			},
			expected: "SELECT `id`, `name` FROM `users` WHERE status = ?",
		},
		{
			name: "select with multiple where",
			build: func() *SelectQuery {
				return Select("id", "name").
					From("users").
					Where("status = ?", "active").
					Where("age >= ?", 18)
			},
			expected: "SELECT `id`, `name` FROM `users` WHERE status = ? AND age >= ?",
		},
		{
			name: "select with where or",
			build: func() *SelectQuery {
				return Select("id", "name").
					From("users").
					Where("status = ?", "active").
					WhereOr("role = ?", "admin")
			},
			expected: "SELECT `id`, `name` FROM `users` WHERE status = ? OR role = ?",
		},
		{
			name: "select distinct",
			build: func() *SelectQuery {
				return Select("country").From("users").Distinct()
			},
			expected: "SELECT DISTINCT `country` FROM `users`",
		},
		{
			name: "select with limit offset",
			build: func() *SelectQuery {
				return Select("id", "name").From("users").Limit(10).Offset(20)
			},
			expected: "SELECT `id`, `name` FROM `users` LIMIT 20, 10",
		},
		{
			name: "select with order by",
			build: func() *SelectQuery {
				return Select("id", "name").From("users").OrderBy("created_at")
			},
			expected: "SELECT `id`, `name` FROM `users` ORDER BY `created_at`",
		},
		{
			name: "select with order by desc",
			build: func() *SelectQuery {
				return Select("id", "name").From("users").OrderByDesc("created_at")
			},
			expected: "SELECT `id`, `name` FROM `users` ORDER BY `created_at` DESC",
		},
		{
			name: "select with group by",
			build: func() *SelectQuery {
				return Select().ColumnExpr("status, count(*) as cnt").
					From("users").
					GroupBy("status")
			},
			expected: "SELECT status, count(*) as cnt FROM `users` GROUP BY `status`",
		},
		{
			name: "select with having",
			build: func() *SelectQuery {
				return Select().ColumnExpr("status, count(*) as cnt").
					From("users").
					GroupBy("status").
					Having("cnt > ?", 10)
			},
			expected: "SELECT status, count(*) as cnt FROM `users` GROUP BY `status` HAVING cnt > ?",
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

func TestSelectClickHouseSpecific(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *SelectQuery
		expected string
	}{
		{
			name: "select with final",
			build: func() *SelectQuery {
				return Select("id", "name").From("users").Final()
			},
			expected: "SELECT `id`, `name` FROM `users` FINAL",
		},
		{
			name: "select with sample",
			build: func() *SelectQuery {
				return Select("id").From("events").Sample(0.1)
			},
			expected: "SELECT `id` FROM `events` SAMPLE 0.1",
		},
		{
			name: "select with sample rows",
			build: func() *SelectQuery {
				return Select("id").From("events").SampleRows(10000)
			},
			expected: "SELECT `id` FROM `events` SAMPLE 10000",
		},
		{
			name: "select with prewhere",
			build: func() *SelectQuery {
				return Select("id").From("events").
					Prewhere("project_id = ?", 123).
					Where("event_type = ?", "click")
			},
			expected: "SELECT `id` FROM `events` PREWHERE project_id = ? WHERE event_type = ?",
		},
		{
			name: "select with limit by",
			build: func() *SelectQuery {
				return Select("user_id", "event", "timestamp").
					From("events").
					OrderByDesc("timestamp").
					LimitBy(5, "user_id")
			},
			expected: "SELECT `user_id`, `event`, `timestamp` FROM `events` ORDER BY `timestamp` DESC LIMIT 5 BY `user_id`",
		},
		{
			name: "select with with totals",
			build: func() *SelectQuery {
				return Select().ColumnExpr("status, count(*) as cnt").
					From("users").
					GroupBy("status").
					WithTotals()
			},
			expected: "SELECT status, count(*) as cnt FROM `users` GROUP BY `status` WITH TOTALS",
		},
		{
			name: "select with with rollup",
			build: func() *SelectQuery {
				return Select().ColumnExpr("country, city, count(*) as cnt").
					From("users").
					GroupBy("country", "city").
					WithRollup()
			},
			expected: "SELECT country, city, count(*) as cnt FROM `users` GROUP BY `country`, `city` WITH ROLLUP",
		},
		{
			name: "select with settings",
			build: func() *SelectQuery {
				return Select("id").From("events").
					Setting("max_threads = 4").
					Setting("max_memory_usage = 1000000000")
			},
			expected: "SELECT `id` FROM `events` SETTINGS max_threads = 4, max_memory_usage = 1000000000",
		},
		{
			name: "select with format",
			build: func() *SelectQuery {
				return Select("id", "name").From("users").Format("JSONEachRow")
			},
			expected: "SELECT `id`, `name` FROM `users` FORMAT JSONEachRow",
		},
		{
			name: "select with array join",
			build: func() *SelectQuery {
				return Select("id", "tag").From("posts").ArrayJoin("tags")
			},
			expected: "SELECT `id`, `tag` FROM `posts` ARRAY JOIN `tags`",
		},
		{
			name: "select with left array join",
			build: func() *SelectQuery {
				return Select("id", "tag").From("posts").LeftArrayJoin("tags")
			},
			expected: "SELECT `id`, `tag` FROM `posts` LEFT ARRAY JOIN `tags`",
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

func TestSelectJoins(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *SelectQuery
		expected string
	}{
		{
			name: "inner join",
			build: func() *SelectQuery {
				// Using ColumnExpr for qualified names since they contain dots
				return Select().ColumnExpr("u.id, u.name, o.total").
					From("users").FromExpr("AS u").
					InnerJoin("orders").As("o").On("o.user_id = u.id").End()
			},
			expected: "SELECT u.id, u.name, o.total FROM `users`, AS u INNER JOIN `orders` AS `o` ON o.user_id = u.id",
		},
		{
			name: "left join with using",
			build: func() *SelectQuery {
				return Select().ColumnExpr("u.id, p.name").
					FromExpr("users AS u").
					LeftJoin("profiles").As("p").Using("user_id").End()
			},
			expected: "SELECT u.id, p.name FROM users AS u LEFT JOIN `profiles` AS `p` USING (`user_id`)",
		},
		{
			name: "global join",
			build: func() *SelectQuery {
				return Select().ColumnExpr("a.id, b.value").
					FromExpr("table_a AS a").
					GlobalJoin("table_b").As("b").On("a.id = b.id").End()
			},
			expected: "SELECT a.id, b.value FROM table_a AS a GLOBAL INNER JOIN `table_b` AS `b` ON a.id = b.id",
		},
		{
			name: "any left join",
			build: func() *SelectQuery {
				return Select().ColumnExpr("a.id, b.value").
					FromExpr("table_a AS a").
					AnyJoin("table_b").As("b").On("a.id = b.id").End()
			},
			expected: "SELECT a.id, b.value FROM table_a AS a ANY INNER JOIN `table_b` AS `b` ON a.id = b.id",
		},
		{
			name: "asof join",
			build: func() *SelectQuery {
				return Select().ColumnExpr("a.id, b.value").
					FromExpr("table_a AS a").
					AsofJoin("table_b").As("b").On("a.id = b.id").End()
			},
			expected: "SELECT a.id, b.value FROM table_a AS a ASOF INNER JOIN `table_b` AS `b` ON a.id = b.id",
		},
		{
			name: "cross join",
			build: func() *SelectQuery {
				return Select().ColumnExpr("a.x, b.y").
					FromExpr("table_a AS a").
					CrossJoin("table_b")
			},
			expected: "SELECT a.x, b.y FROM table_a AS a CROSS JOIN `table_b`",
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

func TestSelectCTE(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *SelectQuery
		expected string
	}{
		{
			name: "with cte",
			build: func() *SelectQuery {
				subquery := Select("id", "name").From("users").Where("status = ?", "active")
				return Select("id", "name").
					With("active_users", subquery).
					From("active_users")
			},
			expected: "WITH active_users AS (SELECT `id`, `name` FROM `users` WHERE status = ?) SELECT `id`, `name` FROM `active_users`",
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

func TestSelectUnion(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *SelectQuery
		expected string
	}{
		{
			name: "union all",
			build: func() *SelectQuery {
				q1 := Select("id", "name").From("users")
				q2 := Select("id", "name").From("admins")
				return q1.UnionAll(q2)
			},
			expected: "SELECT `id`, `name` FROM `users` UNION ALL SELECT `id`, `name` FROM `admins`",
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

func TestSelectString(t *testing.T) {
	q := Select("id", "name").
		From("users").
		Where("status = ?", "active").
		Where("age >= ?", 18).
		Limit(10)

	str := q.String()
	expected := "SELECT `id`, `name` FROM `users` WHERE status = 'active' AND age >= 18 LIMIT 10"

	if str != expected {
		t.Errorf("\nexpected: %s\ngot:      %s", expected, str)
	}
}

func TestSelectWhereIn(t *testing.T) {
	q := Select("id", "name").
		From("users").
		WhereIn("id", []int{1, 2, 3})

	sql, args, err := q.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "SELECT `id`, `name` FROM `users` WHERE `id` IN (?, ?, ?)"
	if sql != expected {
		t.Errorf("\nexpected: %s\ngot:      %s", expected, sql)
	}

	if len(args) != 3 {
		t.Errorf("expected 3 args, got %d", len(args))
	}
}
