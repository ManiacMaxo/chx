package query

import (
	"testing"
)

func TestInsertBasic(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *InsertQuery
		expected string
		argCount int
	}{
		{
			name: "simple insert",
			build: func() *InsertQuery {
				return Insert("users").
					Columns("id", "name", "email").
					Values(1, "John", "john@example.com")
			},
			expected: "INSERT INTO `users` (`id`, `name`, `email`) VALUES (?, ?, ?)",
			argCount: 3,
		},
		{
			name: "insert multiple rows",
			build: func() *InsertQuery {
				return Insert("users").
					Columns("id", "name").
					Values(1, "John").
					Values(2, "Jane")
			},
			expected: "INSERT INTO `users` (`id`, `name`) VALUES (?, ?), (?, ?)",
			argCount: 4,
		},
		{
			name: "insert from select",
			build: func() *InsertQuery {
				selectQuery := Select("id", "name").From("old_users").Where("active = ?", true)
				return Insert("users").
					Columns("id", "name").
					Select(selectQuery)
			},
			expected: "INSERT INTO `users` (`id`, `name`) SELECT `id`, `name` FROM `old_users` WHERE active = ?",
			argCount: 1,
		},
		{
			name: "insert with settings",
			build: func() *InsertQuery {
				return Insert("users").
					Columns("id", "name").
					Values(1, "John").
					Setting("async_insert = 1")
			},
			expected: "INSERT INTO `users` (`id`, `name`) VALUES (?, ?) SETTINGS async_insert = 1",
			argCount: 2,
		},
		{
			name: "insert with format",
			build: func() *InsertQuery {
				return Insert("users").
					Columns("id", "name").
					Format("JSONEachRow")
			},
			expected: "INSERT INTO `users` (`id`, `name`) FORMAT JSONEachRow",
			argCount: 0,
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

func TestInsertStruct(t *testing.T) {
	type User struct {
		ID    int    `ch:"id"`
		Name  string `ch:"name"`
		Email string `ch:"email"`
	}

	user := User{ID: 1, Name: "John", Email: "john@example.com"}

	q := Insert("users").Struct(user)
	sql, args, err := q.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "INSERT INTO `users` (`id`, `name`, `email`) VALUES (?, ?, ?)"
	if sql != expected {
		t.Errorf("\nexpected: %s\ngot:      %s", expected, sql)
	}

	if len(args) != 3 {
		t.Errorf("expected 3 args, got %d", len(args))
	}
}

func TestInsertMap(t *testing.T) {
	data := map[string]any{
		"id":   1,
		"name": "John",
	}

	q := Insert("users").Map(data)
	sql, args, err := q.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Map iteration order is not deterministic, so we just check basic structure
	if len(args) != 2 {
		t.Errorf("expected 2 args, got %d", len(args))
	}
	if sql == "" {
		t.Error("expected non-empty SQL")
	}
}

func TestInsertString(t *testing.T) {
	q := Insert("users").
		Columns("id", "name").
		Values(1, "John")

	str := q.String()
	expected := "INSERT INTO `users` (`id`, `name`) VALUES (1, 'John')"

	if str != expected {
		t.Errorf("\nexpected: %s\ngot:      %s", expected, str)
	}
}
