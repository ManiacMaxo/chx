package chx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type mockConn struct {
	queryFn        func(ctx context.Context, query string, args ...any) (driver.Rows, error)
	queryRowFn     func(ctx context.Context, query string, args ...any) driver.Row
	execFn         func(ctx context.Context, query string, args ...any) error
	pingFn         func(ctx context.Context) error
	prepareBatchFn func(ctx context.Context, query string, opts ...driver.PrepareBatchOption) (driver.Batch, error)
	closeErr       error
}

func (m *mockConn) Contributors() []string                                                { return nil }
func (m *mockConn) ServerVersion() (*driver.ServerVersion, error)                         { return nil, nil }
func (m *mockConn) Select(ctx context.Context, dest any, query string, args ...any) error  { return nil }
func (m *mockConn) Stats() driver.Stats                                                    { return driver.Stats{} }
func (m *mockConn) AsyncInsert(ctx context.Context, query string, wait bool, args ...any) error {
	return nil
}
func (m *mockConn) Close() error { return m.closeErr }
func (m *mockConn) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	return m.queryFn(ctx, query, args...)
}
func (m *mockConn) QueryRow(ctx context.Context, query string, args ...any) driver.Row {
	return m.queryRowFn(ctx, query, args...)
}
func (m *mockConn) Exec(ctx context.Context, query string, args ...any) error {
	return m.execFn(ctx, query, args...)
}
func (m *mockConn) Ping(ctx context.Context) error { return m.pingFn(ctx) }
func (m *mockConn) PrepareBatch(ctx context.Context, query string, opts ...driver.PrepareBatchOption) (driver.Batch, error) {
	return m.prepareBatchFn(ctx, query, opts...)
}

type mockRows struct {
	err error
}

func (m mockRows) Next() bool                   { return false }
func (m mockRows) Scan(...any) error            { return m.err }
func (m mockRows) ScanStruct(any) error         { return m.err }
func (m mockRows) ColumnTypes() []driver.ColumnType { return nil }
func (m mockRows) Totals(...any) error          { return nil }
func (m mockRows) Columns() []string            { return nil }
func (m mockRows) Close() error                 { return nil }
func (m mockRows) Err() error                   { return m.err }

type mockRow struct {
	err error
}

func (m mockRow) Err() error          { return m.err }
func (m mockRow) Scan(...any) error   { return m.err }
func (m mockRow) ScanStruct(any) error { return m.err }

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func newTestRetryClient(cfg *RetryConfig, dialFn func() driver.Conn) *RetryClient {
	cfg = cfg.withDefaults()
	rc := &RetryClient{
		client: New(dialFn()),
		cfg:    cfg,
		dial:   func() (driver.Conn, error) { return dialFn(), nil },
	}
	if cfg.IsRetryable != nil {
		rc.isR = cfg.IsRetryable
	} else {
		rc.isR = defaultIsRetryable
	}
	return rc
}

func TestDefaultIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"io.EOF", io.EOF, true},
		{"io.ErrUnexpectedEOF", io.ErrUnexpectedEOF, true},
		{"net.ErrClosed", net.ErrClosed, true},
		{"net.OpError", &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}, true},
		{"net.Error with Timeout", &timeoutErr{}, true},
		{"connection refused string", fmt.Errorf("dial tcp: connection refused"), true},
		{"broken pipe string", fmt.Errorf("write: broken pipe"), true},
		{"connection reset string", fmt.Errorf("read: connection reset by peer"), true},
		{"closed network connection", fmt.Errorf("use of closed network connection"), true},
		{"clickhouse NETWORK_ERROR", &clickhouse.Exception{Code: 203, Message: "network error"}, true},
		{"clickhouse SOCKET_TIMEOUT", &clickhouse.Exception{Code: 210, Message: "timeout"}, true},
		{"clickhouse SESSION_NOT_FOUND", &clickhouse.Exception{Code: 242, Message: "session"}, true},
		{"clickhouse NO_AVAILABLE_CONNECTION", &clickhouse.Exception{Code: 341, Message: "no conn"}, true},
		{"clickhouse non-network error", &clickhouse.Exception{Code: 60, Message: "syntax error"}, false},
		{"generic error", errors.New("something else"), false},
		{"wrapped io.EOF", fmt.Errorf("wrapped: %w", io.EOF), true},
		{"wrapped net error", fmt.Errorf("failed: %w", &net.OpError{Op: "read", Net: "tcp", Err: errors.New("reset")}), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultIsRetryable(tt.err)
			if got != tt.want {
				t.Errorf("defaultIsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetryConfigDefaults(t *testing.T) {
	tests := []struct {
		name string
		cfg  *RetryConfig
		want *RetryConfig
	}{
		{
			name: "nil config",
			cfg:  nil,
			want: nil,
		},
		{
			name: "zero values get defaults",
			cfg:  &RetryConfig{},
			want: &RetryConfig{MaxRetries: 3, InitialDelay: 100 * time.Millisecond, MaxDelay: 5 * time.Second},
		},
		{
			name: "partial override",
			cfg:  &RetryConfig{MaxRetries: 5},
			want: &RetryConfig{MaxRetries: 5, InitialDelay: 100 * time.Millisecond, MaxDelay: 5 * time.Second},
		},
		{
			name: "full config preserved",
			cfg:  &RetryConfig{MaxRetries: 10, InitialDelay: 200 * time.Millisecond, MaxDelay: 10 * time.Second, RetryExec: true},
			want: &RetryConfig{MaxRetries: 10, InitialDelay: 200 * time.Millisecond, MaxDelay: 10 * time.Second, RetryExec: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.withDefaults()
			if tt.want == nil {
				if got != nil {
					t.Errorf("withDefaults() = %v, want nil", got)
				}
				return
			}
			if got.MaxRetries != tt.want.MaxRetries || got.InitialDelay != tt.want.InitialDelay || got.MaxDelay != tt.want.MaxDelay || got.RetryExec != tt.want.RetryExec {
				t.Errorf("withDefaults() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestBackoff(t *testing.T) {
	cfg := &RetryConfig{MaxRetries: 3, InitialDelay: 100 * time.Millisecond, MaxDelay: 5 * time.Second}
	rc := newTestRetryClient(cfg, func() driver.Conn { return &mockConn{} })

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
	}

	for _, tt := range tests {
		got := rc.backoff(tt.attempt)
		if got != tt.want {
			t.Errorf("backoff(%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestBackoffCapsAtMaxDelay(t *testing.T) {
	cfg := &RetryConfig{MaxRetries: 10, InitialDelay: 100 * time.Millisecond, MaxDelay: 500 * time.Millisecond}
	rc := newTestRetryClient(cfg, func() driver.Conn { return &mockConn{} })

	if got := rc.backoff(10); got != 500*time.Millisecond {
		t.Errorf("backoff(10) = %v, want %v", got, 500*time.Millisecond)
	}
}

func TestRetryClientQueryRetry(t *testing.T) {
	callCount := 0
	conn := &mockConn{
		queryFn: func(ctx context.Context, query string, args ...any) (driver.Rows, error) {
			callCount++
			if callCount < 3 {
				return nil, io.EOF
			}
			return mockRows{}, nil
		},
	}

	rc := newTestRetryClient(&RetryConfig{MaxRetries: 3, InitialDelay: 0, MaxDelay: time.Millisecond}, func() driver.Conn { return conn })

	rows, err := rc.Query(context.Background(), "SELECT 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows == nil {
		t.Fatal("expected rows, got nil")
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestRetryClientQueryNonRetryableError(t *testing.T) {
	expectedErr := errors.New("syntax error")
	callCount := 0
	conn := &mockConn{
		queryFn: func(ctx context.Context, query string, args ...any) (driver.Rows, error) {
			callCount++
			return nil, expectedErr
		},
	}

	rc := newTestRetryClient(&RetryConfig{MaxRetries: 3, InitialDelay: 0, MaxDelay: time.Millisecond}, func() driver.Conn { return conn })

	_, err := rc.Query(context.Background(), "SELECT 1")
	if err != expectedErr {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (no retry), got %d", callCount)
	}
}

func TestRetryClientQueryRowRetry(t *testing.T) {
	callCount := 0
	conn := &mockConn{
		queryRowFn: func(ctx context.Context, query string, args ...any) driver.Row {
			callCount++
			if callCount < 3 {
				return mockRow{err: io.EOF}
			}
			return mockRow{}
		},
	}

	rc := newTestRetryClient(&RetryConfig{MaxRetries: 3, InitialDelay: 0, MaxDelay: time.Millisecond}, func() driver.Conn { return conn })

	row := rc.QueryRow(context.Background(), "SELECT 1")
	if err := row.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestRetryClientPingRetry(t *testing.T) {
	callCount := 0
	conn := &mockConn{
		pingFn: func(ctx context.Context) error {
			callCount++
			if callCount < 2 {
				return &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}
			}
			return nil
		},
	}

	rc := newTestRetryClient(&RetryConfig{MaxRetries: 3, InitialDelay: 0, MaxDelay: time.Millisecond}, func() driver.Conn { return conn })

	if err := rc.Ping(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestRetryClientExecNoRetryByDefault(t *testing.T) {
	expectedErr := io.EOF
	callCount := 0
	conn := &mockConn{
		execFn: func(ctx context.Context, query string, args ...any) error {
			callCount++
			return expectedErr
		},
	}

	rc := newTestRetryClient(&RetryConfig{MaxRetries: 3, InitialDelay: 0, MaxDelay: time.Millisecond}, func() driver.Conn { return conn })

	err := rc.Exec(context.Background(), "INSERT INTO t VALUES (1)")
	if err != expectedErr {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (no retry for Exec by default), got %d", callCount)
	}
}

func TestRetryClientExecRetryWhenEnabled(t *testing.T) {
	callCount := 0
	conn := &mockConn{
		execFn: func(ctx context.Context, query string, args ...any) error {
			callCount++
			if callCount < 3 {
				return io.EOF
			}
			return nil
		},
	}

	rc := newTestRetryClient(
		&RetryConfig{MaxRetries: 3, InitialDelay: 0, MaxDelay: time.Millisecond, RetryExec: true},
		func() driver.Conn { return conn },
	)

	if err := rc.Exec(context.Background(), "INSERT INTO t VALUES (1)"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestRetryClientMaxRetriesExhausted(t *testing.T) {
	callCount := 0
	conn := &mockConn{
		queryFn: func(ctx context.Context, query string, args ...any) (driver.Rows, error) {
			callCount++
			return nil, io.EOF
		},
	}

	rc := newTestRetryClient(&RetryConfig{MaxRetries: 2, InitialDelay: 0, MaxDelay: time.Millisecond}, func() driver.Conn { return conn })

	_, err := rc.Query(context.Background(), "SELECT 1")
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls (initial + 2 retries), got %d", callCount)
	}
}

func TestRetryClientContextCanceled(t *testing.T) {
	callCount := 0
	conn := &mockConn{
		queryFn: func(ctx context.Context, query string, args ...any) (driver.Rows, error) {
			callCount++
			return nil, io.EOF
		},
	}

	rc := newTestRetryClient(&RetryConfig{MaxRetries: 10, InitialDelay: 10 * time.Second, MaxDelay: time.Minute}, func() driver.Conn { return conn })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := rc.Query(ctx, "SELECT 1")
	if err == nil {
		t.Fatal("expected context canceled error")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestRetryClientCustomIsRetryable(t *testing.T) {
	callCount := 0
	conn := &mockConn{
		queryFn: func(ctx context.Context, query string, args ...any) (driver.Rows, error) {
			callCount++
			return nil, fmt.Errorf("custom error")
		},
	}

	cfg := RetryConfig{
		MaxRetries:   3,
		InitialDelay: 0,
		MaxDelay:     time.Millisecond,
		IsRetryable: func(err error) bool {
			return err.Error() == "custom error"
		},
	}

	rc := newTestRetryClient(&cfg, func() driver.Conn { return conn })

	_, err := rc.Query(context.Background(), "SELECT 1")
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if callCount != 4 {
		t.Errorf("expected 4 calls (initial + 3 retries), got %d", callCount)
	}
}

func TestRetryClientConcurrency(t *testing.T) {
	var mu sync.Mutex
	callCount := 0
	conn := &mockConn{
		queryFn: func(ctx context.Context, query string, args ...any) (driver.Rows, error) {
			mu.Lock()
			callCount++
			mu.Unlock()
			return mockRows{}, nil
		},
	}

	rc := newTestRetryClient(&RetryConfig{MaxRetries: 3, InitialDelay: 0, MaxDelay: time.Millisecond}, func() driver.Conn { return conn })

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := rc.Query(context.Background(), "SELECT 1")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if callCount != 100 {
		t.Errorf("expected 100 successful calls, got %d", callCount)
	}
}

func TestRetryClientImplementsExecutor(t *testing.T) {
	var _ interface {
		Query(ctx context.Context, query string, args ...any) (driver.Rows, error)
		QueryRow(ctx context.Context, query string, args ...any) driver.Row
		Exec(ctx context.Context, query string, args ...any) error
	} = (*RetryClient)(nil)
}

func TestErrorRow(t *testing.T) {
	expectedErr := errors.New("test error")
	row := errorRow{err: expectedErr}

	if err := row.Err(); err != expectedErr {
		t.Errorf("Err() = %v, want %v", err, expectedErr)
	}
	if err := row.Scan(nil); err != expectedErr {
		t.Errorf("Scan() = %v, want %v", err, expectedErr)
	}
	if err := row.ScanStruct(nil); err != expectedErr {
		t.Errorf("ScanStruct() = %v, want %v", err, expectedErr)
	}
}
