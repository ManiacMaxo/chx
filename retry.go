package chx

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/ManiacMaxo/chx/query"
)

type RetryConfig struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	RetryExec    bool
	IsRetryable  func(error) bool
}

func (c *RetryConfig) withDefaults() *RetryConfig {
	if c == nil {
		return nil
	}
	cfg := *c
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.InitialDelay == 0 {
		cfg.InitialDelay = 100 * time.Millisecond
	}
	if cfg.MaxDelay == 0 {
		cfg.MaxDelay = 5 * time.Second
	}
	return &cfg
}

func defaultIsRetryable(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, io.EOF) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var ex *clickhouse.Exception
	if errors.As(err, &ex) {
		switch ex.Code {
		case 159, // TOO_SLOW
			203, // NETWORK_ERROR
			210, // SOCKET_TIMEOUT
			242, // SESSION_NOT_FOUND
			279, // TOO_MANY_SIMULTANEOUS_QUERIES
			341, // NO_AVAILABLE_CONNECTION
			352, // CONNECTION_WITH_SLAVE_FAILED
			375: // INCONSISTENT_CHAIN_PARTS
			return true
		}
	}

	msg := err.Error()
	if strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "use of closed network connection") {
		return true
	}

	return false
}

type RetryClient struct {
	client *Client
	mu     sync.Mutex
	cfg    *RetryConfig
	dial   func() (driver.Conn, error)
	isR    func(error) bool
}

func OpenWithRetry(opt *clickhouse.Options, cfg RetryConfig) (*RetryClient, error) {
	conn, err := clickhouse.Open(opt)
	if err != nil {
		return nil, err
	}

	cfg = *cfg.withDefaults()
	rc := &RetryClient{
		client: New(conn),
		cfg:    &cfg,
		dial:   func() (driver.Conn, error) { return clickhouse.Open(opt) },
	}
	if cfg.IsRetryable != nil {
		rc.isR = cfg.IsRetryable
	} else {
		rc.isR = defaultIsRetryable
	}
	return rc, nil
}

var _ query.Executor = (*RetryClient)(nil)

func (rc *RetryClient) Client() *Client {
	return rc.client
}

func (rc *RetryClient) Conn() driver.Conn {
	return rc.client.conn
}

func (rc *RetryClient) Close() error {
	return rc.client.Close()
}

func (rc *RetryClient) Ping(ctx context.Context) error {
	return rc.withRetry(ctx, func() error {
		return rc.client.conn.Ping(ctx)
	})
}

func (rc *RetryClient) Query(ctx context.Context, sql string, args ...any) (driver.Rows, error) {
	return rc.withRetryRows(ctx, func() (driver.Rows, error) {
		return rc.client.conn.Query(ctx, sql, args...)
	})
}

func (rc *RetryClient) QueryRow(ctx context.Context, sql string, args ...any) driver.Row {
	row, err := rc.withRetryRow(ctx, func() driver.Row {
		return rc.client.conn.QueryRow(ctx, sql, args...)
	})
	if err != nil {
		return errorRow{err: err}
	}
	return row
}

func (rc *RetryClient) Exec(ctx context.Context, sql string, args ...any) error {
	if !rc.cfg.RetryExec {
		return rc.client.Exec(ctx, sql, args...)
	}
	return rc.withRetry(ctx, func() error {
		return rc.client.conn.Exec(ctx, sql, args...)
	})
}

func (rc *RetryClient) PrepareBatch(ctx context.Context, sql string) (driver.Batch, error) {
	if !rc.cfg.RetryExec {
		return rc.client.PrepareBatch(ctx, sql)
	}
	var batch driver.Batch
	err := rc.withRetry(ctx, func() error {
		var err error
		batch, err = rc.client.conn.PrepareBatch(ctx, sql)
		return err
	})
	return batch, err
}

func (rc *RetryClient) Select(columns ...string) *query.SelectQuery {
	return query.NewSelect(rc).Columns(columns...)
}

func (rc *RetryClient) SelectExpr(expr string, args ...any) *query.SelectQuery {
	return query.NewSelect(rc).ColumnExpr(expr, args...)
}

func (rc *RetryClient) Insert(table string) *query.InsertQuery {
	return query.NewInsert(rc).Into(table)
}

func (rc *RetryClient) Update(table string) *query.UpdateQuery {
	return query.NewUpdate(rc).Table(table)
}

func (rc *RetryClient) Delete(table string) *query.DeleteQuery {
	return query.NewDelete(rc).From(table)
}

func (rc *RetryClient) CreateTable(table string) *query.CreateTableQuery {
	return query.NewCreateTable(rc).Table(table)
}

func (rc *RetryClient) CreateView(name string) *query.CreateViewQuery {
	return query.NewCreateView(rc).View(name)
}

func (rc *RetryClient) CreateMaterializedView(name string) *query.CreateViewQuery {
	return query.NewCreateView(rc).View(name).Materialized()
}

func (rc *RetryClient) Alter(table string) *query.AlterQuery {
	return query.NewAlter(rc).Table(table)
}

func (rc *RetryClient) DropTable(table string) *query.DropQuery {
	return query.NewDrop(rc).Table(table)
}

func (rc *RetryClient) DropView(name string) *query.DropQuery {
	return query.NewDrop(rc).View(name)
}

func (rc *RetryClient) DropDatabase(name string) *query.DropQuery {
	return query.NewDrop(rc).Database(name)
}

func (rc *RetryClient) Truncate(table string) *query.TruncateQuery {
	return query.NewTruncate(rc).Table(table)
}

func (rc *RetryClient) Optimize(table string) *query.OptimizeQuery {
	return query.NewOptimize(rc).Table(table)
}

func (rc *RetryClient) Raw(ctx context.Context, sql string, args ...any) (driver.Rows, error) {
	return rc.Query(ctx, sql, args...)
}

func (rc *RetryClient) RawExec(ctx context.Context, sql string, args ...any) error {
	return rc.Exec(ctx, sql, args...)
}

func (rc *RetryClient) reconnect() error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	_ = rc.client.conn.Close()

	conn, err := rc.dial()
	if err != nil {
		return err
	}

	rc.client.conn = conn
	return nil
}

func (rc *RetryClient) backoff(attempt int) time.Duration {
	delay := rc.cfg.InitialDelay << uint(attempt)
	if delay > rc.cfg.MaxDelay || delay < 0 {
		delay = rc.cfg.MaxDelay
	}
	return time.Duration(delay)
}

func (rc *RetryClient) withRetry(ctx context.Context, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt <= rc.cfg.MaxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if !rc.isR(err) {
			return err
		}

		if attempt == rc.cfg.MaxRetries {
			break
		}

		select {
		case <-time.After(rc.backoff(attempt)):
		case <-ctx.Done():
			return ctx.Err()
		}

		if err := rc.reconnect(); err != nil {
			lastErr = err
			continue
		}
	}

	return lastErr
}

func (rc *RetryClient) withRetryRows(ctx context.Context, fn func() (driver.Rows, error)) (driver.Rows, error) {
	var lastErr error
	for attempt := 0; attempt <= rc.cfg.MaxRetries; attempt++ {
		rows, err := fn()
		if err == nil {
			return rows, nil
		}

		lastErr = err

		if !rc.isR(err) {
			return nil, err
		}

		if attempt == rc.cfg.MaxRetries {
			break
		}

		select {
		case <-time.After(rc.backoff(attempt)):
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		if err := rc.reconnect(); err != nil {
			lastErr = err
			continue
		}
	}

	return nil, lastErr
}

func (rc *RetryClient) withRetryRow(ctx context.Context, fn func() driver.Row) (driver.Row, error) {
	var lastErr error
	for attempt := 0; attempt <= rc.cfg.MaxRetries; attempt++ {
		row := fn()
		if err := row.Err(); err == nil {
			return row, nil
		} else {
			lastErr = err
		}

		if !rc.isR(lastErr) {
			return nil, lastErr
		}

		if attempt == rc.cfg.MaxRetries {
			break
		}

		select {
		case <-time.After(rc.backoff(attempt)):
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		if err := rc.reconnect(); err != nil {
			lastErr = err
			continue
		}
	}

	return nil, lastErr
}

type errorRow struct {
	err error
}

func (e errorRow) Err() error           { return e.err }
func (e errorRow) Scan(...any) error    { return e.err }
func (e errorRow) ScanStruct(any) error { return e.err }
