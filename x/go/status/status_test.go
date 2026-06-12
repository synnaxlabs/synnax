// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
)

type CustomDetails struct {
	Context string
	Code    int
}

func (d CustomDetails) String() string {
	return fmt.Sprintf("{%d %s}", d.Code, d.Context)
}

var _ = Describe("Status", func() {
	Describe("String", func() {
		Context("Basic status formatting", func() {
			It("Should format a basic info status", func() {
				s := status.Status[any]{Variant: status.VariantInfo, Message: "This is an info message"}
				Expect(s.String()).To(Equal("[ℹ info]: This is an info message"))
			})

			It("Should format an error status with description", func() {
				s := status.Status[any]{
					Variant:     status.VariantError,
					Message:     "Failed to connect",
					Description: "Connection timeout after 30 seconds. Check network settings.",
				}
				Expect(s.String()).To(Equal("[✗ error]: Failed to connect\n  Connection timeout after 30 seconds. Check network settings."))
			})

			It("Should format a warning status with timestamp", func() {
				s := status.Status[any]{
					Variant: status.VariantWarning,
					Message: "High memory usage detected",
					Time:    telem.TimeStamp(1234567890000000000),
				}
				Expect(s.String()).To(Equal("[⚠ warning]: High memory usage detected\n  @ 2009-02-13T23:31:30Z"))
			})

			It("Should format a disabled status minimal", func() {
				s := status.Status[any]{Variant: status.VariantDisabled}
				Expect(s.String()).To(Equal("[⊘ disabled]"))
			})

			It("Should handle unknown variant", func() {
				s := status.Status[any]{Variant: "custom", Message: "Unknown variant type"}
				Expect(s.String()).To(Equal("[• custom]: Unknown variant type"))
			})
		})

		Context("Status with custom details", func() {
			It("Should format status with struct details", func() {
				s := status.Status[CustomDetails]{
					Variant: status.VariantError,
					Message: "Request failed",
					Details: CustomDetails{Code: 404, Context: "Resource not found"},
				}
				Expect(s.String()).To(Equal("[✗ error]: Request failed\n  Details: {404 Resource not found}"))
			})

			It("Should format status with int details", func() {
				s := status.Status[int]{Variant: status.VariantInfo, Message: "Total items", Details: 42}
				Expect(s.String()).To(Equal("[ℹ info]: Total items\n  Details: 42"))
			})

			It("Should omit zero int details", func() {
				s := status.Status[int]{Variant: status.VariantInfo, Message: "No items", Details: 0}
				Expect(s.String()).To(Equal("[ℹ info]: No items"))
			})

			It("Should omit empty string details", func() {
				s := status.Status[string]{Variant: status.VariantInfo, Message: "Ready", Details: ""}
				Expect(s.String()).To(Equal("[ℹ info]: Ready"))
			})
		})
	})
})
