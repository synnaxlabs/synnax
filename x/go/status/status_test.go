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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
)

type CustomDetails struct {
	Code    int
	Context string
}

var _ = Describe("Status", func() {
	Describe("String", func() {
		Context("Basic status formatting", func() {
			It("Should format a basic info status", func() {
				s := status.Status[any]{
					Variant: status.InfoVariant,
					Name:    "Information",
					Message: "This is an info message",
				}
				Expect(s.String()).To(Equal("[ℹ info] Information: This is an info message"))
			})

			It("Should format a success status with key", func() {
				s := status.Status[any]{
					Variant: status.SuccessVariant,
					Key:     "op.success",
					Name:    "Operation Complete",
					Message: "Successfully completed operation",
				}
				Expect(s.String()).To(Equal("[✓ success] Operation Complete (op.success): Successfully completed operation"))
			})

			It("Should format an error status with description", func() {
				s := status.Status[any]{
					Variant:     status.ErrorVariant,
					Name:        "Database Error",
					Message:     "Failed to connect",
					Description: "Connection timeout after 30 seconds. Check network settings.",
				}
				Expect(s.String()).To(Equal("[✗ error] Database Error: Failed to connect\n  Connection timeout after 30 seconds. Check network settings."))
			})

			It("Should format a warning status with timestamp", func() {
				s := status.Status[any]{
					Variant: status.WarningVariant,
					Name:    "Memory Warning",
					Message: "High memory usage detected",
					Time:    telem.TimeStamp(1234567890000000000),
				}
				Expect(s.String()).To(Equal("[⚠ warning] Memory Warning: High memory usage detected\n  @ 2009-02-13T23:31:30Z"))
			})

			It("Should format a disabled status minimal", func() {
				s := status.Status[any]{
					Variant: status.DisabledVariant,
				}
				Expect(s.String()).To(Equal("[⊘ disabled]"))
			})

			It("Should format a loading status with all fields", func() {
				s := status.Status[any]{
					Variant:     status.LoadingVariant,
					Key:         "task.load",
					Name:        "Loading Data",
					Message:     "Processing files",
					Description: "Loading 500 files from disk",
					Time:        telem.TimeStamp(1609459200000000000),
				}
				Expect(s.String()).To(Equal("[◌ loading] Loading Data (task.load): Processing files\n  Loading 500 files from disk\n  @ 2021-01-01T00:00:00Z"))
			})
		})

		Context("Status with custom details", func() {
			It("Should format status with struct details", func() {
				s := status.Status[CustomDetails]{
					Variant: status.ErrorVariant,
					Name:    "API Error",
					Message: "Request failed",
					Details: CustomDetails{
						Code:    404,
						Context: "Resource not found",
					},
				}
				Expect(s.String()).To(Equal("[✗ error] API Error: Request failed\n  Details: {404 Resource not found}"))
			})

			It("Should format status with int details", func() {
				s := status.Status[int]{
					Variant: status.InfoVariant,
					Name:    "Count",
					Message: "Total items",
					Details: 42,
				}
				Expect(s.String()).To(Equal("[ℹ info] Count: Total items\n  Details: 42"))
			})

			It("Should omit zero int details", func() {
				s := status.Status[int]{
					Variant: status.InfoVariant,
					Name:    "Count",
					Message: "No items",
					Details: 0,
				}
				Expect(s.String()).To(Equal("[ℹ info] Count: No items"))
			})

			It("Should format status with string details", func() {
				s := status.Status[string]{
					Variant: status.WarningVariant,
					Name:    "Configuration",
					Message: "Using default",
					Details: "production",
				}
				Expect(s.String()).To(Equal("[⚠ warning] Configuration: Using default\n  Details: production"))
			})

			It("Should omit empty string details", func() {
				s := status.Status[string]{
					Variant: status.InfoVariant,
					Name:    "Status",
					Message: "Ready",
					Details: "",
				}
				Expect(s.String()).To(Equal("[ℹ info] Status: Ready"))
			})

			It("Should format status with map details", func() {
				s := status.Status[map[string]interface{}]{
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
				}
				Expect(s.String()).To(Equal("[✗ error] Critical Database Failure (sys.critical.db): Unable to write to primary database\n  The primary database cluster is unreachable. Failover to secondary cluster initiated. Data loss may have occurred for transactions between 14:30:00 and 14:30:45.\n  @ 2021-01-01T00:00:00Z\n  Details: map[affected_tables:[users sessions] lost_records:127]"))
			})

			It("Should omit nil pointer details", func() {
				type PtrDetails struct {
					Value *string
				}

				s := status.Status[*PtrDetails]{
					Variant: status.InfoVariant,
					Name:    "Nil Test",
					Message: "Testing nil details",
					Details: nil,
				}
				Expect(s.String()).To(Equal("[ℹ info] Nil Test: Testing nil details"))
			})
		})

		Context("Special cases", func() {
			It("Should handle unknown variant", func() {
				s := status.Status[any]{
					Variant: "custom",
					Name:    "Custom Status",
					Message: "Unknown variant type",
				}
				Expect(s.String()).To(Equal("[• custom] Custom Status: Unknown variant type"))
			})

			It("Should omit key when same as name", func() {
				s := status.Status[any]{
					Variant: status.InfoVariant,
					Key:     "SystemStatus",
					Name:    "SystemStatus",
					Message: "All systems operational",
				}
				Expect(s.String()).To(Equal("[ℹ info] SystemStatus: All systems operational"))
			})

			It("Should format status with only variant", func() {
				s := status.Status[any]{
					Variant: status.SuccessVariant,
				}
				Expect(s.String()).To(Equal("[✓ success]"))
			})
		})
	})
})
