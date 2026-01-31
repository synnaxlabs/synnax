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

var _ = Describe("Accumulator", func() {
	Describe("No error encountered", func() {
		It("Should execute function and without errors", func() {
			var (
				counter     int
				accumulator errors.Accumulator
			)
			const count = 3
			for range count {
				accumulator.Exec(func() error { counter++; return nil })
			}
			Expect(counter).To(Equal(count))
			Expect(accumulator.Error()).To(BeNil())
		})
	})
	Describe("Errors encountered", func() {
		It("Should accumulate errors and continue execution", func() {
			var (
				counter     int
				accumulator errors.Accumulator
				testErr     = errors.New("test error")
			)
			const count = 3
			for i := range count {
				accumulator.Exec(func() error {
					counter++
					if i == 2 {
						return testErr
					}
					return nil
				})
			}
			Expect(counter).To(Equal(count))
			Expect(accumulator.Error()).To(MatchError(testErr))
		})
		It("Should accumulate multiple errors", func() {
			errMap := map[int]error{
				0: errors.New("error 0"),
				1: errors.New("error 1"),
				2: errors.New("error 2"),
			}
			var accumulator errors.Accumulator
			for i := range len(errMap) {
				accumulator.Exec(func() error { return errMap[i] })
			}
			for i := range len(errMap) {
				Expect(accumulator.Error()).To(MatchError(errMap[i]))
			}
		})
	})
})
