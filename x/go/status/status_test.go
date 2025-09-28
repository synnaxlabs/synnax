// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"testing"
)

type CustomDetails struct {
	Code    int
	Context string
}

func TestStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   interface{}
		expected string
	}{
		{
			name: "basic info status",
			status: status.Status[any]{
				Variant: status.InfoVariant,
				Name:    "Information",
				Message: "This is an info message",
			},
			expected: "[ℹ info] Information: This is an info message",
		},
		{
			name: "success status with key",
			status: status.Status[any]{
				Variant: status.SuccessVariant,
				Key:     "op.success",
				Name:    "Operation Complete",
				Message: "Successfully completed operation",
			},
			expected: "[✓ success] Operation Complete (op.success): Successfully completed operation",
		},
		{
			name: "error status with description",
			status: status.Status[any]{
				Variant:     status.ErrorVariant,
				Name:        "Database Error",
				Message:     "Failed to connect",
				Description: "Connection timeout after 30 seconds. Check network settings.",
			},
			expected: "[✗ error] Database Error: Failed to connect\n  Connection timeout after 30 seconds. Check network settings.",
		},
		{
			name: "warning status with timestamp",
			status: status.Status[any]{
				Variant: status.WarningVariant,
				Name:    "Memory Warning",
				Message: "High memory usage detected",
				Time:    telem.TimeStamp(1234567890000000000),
			},
			expected: "[⚠ warning] Memory Warning: High memory usage detected\n  @ 2009-02-13T23:31:30Z",
		},
		{
			name: "disabled status minimal",
			status: status.Status[any]{
				Variant: status.DisabledVariant,
			},
			expected: "[⊘ disabled]",
		},
		{
			name: "loading status with all fields",
			status: status.Status[any]{
				Variant:     status.LoadingVariant,
				Key:         "task.load",
				Name:        "Loading Data",
				Message:     "Processing files",
				Description: "Loading 500 files from disk",
				Time:        telem.TimeStamp(1609459200000000000),
			},
			expected: "[◌ loading] Loading Data (task.load): Processing files\n  Loading 500 files from disk\n  @ 2021-01-01T00:00:00Z",
		},
		{
			name: "status with custom details",
			status: status.Status[CustomDetails]{
				Variant: status.ErrorVariant,
				Name:    "API Error",
				Message: "Request failed",
				Details: CustomDetails{
					Code:    404,
					Context: "Resource not found",
				},
			},
			expected: "[✗ error] API Error: Request failed\n  Details: {404 Resource not found}",
		},
		{
			name: "status with int details",
			status: status.Status[int]{
				Variant: status.InfoVariant,
				Name:    "Count",
				Message: "Total items",
				Details: 42,
			},
			expected: "[ℹ info] Count: Total items\n  Details: 42",
		},
		{
			name: "status with zero int details",
			status: status.Status[int]{
				Variant: status.InfoVariant,
				Name:    "Count",
				Message: "No items",
				Details: 0,
			},
			expected: "[ℹ info] Count: No items",
		},
		{
			name: "status with string details",
			status: status.Status[string]{
				Variant: status.WarningVariant,
				Name:    "Configuration",
				Message: "Using default",
				Details: "production",
			},
			expected: "[⚠ warning] Configuration: Using default\n  Details: production",
		},
		{
			name: "status with empty string details",
			status: status.Status[string]{
				Variant: status.InfoVariant,
				Name:    "Status",
				Message: "Ready",
				Details: "",
			},
			expected: "[ℹ info] Status: Ready",
		},
		{
			name: "unknown variant",
			status: status.Status[any]{
				Variant: "custom",
				Name:    "Custom Status",
				Message: "Unknown variant type",
			},
			expected: "[• custom] Custom Status: Unknown variant type",
		},
		{
			name: "key same as name",
			status: status.Status[any]{
				Variant: status.InfoVariant,
				Key:     "SystemStatus",
				Name:    "SystemStatus",
				Message: "All systems operational",
			},
			expected: "[ℹ info] SystemStatus: All systems operational",
		},
		{
			name: "only variant",
			status: status.Status[any]{
				Variant: status.SuccessVariant,
			},
			expected: "[✓ success]",
		},
		{
			name: "complex status with everything",
			status: status.Status[map[string]interface{}]{
				Variant:     status.ErrorVariant,
				Key:         "sys.critical.db",
				Name:        "Critical Database Failure",
				Message:     "Unable to write to primary database",
				Description: "The primary database cluster is unreachable. Failover to secondary cluster initiated. Data loss may have occurred for transactions between 14:30:00 and 14:30:45.",
				Time:        telem.TimeStamp(1609459200000000000),
				Details: map[string]interface{}{
					"affected_tables": []string{"users", "sessions"},
					"lost_records":    127,
				},
			},
			expected: "[✗ error] Critical Database Failure (sys.critical.db): Unable to write to primary database\n  The primary database cluster is unreachable. Failover to secondary cluster initiated. Data loss may have occurred for transactions between 14:30:00 and 14:30:45.\n  @ 2021-01-01T00:00:00Z\n  Details: map[affected_tables:[users sessions] lost_records:127]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			switch s := tt.status.(type) {
			case status.Status[any]:
				result = s.String()
			case status.Status[CustomDetails]:
				result = s.String()
			case status.Status[int]:
				result = s.String()
			case status.Status[string]:
				result = s.String()
			case status.Status[map[string]interface{}]:
				result = s.String()
			default:
				t.Fatalf("unexpected status type: %T", s)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatus_String_NilDetails(t *testing.T) {
	// Test with pointer details that are nil
	type PtrDetails struct {
		Value *string
	}

	s := status.Status[*PtrDetails]{
		Variant: status.InfoVariant,
		Name:    "Nil Test",
		Message: "Testing nil details",
		Details: nil,
	}

	expected := "[ℹ info] Nil Test: Testing nil details"
	assert.Equal(t, expected, s.String())
}

func BenchmarkStatus_String(b *testing.B) {
	s := status.Status[map[string]interface{}]{
		Variant:     status.ErrorVariant,
		Key:         "bench.test",
		Name:        "Benchmark Status",
		Message:     "Performance test",
		Description: "This is a benchmark test for the String() method",
		Time:        telem.TimeStamp(1609459200000000000),
		Details: map[string]interface{}{
			"iterations": 1000000,
			"duration":   "5s",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.String()
	}
}
