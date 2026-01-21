package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	chx "github.com/ManiacMaxo/chx"
	chmodule "github.com/testcontainers/testcontainers-go/modules/clickhouse"
)

// testClient is the shared client for all integration tests.
var testClient *chx.Client

// testCtx is a background context for tests.
var testCtx = context.Background()

// TestMain sets up the ClickHouse container and client for all tests.
func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Start ClickHouse container with explicit credentials
	const testUser = "testuser"
	const testPass = "testpass"
	const testDB = "default"

	container, err := chmodule.Run(ctx,
		"clickhouse/clickhouse-server:24.8",
		chmodule.WithUsername(testUser),
		chmodule.WithPassword(testPass),
		chmodule.WithDatabase(testDB),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start clickhouse container: %v\n", err)
		os.Exit(1)
	}

	// Get connection details
	host, err := container.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get container host: %v\n", err)
		os.Exit(1)
	}

	port, err := container.MappedPort(ctx, "9000/tcp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get container port: %v\n", err)
		os.Exit(1)
	}

	// Create client with matching credentials
	testClient, err = chx.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", host, port.Port())},
		Auth: clickhouse.Auth{
			Database: testDB,
			Username: testUser,
			Password: testPass,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create clickhouse client: %v\n", err)
		os.Exit(1)
	}

	// Verify connection
	if err := testClient.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to ping clickhouse: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("ClickHouse container started successfully")

	// Run tests
	code := m.Run()

	// Cleanup
	testClient.Close()
	if err := container.Terminate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to terminate container: %v\n", err)
	}

	os.Exit(code)
}

// TestPing verifies the connection is working.
func TestPing(t *testing.T) {
	err := testClient.Ping(testCtx)
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}
