package query

import (
	"testing"
)

func TestDeleteQuery(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *DeleteQuery
		expected string
		argCount int
	}{
		{
			name: "simple delete",
			build: func() *DeleteQuery {
				return Delete("users").Where("id = ?", 123)
			},
			expected: "DELETE FROM `users` WHERE id = ?",
			argCount: 1,
		},
		{
			name: "delete with multiple conditions",
			build: func() *DeleteQuery {
				return Delete("users").
					Where("status = ?", "inactive").
					Where("last_login < ?", "2023-01-01")
			},
			expected: "DELETE FROM `users` WHERE status = ? AND last_login < ?",
			argCount: 2,
		},
		{
			name: "delete with in clause",
			build: func() *DeleteQuery {
				return Delete("users").WhereIn("id", []int{1, 2, 3})
			},
			expected: "DELETE FROM `users` WHERE `id` IN (?, ?, ?)",
			argCount: 3,
		},
		{
			name: "delete on cluster",
			build: func() *DeleteQuery {
				return Delete("users").
					OnCluster("cluster1").
					Where("id = ?", 123)
			},
			expected: "DELETE FROM `users` ON CLUSTER `cluster1` WHERE id = ?",
			argCount: 1,
		},
		{
			name: "delete with settings",
			build: func() *DeleteQuery {
				return Delete("users").
					Where("id = ?", 123).
					Setting("mutations_sync = 1")
			},
			expected: "DELETE FROM `users` WHERE id = ? SETTINGS mutations_sync = 1",
			argCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.build()
			sql, args, err := q.Build()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sql != tt.expected {
				t.Errorf("\nexpected: %s\ngot:      %s", tt.expected, sql)
			}
			if len(args) != tt.argCount {
				t.Errorf("expected %d args, got %d", tt.argCount, len(args))
			}
		})
	}
}

func TestDeleteQueryErrors(t *testing.T) {
	// Delete without WHERE should fail
	q := Delete("users")
	_, _, err := q.Build()
	if err == nil {
		t.Error("expected error for DELETE without WHERE")
	}
}

func TestUpdateQuery(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *UpdateQuery
		expected string
		argCount int
	}{
		{
			name: "simple update",
			build: func() *UpdateQuery {
				return Update("users").
					Set("status", "inactive").
					Where("id = ?", 123)
			},
			expected: "ALTER TABLE `users` UPDATE `status` = ? WHERE id = ?",
			argCount: 2,
		},
		{
			name: "update multiple columns",
			build: func() *UpdateQuery {
				return Update("users").
					Set("status", "inactive").
					Set("updated_at", "now()").
					Where("id = ?", 123)
			},
			expected: "ALTER TABLE `users` UPDATE `status` = ?, `updated_at` = ? WHERE id = ?",
			argCount: 3,
		},
		{
			name: "update with expression",
			build: func() *UpdateQuery {
				return Update("users").
					SetExpr("counter = counter + 1").
					Where("id = ?", 123)
			},
			expected: "ALTER TABLE `users` UPDATE counter = counter + 1 WHERE id = ?",
			argCount: 1,
		},
		{
			name: "update on cluster",
			build: func() *UpdateQuery {
				return Update("users").
					OnCluster("cluster1").
					Set("status", "inactive").
					Where("id = ?", 123)
			},
			expected: "ALTER TABLE `users` ON CLUSTER `cluster1` UPDATE `status` = ? WHERE id = ?",
			argCount: 2,
		},
		{
			name: "update with settings",
			build: func() *UpdateQuery {
				return Update("users").
					Set("status", "inactive").
					Where("id = ?", 123).
					Setting("mutations_sync = 1")
			},
			expected: "ALTER TABLE `users` UPDATE `status` = ? WHERE id = ? SETTINGS mutations_sync = 1",
			argCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.build()
			sql, args, err := q.Build()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sql != tt.expected {
				t.Errorf("\nexpected: %s\ngot:      %s", tt.expected, sql)
			}
			if len(args) != tt.argCount {
				t.Errorf("expected %d args, got %d", tt.argCount, len(args))
			}
		})
	}
}

func TestUpdateQueryErrors(t *testing.T) {
	// Update without SET should fail
	q := Update("users").Where("id = ?", 123)
	_, _, err := q.Build()
	if err == nil {
		t.Error("expected error for UPDATE without SET")
	}

	// Update without WHERE should fail
	q = Update("users").Set("status", "inactive")
	_, _, err = q.Build()
	if err == nil {
		t.Error("expected error for UPDATE without WHERE")
	}
}

func TestDeleteString(t *testing.T) {
	q := Delete("users").Where("id = ?", 123)
	str := q.String()
	expected := "DELETE FROM `users` WHERE id = 123"

	if str != expected {
		t.Errorf("\nexpected: %s\ngot:      %s", expected, str)
	}
}

func TestUpdateString(t *testing.T) {
	q := Update("users").
		Set("status", "inactive").
		Where("id = ?", 123)
	str := q.String()
	expected := "ALTER TABLE `users` UPDATE `status` = 'inactive' WHERE id = 123"

	if str != expected {
		t.Errorf("\nexpected: %s\ngot:      %s", expected, str)
	}
}
