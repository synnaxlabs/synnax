// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package middleware_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/middleware"
)

type (
	input          = string
	output         = string
	handler        = middleware.Handler[input, output]
	testMiddleware = middleware.Middleware[input, output]
	chain          = middleware.Chain[input, output]
	collector      = middleware.Collector[input, output]
	testFunc       = middleware.Func[input, output]
)

func newStepper(steps *[]string) func(string) testMiddleware {
	return func(name string) testMiddleware {
		return testFunc(func(in input, next handler) (output, error) {
			*steps = append(*steps, fmt.Sprintf("enter %s", name))
			out, err := next(in)
			*steps = append(*steps, fmt.Sprintf("exit %s", name))
			return out, err
		})
	}
}

var _ = Describe("Middleware", func() {
	Describe("Chain", func() {
		var (
			steps   []string
			stepper func(string) testMiddleware
		)
		BeforeEach(func() {
			steps = []string{}
			stepper = newStepper(&steps)
		})
		It("executes middleware in order and finalizer last", func() {
			ch := chain{stepper("A"), stepper("B")}
			finalizer := func(input) (output, error) {
				steps = append(steps, "finalizer")
				return "final output", nil
			}
			Expect(ch.Exec("hello", finalizer)).To(Equal("final output"))
			Expect(steps).To(Equal([]string{
				"enter A",
				"enter B",
				"finalizer",
				"exit B",
				"exit A",
			}))
		})
		It("can short-circuit and skip remaining middleware", func() {
			ch := chain{
				testFunc(func(string, handler) (string, error) {
					steps = append(steps, "short-circuit")
					return "shorted", nil
				}),
				stepper("should not run"),
			}
			Expect(ch.Exec("hi", func(input) (output, error) {
				Fail("finalizer should not be called")
				return "", nil
			})).To(Equal("shorted"))
			Expect(steps).To(Equal([]string{"short-circuit"}))
		})
		It("propagates errors from middleware", func() {
			ch := chain{
				stepper("X"),
				testFunc(func(input, handler) (output, error) {
					return "", errors.New("middleware error")
				}),
				stepper("Z"),
			}
			Expect(ch.Exec("abc", func(input) (output, error) {
				Fail("finalizer should not be called")
				return "", nil
			})).Error().To(MatchError("middleware error"))
			Expect(steps).To(Equal([]string{"enter X", "exit X"}))
		})
		It("works with an empty chain", func() {
			ch := chain{}
			Expect(ch.Exec("foo", func(input) (output, error) {
				return "no middleware", nil
			})).To(Equal("no middleware"))
		})
	})
	Describe("Collector", func() {
		It("uses and executes middleware in correct order", func() {
			c := collector{}
			steps := []string{}
			stepper := newStepper(&steps)
			c.Use(stepper("A"), stepper("B"))
			finalizer := func(input) (output, error) {
				steps = append(steps, "finalizer")
				return "collector output", nil
			}
			Expect(c.Exec("hello", finalizer)).To(Equal("collector output"))
			Expect(steps).To(Equal([]string{
				"enter A",
				"enter B",
				"finalizer",
				"exit B",
				"exit A",
			}))
		})
	})
	Describe("Func", func() {
		It("adapts a plain function to Middleware and executes correctly", func() {
			var called bool
			fn := testFunc(func(in input, next handler) (output, error) {
				called = true
				return next("wrapped:" + in)
			})
			Expect(fn.Exec("data", func(in input) (output, error) {
				return "handled:" + in, nil
			})).To(Equal("handled:wrapped:data"))
			Expect(called).To(BeTrue())
		})
	})
})
