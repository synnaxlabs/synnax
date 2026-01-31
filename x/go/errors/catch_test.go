// Copyright 2026 Synnax Labs, Inc.
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
					catcher.Exec(func() error { counter++; return nil })
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
			It("Should aggregate errors and continue execution", func() {
				counter := 1
				catcher := errors.NewCatcher()
				for i := range 4 {
					catcher.Exec(func() error {
						if i == 2 {
							return errors.Newf("encountered unknown error")
						}
						counter++
						return nil
					})
				}
				Expect(counter).To(Equal(4))
				Expect(catcher.Error()).ToNot(BeNil())
				Expect(catcher.Errors()).To(HaveLen(1))
			})
			It("Should aggregate multiple errors", func() {
				counter := 1
				catcher := errors.NewCatcher()
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
