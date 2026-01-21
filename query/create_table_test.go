package query

import (
	"testing"
)

func TestCreateTableBasic(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *CreateTableQuery
		expected string
	}{
		{
			name: "simple create table",
			build: func() *CreateTableQuery {
				return CreateTable("users").
					Column("id", "UInt64").Add().
					Column("name", "String").Add().
					MergeTree().
					OrderBy("id")
			},
			expected: "CREATE TABLE `users` (`id` UInt64, `name` String) ENGINE = MergeTree() ORDER BY (`id`)",
		},
		{
			name: "create table if not exists",
			build: func() *CreateTableQuery {
				return CreateTable("users").
					IfNotExists().
					Column("id", "UInt64").Add().
					MergeTree().
					OrderBy("id")
			},
			expected: "CREATE TABLE IF NOT EXISTS `users` (`id` UInt64) ENGINE = MergeTree() ORDER BY (`id`)",
		},
		{
			name: "create table with partition",
			build: func() *CreateTableQuery {
				return CreateTable("events").
					Column("id", "UInt64").Add().
					Column("timestamp", "DateTime").Add().
					MergeTree().
					PartitionBy("toYYYYMM(timestamp)").
					OrderBy("id", "timestamp")
			},
			expected: "CREATE TABLE `events` (`id` UInt64, `timestamp` DateTime) ENGINE = MergeTree() PARTITION BY toYYYYMM(timestamp) ORDER BY (`id`, `timestamp`)",
		},
		{
			name: "create table with ttl",
			build: func() *CreateTableQuery {
				return CreateTable("logs").
					Column("id", "UInt64").Add().
					Column("timestamp", "DateTime").Add().
					MergeTree().
					OrderBy("id").
					TTL("timestamp + INTERVAL 30 DAY")
			},
			expected: "CREATE TABLE `logs` (`id` UInt64, `timestamp` DateTime) ENGINE = MergeTree() ORDER BY (`id`) TTL timestamp + INTERVAL 30 DAY",
		},
		{
			name: "create table with replacing merge tree",
			build: func() *CreateTableQuery {
				return CreateTable("users").
					Column("id", "UInt64").Add().
					Column("version", "UInt64").Add().
					ReplacingMergeTree("version").
					OrderBy("id")
			},
			expected: "CREATE TABLE `users` (`id` UInt64, `version` UInt64) ENGINE = ReplacingMergeTree(version) ORDER BY (`id`)",
		},
		{
			name: "create table on cluster",
			build: func() *CreateTableQuery {
				return CreateTable("users").
					IfNotExists().
					OnCluster("cluster1").
					Column("id", "UInt64").Add().
					ReplicatedMergeTree("/clickhouse/tables/{shard}/users", "{replica}").
					OrderBy("id")
			},
			expected: "CREATE TABLE IF NOT EXISTS `users` ON CLUSTER `cluster1` (`id` UInt64) ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/users', '{replica}') ORDER BY (`id`)",
		},
		{
			name: "create table with column options",
			build: func() *CreateTableQuery {
				return CreateTable("users").
					Column("id", "UInt64").Add().
					Column("name", "String").Default("''").Codec("ZSTD(1)").Comment("User name").Add().
					Column("created_at", "DateTime").Default("now()").Add().
					MergeTree().
					OrderBy("id")
			},
			expected: "CREATE TABLE `users` (`id` UInt64, `name` String DEFAULT '' CODEC(ZSTD(1)) COMMENT 'User name', `created_at` DateTime DEFAULT now()) ENGINE = MergeTree() ORDER BY (`id`)",
		},
		{
			name: "create table with settings",
			build: func() *CreateTableQuery {
				return CreateTable("users").
					Column("id", "UInt64").Add().
					MergeTree().
					OrderBy("id").
					Setting("index_granularity = 8192")
			},
			expected: "CREATE TABLE `users` (`id` UInt64) ENGINE = MergeTree() ORDER BY (`id`) SETTINGS index_granularity = 8192",
		},
		{
			name: "create table with comment",
			build: func() *CreateTableQuery {
				return CreateTable("users").
					Column("id", "UInt64").Add().
					MergeTree().
					OrderBy("id").
					Comment("User accounts table")
			},
			expected: "CREATE TABLE `users` (`id` UInt64) ENGINE = MergeTree() ORDER BY (`id`) COMMENT 'User accounts table'",
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

func TestCreateTableEngines(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *CreateTableQuery
		contains string
	}{
		{
			name: "MergeTree",
			build: func() *CreateTableQuery {
				return CreateTable("t").Column("id", "UInt64").Add().MergeTree().OrderBy("id")
			},
			contains: "ENGINE = MergeTree()",
		},
		{
			name: "ReplacingMergeTree",
			build: func() *CreateTableQuery {
				return CreateTable("t").Column("id", "UInt64").Add().ReplacingMergeTree().OrderBy("id")
			},
			contains: "ENGINE = ReplacingMergeTree()",
		},
		{
			name: "ReplacingMergeTree with version",
			build: func() *CreateTableQuery {
				return CreateTable("t").Column("id", "UInt64").Add().ReplacingMergeTree("ver").OrderBy("id")
			},
			contains: "ENGINE = ReplacingMergeTree(ver)",
		},
		{
			name: "SummingMergeTree",
			build: func() *CreateTableQuery {
				return CreateTable("t").Column("id", "UInt64").Add().SummingMergeTree("value").OrderBy("id")
			},
			contains: "ENGINE = SummingMergeTree((value))",
		},
		{
			name: "AggregatingMergeTree",
			build: func() *CreateTableQuery {
				return CreateTable("t").Column("id", "UInt64").Add().AggregatingMergeTree().OrderBy("id")
			},
			contains: "ENGINE = AggregatingMergeTree()",
		},
		{
			name: "CollapsingMergeTree",
			build: func() *CreateTableQuery {
				return CreateTable("t").Column("id", "UInt64").Add().CollapsingMergeTree("sign").OrderBy("id")
			},
			contains: "ENGINE = CollapsingMergeTree(sign)",
		},
		{
			name: "Memory",
			build: func() *CreateTableQuery {
				return CreateTable("t").Column("id", "UInt64").Add().Memory()
			},
			contains: "ENGINE = Memory",
		},
		{
			name: "Log",
			build: func() *CreateTableQuery {
				return CreateTable("t").Column("id", "UInt64").Add().Log()
			},
			contains: "ENGINE = Log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.build()
			sql, _, err := q.Build()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !containsSubstring(sql, tt.contains) {
				t.Errorf("\nexpected SQL to contain: %s\ngot: %s", tt.contains, sql)
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s[1:], substr) || s[:len(substr)] == substr)
}
