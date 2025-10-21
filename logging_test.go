package go11y_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/cirruscomms/go11y"
)

func TestLoggingContext(t *testing.T) {
	t.Setenv("ENV", "test")
	t.Setenv("LOG_LEVEL", "develop")

	buf := new(bytes.Buffer)

	cfg, err := go11y.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	ctx, o, err := go11y.Initialise(context.Background(), cfg, buf)
	if err != nil {
		t.Fatalf("failed to initialise observer: %v", err)
	}
	defer func() {
		o.Close()
	}()

	o.Error("Test Logging Context", errors.New("TestLoggingContext"), go11y.SeverityHighest, "fatal", 1)
	ctx, o = go11y.Extend(ctx, nil, "", go11y.FieldRequestID, uuid.New())
	o.Info("TestLoggingContext", nil, "info", 1)
	ctx = AddFieldsToLoggerInContext(t, ctx, go11y.FieldRequestMethod, "GET", go11y.FieldRequestPath, "/api/v1/test")
	_, o = go11y.Get(ctx)
	o.Info("TestLoggingContext", nil, "info", 2)

	// @TODO: read the buffer and check the output matches expected log format
	// and content
}

func AddFieldsToLoggerInContext(t *testing.T, ctx context.Context, args ...any) (modCtx context.Context) {
	// Add fields to the logger in the context
	c, o := go11y.Extend(ctx, args...)

	o.Info("AddFieldsToLoggerInContext", nil, "info", 1)

	return c
}

func TestDeduplication(t *testing.T) {
	testCases := []struct {
		name      string
		input     []any
		expected  []any
		dupedKeys []string
	}{
		{
			name: "no duplicates",
			input: []any{
				"key1", "value1",
				"key2", "value2",
			},
			expected: []any{
				"key1", "value1",
				"key2", "value2",
			},
			dupedKeys: []string{},
		},
		{
			name: "just identical duplicates",
			input: []any{
				"key1", "value1",
				"key1", "value1",
			},
			expected: []any{
				"key1", "value1",
			},
			dupedKeys: []string{
				"key1",
			},
		},
		{
			name: "just duplicate keys with different values",
			input: []any{
				"key1", "value1",
				"key1", "value2",
			},
			expected: []any{
				"key1", "value1",
			},
			dupedKeys: []string{
				"key1",
			},
		},
		{
			name: "unique arg-pairs plus duplicate keys with different values",
			input: []any{
				"key1", "value1",
				"key2", "value2",
				"key1", "value2",
			},
			expected: []any{
				"key1", "value1",
				"key2", "value2",
			},
			dupedKeys: []string{
				"key1",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := go11y.DeduplicateArgs(tc.input)
			if len(result) != len(tc.expected) {
				t.Fatalf("expected length %d, got %d", len(tc.expected), len(result))
			}

			for i := range result {
				if result[i] != tc.expected[i] {
					t.Errorf("at index %d, expected %v, got %v", i, tc.expected[i], result[i])
				}
			}
		})
	}
}
