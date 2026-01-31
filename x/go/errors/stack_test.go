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
	stderrors "errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
)

var _ = Describe("Stack", func() {
	Describe("GetStackTrace", func() {
		It("Should return empty stack trace for nil error", func() {
			stack := errors.GetStackTrace(nil)
			Expect(stack.String()).To(BeEmpty())
		})

		It("Should return empty stack trace for error without stack", func() {
			err := stderrors.New("test")
			stack := errors.GetStackTrace(err)
			Expect(stack.String()).To(BeEmpty())
		})

		It("Should return stack trace for error with stack", func() {
			err := errors.WithStack(errors.New("test error"))
			stack := errors.GetStackTrace(err)
			Expect(stack.Frames).To(HaveLen(4))
			Expect(stack.String()).To(Not(BeEmpty()))
			Expect(stack.String()).To(ContainSubstring(".go"))
		})
	})

	Describe("WithStack", func() {
		It("Should add stack trace to error", func() {
			baseErr := errors.New("base error")
			err := errors.WithStack(baseErr)
			Expect(errors.Is(err, baseErr)).To(BeTrue())
			stack := errors.GetStackTrace(err)
			Expect(stack.Frames).To(HaveLen(4))
			Expect(stack.String()).To(Not(BeEmpty()))
			Expect(stack.String()).To(ContainSubstring(".go"))
			Expect(stack.String()).To(ContainSubstring("runNode"))
		})

		It("Should preserve error message", func() {
			baseErr := errors.New("base error")
			Expect(errors.WithStack(baseErr).Error()).To(Equal("base error"))
		})
	})

	Describe("WithStackDepth", func() {
		It("Should add stack trace starting from specified depth", func() {
			baseErr := errors.New("base error")
			err := errors.WithStackDepth(baseErr, 1)
			Expect(errors.Is(err, baseErr)).To(BeTrue())
			stack := errors.GetStackTrace(err)
			Expect(stack.Frames).To(HaveLen(3))
			Expect(stack.String()).To(Not(BeEmpty()))
		})

		It("Should handle different depths", func() {
			baseErr := errors.New("base error")
			err1 := errors.WithStackDepth(baseErr, 0)
			err2 := errors.WithStackDepth(baseErr, 1)
			Expect(errors.Is(err1, baseErr)).To(BeTrue())
			Expect(errors.Is(err2, baseErr)).To(BeTrue())
			Expect(err1.Error()).To(Equal(err2.Error()))
		})
	})
})
