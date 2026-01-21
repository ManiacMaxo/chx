package query

import (
	"strings"
	"testing"
)

func TestAlterTable(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *AlterQuery
		expected string
	}{
		{
			name: "add column",
			build: func() *AlterQuery {
				return Alter("users").
					AddColumn("email", "String").End()
			},
			expected: "ALTER TABLE `users` ADD COLUMN `email` String",
		},
		{
			name: "add column after",
			build: func() *AlterQuery {
				return Alter("users").
					AddColumn("email", "String").After("name").End()
			},
			expected: "ALTER TABLE `users` ADD COLUMN `email` String AFTER `name`",
		},
		{
			name: "add column first",
			build: func() *AlterQuery {
				return Alter("users").
					AddColumn("id", "UInt64").First().End()
			},
			expected: "ALTER TABLE `users` ADD COLUMN `id` UInt64 FIRST",
		},
		{
			name: "add column with default",
			build: func() *AlterQuery {
				return Alter("users").
					AddColumn("status", "String").Default("'active'").End()
			},
			expected: "ALTER TABLE `users` ADD COLUMN `status` String DEFAULT 'active'",
		},
		{
			name: "drop column",
			build: func() *AlterQuery {
				return Alter("users").DropColumn("email")
			},
			expected: "ALTER TABLE `users` DROP COLUMN `email`",
		},
		{
			name: "rename column",
			build: func() *AlterQuery {
				return Alter("users").RenameColumn("email", "user_email")
			},
			expected: "ALTER TABLE `users` RENAME COLUMN `email` TO `user_email`",
		},
		{
			name: "modify column",
			build: func() *AlterQuery {
				return Alter("users").ModifyColumn("age", "UInt32")
			},
			expected: "ALTER TABLE `users` MODIFY COLUMN `age` UInt32",
		},
		{
			name: "comment column",
			build: func() *AlterQuery {
				return Alter("users").CommentColumn("email", "User's email address")
			},
			expected: "ALTER TABLE `users` COMMENT COLUMN `email` 'User\\'s email address'",
		},
		{
			name: "add index",
			build: func() *AlterQuery {
				return Alter("users").AddIndex("idx_email", "email", "bloom_filter", 4)
			},
			expected: "ALTER TABLE `users` ADD INDEX `idx_email` email TYPE bloom_filter GRANULARITY 4",
		},
		{
			name: "drop index",
			build: func() *AlterQuery {
				return Alter("users").DropIndex("idx_email")
			},
			expected: "ALTER TABLE `users` DROP INDEX `idx_email`",
		},
		{
			name: "drop partition",
			build: func() *AlterQuery {
				return Alter("events").DropPartition("202301")
			},
			expected: "ALTER TABLE `events` DROP PARTITION 202301",
		},
		{
			name: "detach partition",
			build: func() *AlterQuery {
				return Alter("events").DetachPartition("202301")
			},
			expected: "ALTER TABLE `events` DETACH PARTITION 202301",
		},
		{
			name: "attach partition",
			build: func() *AlterQuery {
				return Alter("events").AttachPartition("202301")
			},
			expected: "ALTER TABLE `events` ATTACH PARTITION 202301",
		},
		{
			name: "modify ttl",
			build: func() *AlterQuery {
				return Alter("logs").ModifyTTL("timestamp + INTERVAL 60 DAY")
			},
			expected: "ALTER TABLE `logs` MODIFY TTL timestamp + INTERVAL 60 DAY",
		},
		{
			name: "remove ttl",
			build: func() *AlterQuery {
				return Alter("logs").RemoveTTL()
			},
			expected: "ALTER TABLE `logs` REMOVE TTL",
		},
		{
			name: "delete mutation",
			build: func() *AlterQuery {
				return Alter("users").Delete("status = ?", "deleted")
			},
			expected: "ALTER TABLE `users` DELETE WHERE status = ?",
		},
		{
			name: "update mutation",
			build: func() *AlterQuery {
				return Alter("users").UpdateMutation("status = 'inactive'", "last_login < ?", "2023-01-01")
			},
			expected: "ALTER TABLE `users` UPDATE status = 'inactive' WHERE last_login < ?",
		},
		{
			name: "multiple alterations",
			build: func() *AlterQuery {
				return Alter("users").
					AddColumn("email", "String").End().
					AddColumn("phone", "String").End()
			},
			expected: "ALTER TABLE `users` ADD COLUMN `email` String, ADD COLUMN `phone` String",
		},
		{
			name: "on cluster",
			build: func() *AlterQuery {
				return Alter("users").
					OnCluster("cluster1").
					AddColumn("email", "String").End()
			},
			expected: "ALTER TABLE `users` ON CLUSTER `cluster1` ADD COLUMN `email` String",
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

func TestAlterTableString(t *testing.T) {
	q := Alter("users").Delete("id = ?", 123)
	str := q.String()

	if !strings.Contains(str, "DELETE WHERE id = 123") {
		t.Errorf("expected interpolated value, got: %s", str)
	}
}
