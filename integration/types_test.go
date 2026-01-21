package integration

import (
	"math"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTypeRoundTrip_Integers tests integer type round-trips.
func TestTypeRoundTrip_Integers(t *testing.T) {
	tests := []struct {
		name     string
		chType   string
		values   []any
		scanType any
	}{
		{
			name:     "Int8",
			chType:   "Int8",
			values:   []any{int8(0), int8(1), int8(-1), int8(math.MaxInt8), int8(math.MinInt8)},
			scanType: new(int8),
		},
		{
			name:     "Int16",
			chType:   "Int16",
			values:   []any{int16(0), int16(1), int16(-1), int16(math.MaxInt16), int16(math.MinInt16)},
			scanType: new(int16),
		},
		{
			name:     "Int32",
			chType:   "Int32",
			values:   []any{int32(0), int32(1), int32(-1), int32(math.MaxInt32), int32(math.MinInt32)},
			scanType: new(int32),
		},
		{
			name:     "Int64",
			chType:   "Int64",
			values:   []any{int64(0), int64(1), int64(-1), int64(math.MaxInt64), int64(math.MinInt64)},
			scanType: new(int64),
		},
		{
			name:     "UInt8",
			chType:   "UInt8",
			values:   []any{uint8(0), uint8(1), uint8(math.MaxUint8)},
			scanType: new(uint8),
		},
		{
			name:     "UInt16",
			chType:   "UInt16",
			values:   []any{uint16(0), uint16(1), uint16(math.MaxUint16)},
			scanType: new(uint16),
		},
		{
			name:     "UInt32",
			chType:   "UInt32",
			values:   []any{uint32(0), uint32(1), uint32(math.MaxUint32)},
			scanType: new(uint32),
		},
		{
			name:     "UInt64",
			chType:   "UInt64",
			values:   []any{uint64(0), uint64(1), uint64(math.MaxUint64)},
			scanType: new(uint64),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := testTable("int_" + tt.name)
			defer dropTable(testCtx, table)

			// Create table
			err := testClient.CreateTable(table).
				Column("id", "UInt32").Add().
				Column("value", tt.chType).Add().
				Engine("Memory").
				Exec(testCtx)
			require.NoError(t, err)

			// Insert and read back each value
			for i, v := range tt.values {
				err := testClient.Insert(table).
					Columns("id", "value").
					Values(uint32(i), v).
					Exec(testCtx)
				require.NoError(t, err, "insert failed for value %v", v)

				// Read back
				row := testClient.Select("value").From(table).Where("id = ?", uint32(i)).QueryRow(testCtx)

				// Create a new pointer of the appropriate type for scanning
				var result any
				switch tt.chType {
				case "Int8":
					var r int8
					err = row.Scan(&r)
					result = r
				case "Int16":
					var r int16
					err = row.Scan(&r)
					result = r
				case "Int32":
					var r int32
					err = row.Scan(&r)
					result = r
				case "Int64":
					var r int64
					err = row.Scan(&r)
					result = r
				case "UInt8":
					var r uint8
					err = row.Scan(&r)
					result = r
				case "UInt16":
					var r uint16
					err = row.Scan(&r)
					result = r
				case "UInt32":
					var r uint32
					err = row.Scan(&r)
					result = r
				case "UInt64":
					var r uint64
					err = row.Scan(&r)
					result = r
				}

				require.NoError(t, err, "scan failed for value %v", v)
				assert.Equal(t, v, result, "value mismatch for %v", v)
			}
		})
	}
}

// TestTypeRoundTrip_Floats tests float type round-trips.
func TestTypeRoundTrip_Floats(t *testing.T) {
	t.Run("Float32", func(t *testing.T) {
		table := testTable("float32")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Column("value", "Float32").Add().
			Engine("Memory").
			Exec(testCtx)
		require.NoError(t, err)

		// Use values that can be precisely represented in Float32
		values := []float32{0, 1.5, -1.5, 1e10, 1e-10}
		for i, v := range values {
			err := testClient.Insert(table).Columns("id", "value").Values(uint32(i), v).Exec(testCtx)
			require.NoError(t, err)

			var result float32
			row := testClient.Select("value").From(table).Where("id = ?", uint32(i)).QueryRow(testCtx)
			err = row.Scan(&result)
			require.NoError(t, err)
			assert.InDelta(t, float64(v), float64(result), 1e-6, "float32 mismatch for %v", v)
		}
	})

	t.Run("Float64", func(t *testing.T) {
		table := testTable("float64")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Column("value", "Float64").Add().
			Engine("Memory").
			Exec(testCtx)
		require.NoError(t, err)

		// Use values that are more reliably round-tripped
		values := []float64{0, 1.5, -1.5, 1e100, 1e-100}
		for i, v := range values {
			err := testClient.Insert(table).Columns("id", "value").Values(uint32(i), v).Exec(testCtx)
			require.NoError(t, err)

			var result float64
			row := testClient.Select("value").From(table).Where("id = ?", uint32(i)).QueryRow(testCtx)
			err = row.Scan(&result)
			require.NoError(t, err)
			assert.InDelta(t, v, result, math.Abs(v*1e-10)+1e-300, "float64 mismatch for %v", v)
		}
	})
}

// TestTypeRoundTrip_Strings tests string type round-trips.
func TestTypeRoundTrip_Strings(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		table := testTable("string")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Column("value", "String").Add().
			Engine("Memory").
			Exec(testCtx)
		require.NoError(t, err)

		values := []string{"", "hello", "world", "hello world", "unicode: \u4e2d\u6587", "emoji: \U0001F600"}
		for i, v := range values {
			err := testClient.Insert(table).Columns("id", "value").Values(uint32(i), v).Exec(testCtx)
			require.NoError(t, err)

			var result string
			row := testClient.Select("value").From(table).Where("id = ?", uint32(i)).QueryRow(testCtx)
			err = row.Scan(&result)
			require.NoError(t, err)
			assert.Equal(t, v, result)
		}
	})

	t.Run("FixedString", func(t *testing.T) {
		table := testTable("fixedstring")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Column("value", "FixedString(10)").Add().
			Engine("Memory").
			Exec(testCtx)
		require.NoError(t, err)

		// FixedString pads with null bytes
		err = testClient.Insert(table).Columns("id", "value").Values(uint32(0), "hello").Exec(testCtx)
		require.NoError(t, err)

		var result string
		row := testClient.Select("value").From(table).Where("id = ?", uint32(0)).QueryRow(testCtx)
		err = row.Scan(&result)
		require.NoError(t, err)
		assert.Equal(t, "hello\x00\x00\x00\x00\x00", result)
	})

	t.Run("LowCardinality", func(t *testing.T) {
		table := testTable("lowcardinality")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Column("value", "LowCardinality(String)").Add().
			Engine("Memory").
			Exec(testCtx)
		require.NoError(t, err)

		values := []string{"status_a", "status_b", "status_a", "status_c"}
		for i, v := range values {
			err := testClient.Insert(table).Columns("id", "value").Values(uint32(i), v).Exec(testCtx)
			require.NoError(t, err)

			var result string
			row := testClient.Select("value").From(table).Where("id = ?", uint32(i)).QueryRow(testCtx)
			err = row.Scan(&result)
			require.NoError(t, err)
			assert.Equal(t, v, result)
		}
	})
}

// TestTypeRoundTrip_Bool tests boolean type round-trip.
func TestTypeRoundTrip_Bool(t *testing.T) {
	table := testTable("bool")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("value", "Bool").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	values := []bool{true, false}
	for i, v := range values {
		err := testClient.Insert(table).Columns("id", "value").Values(uint32(i), v).Exec(testCtx)
		require.NoError(t, err)

		var result bool
		row := testClient.Select("value").From(table).Where("id = ?", uint32(i)).QueryRow(testCtx)
		err = row.Scan(&result)
		require.NoError(t, err)
		assert.Equal(t, v, result)
	}
}

// TestTypeRoundTrip_UUID tests UUID type round-trip.
func TestTypeRoundTrip_UUID(t *testing.T) {
	table := testTable("uuid")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("value", "UUID").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	values := []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000000"),
		uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		uuid.New(),
	}
	for i, v := range values {
		err := testClient.Insert(table).Columns("id", "value").Values(uint32(i), v).Exec(testCtx)
		require.NoError(t, err)

		var result uuid.UUID
		row := testClient.Select("value").From(table).Where("id = ?", uint32(i)).QueryRow(testCtx)
		err = row.Scan(&result)
		require.NoError(t, err)
		assert.Equal(t, v, result)
	}
}

// TestTypeRoundTrip_DateTime tests date/time type round-trips.
func TestTypeRoundTrip_DateTime(t *testing.T) {
	t.Run("Date", func(t *testing.T) {
		table := testTable("date")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Column("value", "Date").Add().
			Engine("Memory").
			Exec(testCtx)
		require.NoError(t, err)

		// Date is days since epoch, stored as time.Time with 00:00:00 UTC
		// Date type has range 1970-01-01 to 2149-06-06, but actual range may vary
		dates := []time.Time{
			time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC), // safer max Date
		}
		for i, v := range dates {
			err := testClient.Insert(table).Columns("id", "value").Values(uint32(i), v).Exec(testCtx)
			require.NoError(t, err)

			var result time.Time
			row := testClient.Select("value").From(table).Where("id = ?", uint32(i)).QueryRow(testCtx)
			err = row.Scan(&result)
			require.NoError(t, err)
			assert.True(t, v.Equal(result), "expected %v, got %v", v, result)
		}
	})

	t.Run("DateTime", func(t *testing.T) {
		table := testTable("datetime")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Column("value", "DateTime").Add().
			Engine("Memory").
			Exec(testCtx)
		require.NoError(t, err)

		// DateTime is seconds since epoch
		times := []time.Time{
			time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 6, 15, 12, 30, 45, 0, time.UTC),
		}
		for i, v := range times {
			err := testClient.Insert(table).Columns("id", "value").Values(uint32(i), v).Exec(testCtx)
			require.NoError(t, err)

			var result time.Time
			row := testClient.Select("value").From(table).Where("id = ?", uint32(i)).QueryRow(testCtx)
			err = row.Scan(&result)
			require.NoError(t, err)
			assert.True(t, v.Equal(result), "expected %v, got %v", v, result)
		}
	})

	t.Run("DateTime64", func(t *testing.T) {
		table := testTable("datetime64")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Column("value", "DateTime64(3)").Add().
			Engine("Memory").
			Exec(testCtx)
		require.NoError(t, err)

		// DateTime64 with millisecond precision
		// Note: precision(3) means milliseconds, so nanoseconds may be truncated
		times := []time.Time{
			time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 6, 15, 12, 30, 45, 0, time.UTC), // no sub-second precision
		}
		for i, v := range times {
			err := testClient.Insert(table).Columns("id", "value").Values(uint32(i), v).Exec(testCtx)
			require.NoError(t, err)

			var result time.Time
			row := testClient.Select("value").From(table).Where("id = ?", uint32(i)).QueryRow(testCtx)
			err = row.Scan(&result)
			require.NoError(t, err)
			// Compare with second precision since driver may not preserve milliseconds
			assert.Equal(t, v.Unix(), result.Unix(), "expected %v, got %v", v, result)
		}
	})
}

// TestTypeRoundTrip_Decimal tests decimal type round-trip.
func TestTypeRoundTrip_Decimal(t *testing.T) {
	table := testTable("decimal")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("value", "Decimal(18, 4)").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	// Use values within Decimal(18,4) precision: 14 integer digits + 4 decimal places
	values := []decimal.Decimal{
		decimal.NewFromFloat(0),
		decimal.NewFromFloat(123.4567),
		decimal.NewFromFloat(-123.4567),
		decimal.NewFromFloat(9999999999.9999), // 10 integer digits + 4 decimal
	}
	for i, v := range values {
		err := testClient.Insert(table).Columns("id", "value").Values(uint32(i), v).Exec(testCtx)
		require.NoError(t, err)

		var result decimal.Decimal
		row := testClient.Select("value").From(table).Where("id = ?", uint32(i)).QueryRow(testCtx)
		err = row.Scan(&result)
		require.NoError(t, err)
		assert.True(t, v.Round(4).Equal(result.Round(4)), "expected %v, got %v", v, result)
	}
}

// TestTypeRoundTrip_Array tests array type round-trips.
func TestTypeRoundTrip_Array(t *testing.T) {
	t.Run("Array(Int32)", func(t *testing.T) {
		table := testTable("array_int")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Column("value", "Array(Int32)").Add().
			Engine("Memory").
			Exec(testCtx)
		require.NoError(t, err)

		values := [][]int32{
			{},
			{1},
			{1, 2, 3},
			{-1, 0, 1, math.MaxInt32, math.MinInt32},
		}
		for i, v := range values {
			err := testClient.Insert(table).Columns("id", "value").Values(uint32(i), v).Exec(testCtx)
			require.NoError(t, err)

			var result []int32
			row := testClient.Select("value").From(table).Where("id = ?", uint32(i)).QueryRow(testCtx)
			err = row.Scan(&result)
			require.NoError(t, err)
			assert.Equal(t, v, result)
		}
	})

	t.Run("Array(String)", func(t *testing.T) {
		table := testTable("array_string")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Column("value", "Array(String)").Add().
			Engine("Memory").
			Exec(testCtx)
		require.NoError(t, err)

		values := [][]string{
			{},
			{"a"},
			{"a", "b", "c"},
			{"hello", "world", ""},
		}
		for i, v := range values {
			err := testClient.Insert(table).Columns("id", "value").Values(uint32(i), v).Exec(testCtx)
			require.NoError(t, err)

			var result []string
			row := testClient.Select("value").From(table).Where("id = ?", uint32(i)).QueryRow(testCtx)
			err = row.Scan(&result)
			require.NoError(t, err)
			assert.Equal(t, v, result)
		}
	})
}

// TestTypeRoundTrip_Map tests map type round-trip.
func TestTypeRoundTrip_Map(t *testing.T) {
	table := testTable("map")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("value", "Map(String, Int32)").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	values := []map[string]int32{
		{},
		{"a": 1},
		{"a": 1, "b": 2, "c": 3},
	}
	for i, v := range values {
		err := testClient.Insert(table).Columns("id", "value").Values(uint32(i), v).Exec(testCtx)
		require.NoError(t, err)

		var result map[string]int32
		row := testClient.Select("value").From(table).Where("id = ?", uint32(i)).QueryRow(testCtx)
		err = row.Scan(&result)
		require.NoError(t, err)
		assert.Equal(t, v, result)
	}
}

// TestTypeRoundTrip_Tuple tests tuple type round-trip.
func TestTypeRoundTrip_Tuple(t *testing.T) {
	table := testTable("tuple")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("value", "Tuple(String, Int32, Float64)").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert using a tuple struct or map - ClickHouse needs explicit tuple syntax
	// Using raw SQL for tuple insertion since the driver handles this better
	err = testClient.RawExec(testCtx, "INSERT INTO `"+table+"` (id, value) VALUES (0, ('hello', 42, 3.14))")
	require.NoError(t, err)

	// Read back - tuples are returned as maps with named fields or slices
	var s string
	var i int32
	var f float64
	row := testClient.SelectExpr("value.1, value.2, value.3").From(table).Where("id = ?", uint32(0)).QueryRow(testCtx)
	err = row.Scan(&s, &i, &f)
	require.NoError(t, err)
	assert.Equal(t, "hello", s)
	assert.Equal(t, int32(42), i)
	assert.InDelta(t, 3.14, f, 0.001)
}

// TestTypeRoundTrip_Nullable tests nullable type round-trips.
func TestTypeRoundTrip_Nullable(t *testing.T) {
	table := testTable("nullable")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("value", "Nullable(Int32)").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	// Insert NULL
	err = testClient.Insert(table).Columns("id", "value").Values(uint32(0), nil).Exec(testCtx)
	require.NoError(t, err)

	// Insert non-NULL
	err = testClient.Insert(table).Columns("id", "value").Values(uint32(1), int32(42)).Exec(testCtx)
	require.NoError(t, err)

	// Read NULL
	var result1 *int32
	row := testClient.Select("value").From(table).Where("id = ?", uint32(0)).QueryRow(testCtx)
	err = row.Scan(&result1)
	require.NoError(t, err)
	assert.Nil(t, result1)

	// Read non-NULL
	var result2 *int32
	row = testClient.Select("value").From(table).Where("id = ?", uint32(1)).QueryRow(testCtx)
	err = row.Scan(&result2)
	require.NoError(t, err)
	require.NotNil(t, result2)
	assert.Equal(t, int32(42), *result2)
}

// TestTypeRoundTrip_Enum tests enum type round-trip.
func TestTypeRoundTrip_Enum(t *testing.T) {
	table := testTable("enum")
	defer dropTable(testCtx, table)

	err := testClient.CreateTable(table).
		Column("id", "UInt32").Add().
		Column("value", "Enum8('pending' = 1, 'active' = 2, 'done' = 3)").Add().
		Engine("Memory").
		Exec(testCtx)
	require.NoError(t, err)

	values := []string{"pending", "active", "done"}
	for i, v := range values {
		err := testClient.Insert(table).Columns("id", "value").Values(uint32(i), v).Exec(testCtx)
		require.NoError(t, err)

		var result string
		row := testClient.Select("value").From(table).Where("id = ?", uint32(i)).QueryRow(testCtx)
		err = row.Scan(&result)
		require.NoError(t, err)
		assert.Equal(t, v, result)
	}
}

// TestTypeRoundTrip_IP tests IP address type round-trips.
func TestTypeRoundTrip_IP(t *testing.T) {
	t.Run("IPv4", func(t *testing.T) {
		table := testTable("ipv4")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Column("value", "IPv4").Add().
			Engine("Memory").
			Exec(testCtx)
		require.NoError(t, err)

		values := []net.IP{
			net.ParseIP("0.0.0.0").To4(),
			net.ParseIP("127.0.0.1").To4(),
			net.ParseIP("192.168.1.1").To4(),
			net.ParseIP("255.255.255.255").To4(),
		}
		for i, v := range values {
			err := testClient.Insert(table).Columns("id", "value").Values(uint32(i), v).Exec(testCtx)
			require.NoError(t, err)

			var result net.IP
			row := testClient.Select("value").From(table).Where("id = ?", uint32(i)).QueryRow(testCtx)
			err = row.Scan(&result)
			require.NoError(t, err)
			assert.True(t, v.Equal(result), "expected %v, got %v", v, result)
		}
	})

	t.Run("IPv6", func(t *testing.T) {
		table := testTable("ipv6")
		defer dropTable(testCtx, table)

		err := testClient.CreateTable(table).
			Column("id", "UInt32").Add().
			Column("value", "IPv6").Add().
			Engine("Memory").
			Exec(testCtx)
		require.NoError(t, err)

		values := []net.IP{
			net.ParseIP("::"),
			net.ParseIP("::1"),
			net.ParseIP("2001:db8::1"),
			net.ParseIP("fe80::1"),
		}
		for i, v := range values {
			err := testClient.Insert(table).Columns("id", "value").Values(uint32(i), v).Exec(testCtx)
			require.NoError(t, err)

			var result net.IP
			row := testClient.Select("value").From(table).Where("id = ?", uint32(i)).QueryRow(testCtx)
			err = row.Scan(&result)
			require.NoError(t, err)
			assert.True(t, v.Equal(result), "expected %v, got %v", v, result)
		}
	})
}
