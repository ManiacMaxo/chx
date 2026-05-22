# chx - Clickhouse (Extended)

A fluent query builder for ClickHouse in Go, inspired by [uptrace/go-clickhouse](https://github.com/uptrace/go-clickhouse).

## Features

- **Fluent API** - Method chaining for building queries
- **Type-safe** - Identifiers are quoted by default, raw SQL via `*Expr` methods
- **Complete ClickHouse support** - All keywords, clauses, and engine types
- **Minimal dependencies** - Only requires the official ClickHouse driver

## Installation

```bash
go get github.com/ManiacMaxo/chx
```

## Quick Start

```go
package main

import (
    "context"
    "log"

    "github.com/ClickHouse/clickhouse-go/v2"
    ch "github.com/ManiacMaxo/chx"
)

func main() {
    client, err := ch.Open(&clickhouse.Options{
        Addr: []string{"localhost:9000"},
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    // Simple SELECT
    rows, err := client.Select("id", "name", "email").
        From("users").
        Where("status = ?", "active").
        OrderByDesc("created_at").
        Limit(10).
        Query(ctx)
}
```

## Query Builders

### SELECT

```go
// Basic SELECT
q := client.Select("id", "name", "email").
    From("users").
    Where("status = ?", "active").
    Where("age >= ?", 18).
    OrderByDesc("created_at").
    Limit(10)

// With ClickHouse-specific features
q := client.Select("user_id", "event").
    ColumnExpr("count(*) as cnt").
    From("events").
    Final().                              // FINAL modifier
    Sample(0.1).                          // SAMPLE 10%
    Prewhere("project_id = ?", 123).      // PREWHERE
    Where("timestamp > ?", startDate).
    GroupBy("user_id", "event").
    WithTotals().                         // WITH TOTALS
    Having("cnt > ?", 100).
    OrderByDesc("cnt").
    LimitBy(5, "user_id").                // LIMIT BY
    Setting("max_threads = 4")            // SETTINGS

// CTEs (Common Table Expressions)
activeUsers := client.Select("id", "name").
    From("users").
    Where("status = ?", "active")

q := client.Select("id", "name").
    With("active_users", activeUsers).
    From("active_users")

// JOINs
q := client.Select().
    ColumnExpr("u.id, u.name, count(o.id) as order_count").
    FromExpr("users AS u").
    LeftJoin("orders").As("o").On("o.user_id = u.id").End().
    GroupByExpr("u.id, u.name")

// All JOIN types supported:
// - InnerJoin, LeftJoin, RightJoin, FullJoin, CrossJoin
// - GlobalJoin (for distributed queries)
// - AnyJoin, AllJoin, AsofJoin
// - SemiJoin, AntiJoin

// ARRAY JOIN
q := client.Select("id", "tag").
    From("posts").
    ArrayJoin("tags")

// UNION
q1 := client.Select("id", "name").From("admins")
q2 := client.Select("id", "name").From("users")
q := q1.UnionAll(q2)
```

### INSERT

```go
// Insert values
err := client.Insert("users").
    Columns("id", "name", "email").
    Values(1, "John", "john@example.com").
    Values(2, "Jane", "jane@example.com").
    Exec(ctx)

// Insert from SELECT
selectQuery := client.Select("id", "name", "email").
    From("old_users").
    Where("migrated = ?", false)

err := client.Insert("users").
    Columns("id", "name", "email").
    Select(selectQuery).
    Exec(ctx)

// Insert from struct
type User struct {
    ID    int    `ch:"id"`
    Name  string `ch:"name"`
    Email string `ch:"email"`
}

err := client.Insert("users").
    Struct(User{ID: 1, Name: "John", Email: "john@example.com"}).
    Exec(ctx)
```

### CREATE TABLE

```go
// MergeTree
err := client.CreateTable("events").
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
    Setting("index_granularity = 8192").
    Exec(ctx)

// Replicated on cluster
err := client.CreateTable("events").
    IfNotExists().
    OnCluster("production").
    Column("id", "UInt64").Add().
    Column("data", "String").Add().
    ReplicatedMergeTree("/clickhouse/tables/{shard}/events", "{replica}").
    OrderBy("id").
    Exec(ctx)

// Supported engines:
// MergeTree, ReplacingMergeTree, SummingMergeTree, AggregatingMergeTree,
// CollapsingMergeTree, VersionedCollapsingMergeTree, ReplicatedMergeTree,
// Memory, Log, TinyLog, StripeLog, Null, Buffer, Distributed
```

### CREATE MATERIALIZED VIEW

```go
selectQuery := client.Select().
    ColumnExpr("user_id, toDate(timestamp) as date, count(*) as cnt").
    From("events").
    GroupBy("user_id", "date")

err := client.CreateMaterializedView("daily_events").
    IfNotExists().
    To("daily_events_data").
    Populate().
    As(selectQuery).
    Exec(ctx)
```

### ALTER TABLE

```go
// Add column
err := client.Alter("users").
    AddColumn("phone", "Nullable(String)").After("email").End().
    Exec(ctx)

// Multiple alterations
err := client.Alter("users").
    AddColumn("phone", "String").End().
    DropColumn("legacy_field").
    ModifyColumn("age", "UInt32").
    Exec(ctx)

// Partition operations
err := client.Alter("events").
    DropPartition("202301").
    DropPartition("202302").
    Exec(ctx)

// TTL modification
err := client.Alter("logs").
    ModifyTTL("timestamp + INTERVAL 60 DAY").
    Exec(ctx)
```

### DELETE (Lightweight)

```go
err := client.Delete("users").
    Where("status = ?", "deleted").
    Where("deleted_at < ?", "2024-01-01").
    Exec(ctx)
```

### UPDATE (Lightweight)

```go
err := client.Update("users").
    Set("status", "inactive").
    SetExpr("updated_at = now()").
    Where("last_login < ?", "2023-01-01").
    Exec(ctx)
```

### DROP / TRUNCATE / OPTIMIZE

```go
// Drop table
err := client.DropTable("old_events").
    IfExists().
    OnCluster("production").
    Sync().
    Exec(ctx)

// Truncate
err := client.Truncate("events").
    IfExists().
    Exec(ctx)

// Optimize
err := client.Optimize("events").
    Partition("202401").
    Final().
    Deduplicate().
    Exec(ctx)
```

## Expression Helpers

```go
import "github.com/ManiacMaxo/chx/query"

// IN clause
q := client.Select("id", "name").
    From("users").
    WhereIn("id", []int{1, 2, 3, 4, 5})

// Using query.In directly
q := client.Select("*").
    From("users").
    Where("id IN ?", query.In([]int{1, 2, 3}))

// Array literal
q := client.Select("*").
    From("users").
    Where("tags && ?", query.Array([]string{"admin", "moderator"}))

// Tuple
q := client.Select("*").
    From("users").
    Where("(id, name) = ?", query.Tuple(1, "John"))

// Safe identifier (quoted)
q := client.Select().
    ColumnExpr("? as alias", query.Ident("column_name"))

// Raw expression (unquoted, use with caution)
q := client.Select().
    ColumnExpr("?", query.Raw("now()"))
```

## Building Queries Without Execution

```go
// Get SQL and arguments
sql, args, err := client.Select("id", "name").
    From("users").
    Where("status = ?", "active").
    Build()

// Get interpolated SQL (for debugging)
debugSQL := client.Select("id", "name").
    From("users").
    Where("status = ?", "active").
    String()
```

## Auto-Reconnection

`RetryClient` wraps a `Client` with automatic reconnection on transient network errors. It implements the same `Executor` interface, so all query builders work transparently.

```go
rc, err := ch.OpenWithRetry(&clickhouse.Options{
    Addr: []string{"localhost:9000"},
}, ch.RetryConfig{
    MaxRetries: 3, // 0 uses default (3)
})
if err != nil {
    log.Fatal(err)
}
defer rc.Close()

// All query builders work the same — retries happen transparently
rows, err := rc.Select("id", "name").
    From("users").
    Query(ctx)

// Access the underlying *Client if needed
rc.Client()
```

### Configuration

| Field | Default | Description |
|-------|---------|-------------|
| `MaxRetries` | `3` | Maximum number of retry attempts |
| `InitialDelay` | `100ms` | Delay before first retry |
| `MaxDelay` | `5s` | Maximum backoff delay |
| `RetryExec` | `false` | Enable retries for `Exec`/`PrepareBatch` (write operations) |
| `IsRetryable` | built-in | Custom function to classify retryable errors |

Uses exponential backoff (doubles each attempt, capped at `MaxDelay`). Thread-safe for concurrent use.

By default, only `Query`, `QueryRow`, and `Ping` are retried. Set `RetryExec: true` to also retry `Exec` and `PrepareBatch` (be mindful of idempotency).

## ClickHouse Features Support

| Feature | Status |
|---------|--------|
| WITH (CTEs) | ✅ |
| SELECT DISTINCT / DISTINCT ON | ✅ |
| FINAL | ✅ |
| SAMPLE | ✅ |
| ARRAY JOIN / LEFT ARRAY JOIN | ✅ |
| All JOIN types (GLOBAL, ANY, ALL, ASOF, SEMI, ANTI) | ✅ |
| PREWHERE | ✅ |
| WHERE | ✅ |
| GROUP BY (WITH ROLLUP/CUBE/TOTALS) | ✅ |
| HAVING | ✅ |
| WINDOW | ✅ |
| QUALIFY | ✅ |
| ORDER BY (WITH FILL) | ✅ |
| LIMIT / OFFSET | ✅ |
| LIMIT BY | ✅ |
| UNION / INTERSECT / EXCEPT | ✅ |
| INTO OUTFILE | ✅ |
| FORMAT | ✅ |
| SETTINGS | ✅ |
| All MergeTree engines | ✅ |
| Replicated tables | ✅ |
| ON CLUSTER | ✅ |
| TTL | ✅ |
| Materialized Views | ✅ |
| Lightweight DELETE/UPDATE | ✅ |

## License

MIT
