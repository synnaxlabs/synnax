// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errors_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
)

var _ = Describe("Catcher", func() {
	Describe("Catcher", func() {
		Context("No error encountered", func() {
			var (
				counter int
				catcher *errors.Catcher
			)
			BeforeEach(func() {
				counter = 1
				catcher = errors.NewCatcher()
				for range 4 {
					catcher.Exec(func() error {
						counter++
						return nil
					})
				}
			})
			It("Should continue to execute functions", func() {

				Expect(counter).To(Equal(5))
			})
			It("Should contain a nil error", func() {
				Expect(catcher.Error()).To(BeNil())
			})
		})
		Context("Errors encountered", func() {
			var (
				counter int
				catcher *errors.Catcher
			)
			BeforeEach(func() {
				counter = 1
				catcher = errors.NewCatcher()
				for i := range 4 {
					catcher.Exec(func() error {
						if i == 2 {
							return errors.Newf("encountered unknown error")
						}
						counter++
						return nil
					})
				}
			})
			It("Should stop execution", func() {
				Expect(counter).To(Equal(3))
			})
			It("Should contain a non-nil error", func() {
				Expect(catcher.Error()).ToNot(BeNil())
			})
			Describe("Reset", func() {
				It("Should reset the catcher", func() {
					catcher.Reset()
					Expect(catcher.Error()).To(BeNil())
				})
			})

		})
		Context("Aggregation", func() {
			var catcher = errors.NewCatcher(errors.WithAggregation())
			It("Should aggregate the errors", func() {
				counter := 1
				for range 4 {
					catcher.Exec(func() error {
						counter++
						return errors.Newf("error encountered")
					})
				}
				Expect(counter).To(Equal(5))
				Expect(catcher.Errors()).To(HaveLen(4))
			})
		})
	})

})
