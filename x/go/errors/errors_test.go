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

type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}

var _ = Describe("Is", func() {
	Describe("Is", func() {
		It("Should work with sub-errors", func() {
			err1 := errors.New("err1")
			errUnderneath := errors.Wrap(err1, "errUnderneath")
			By("Returning true for sub-errors")
			Expect(errors.Is(errUnderneath, err1)).To(BeTrue())
			By("Returning false for same layer errors")
			Expect(errors.Is(errUnderneath, errUnderneath)).To(BeTrue())
			By("Returning false for parent errors")
			Expect(errors.Is(err1, errUnderneath)).To(BeFalse())
			Expect(errors.Is(err1, err1)).To(BeTrue())
		})

		It("Should return false for errors with completely different types", func() {
			err1 := errors.New("err1")
			err2 := errors.New("err2")
			Expect(errors.Is(err1, err2)).To(BeFalse())
		})
	})

	Describe("Wrap", func() {
		base := errors.New("base")
		It("Should wrap the given error with a sub message", func() {
			wrapped := errors.Wrap(base, "wrapped")
			Expect(errors.Is(wrapped, base)).To(BeTrue())
			Expect(wrapped).To(MatchError("wrapped: base"))
		})

		Describe("Wrapf", func() {
			It("Should wrap an error with a format", func() {
				wrapped := errors.Wrapf(base, "wrapped %s", "cat")
				Expect(errors.Is(wrapped, base)).To(BeTrue())
				Expect(wrapped).To(MatchError("wrapped cat: base"))
			})
		})
	})

	DescribeTable("IsAny", func(err error, errs []error, expected bool) {
		Expect(errors.IsAny(err, errs...)).To(BeEquivalentTo(expected))
	},
		Entry("Should return false if no errors are given", errors.Newf("test"), []error{}, false),
		Entry("Should return false if no errors are the same as the given error", errors.Newf("test"), []error{errors.Newf("test1"), errors.Newf("test2")}, false),
		Entry("Should return true if any of the errors are the same as the given error", errors.Newf("test"), []error{errors.Newf("test1"), errors.Newf("test")}, true),
	)

	Describe("Skip", func() {
		It("Should return nil if the error matches the reference error", func() {
			err := errors.Newf("test")
			Expect(errors.Skip(err, err)).To(BeNil())
		})
		It("Should return nil if the error is nil", func() {
			Expect(errors.Skip(nil, nil)).To(BeNil())
		})
		It("Should return the error if the error does not match the reference error", func() {
			e1 := errors.Newf("test1")
			e2 := errors.Newf("test2")
			Expect(errors.Skip(e1, e2)).To(Equal(e1))
		})
	})

	Describe("Combine", func() {
		It("Should return err if otherErr is nil", func() {
			err := errors.New("test")
			Expect(errors.Combine(err, nil)).To(Equal(err))
		})

		It("Should return nil if both errors are nil", func() {
			Expect(errors.Combine(nil, nil)).To(BeNil())
		})

		It("Should return otherErr if err is nil", func() {
			otherErr := errors.New("other")
			Expect(errors.Combine(nil, otherErr)).To(Equal(otherErr))
		})

		It("Should combine both errors when both are non-nil", func() {
			err1 := errors.New("err1")
			err2 := errors.New("err2")
			combined := errors.Combine(err1, err2)
			Expect(errors.Is(combined, err1)).To(BeTrue())
			// Expect this to be attached as a secondary error.
			Expect(errors.Is(combined, err2)).To(BeFalse())
		})
	})

	Describe("Join", func() {
		It("Should return nil if no errors are provided", func() {
			Expect(errors.Join()).To(BeNil())
		})

		It("Should return nil if all errors are nil", func() {
			Expect(errors.Join(nil, nil, nil)).To(BeNil())
		})

		It("Should join multiple errors", func() {
			err1 := errors.New("err1")
			err2 := errors.New("err2")
			joined := errors.Join(err1, err2)
			Expect(errors.Is(joined, err1)).To(BeTrue())
			Expect(errors.Is(joined, err2)).To(BeTrue())
		})

		It("Should skip nil errors when joining", func() {
			err1 := errors.New("err1")
			err2 := errors.New("err2")
			joined := errors.Join(err1, nil, err2)
			Expect(errors.Is(joined, err1)).To(BeTrue())
			Expect(errors.Is(joined, err2)).To(BeTrue())
		})
	})

	Describe("New", func() {
		It("Should create a new error with the given message", func() {
			err := errors.New("test error")
			Expect(err.Error()).To(Equal("test error"))
		})

		It("Should create unique errors for different messages", func() {
			err1 := errors.New("test1")
			err2 := errors.New("test2")
			Expect(errors.Is(err1, err2)).To(BeFalse())
		})
	})

	Describe("Newf", func() {
		It("Should create a new error with formatted message", func() {
			err := errors.Newf("test %s", "error")
			Expect(err.Error()).To(Equal("test error"))
		})

		It("Should handle multiple format arguments", func() {
			err := errors.Newf("test %s %d", "error", 123)
			Expect(err.Error()).To(Equal("test error 123"))
		})
	})

	Describe("As", func() {
		It("Should return false for non-matching error types", func() {
			err := errors.New("test")
			var errCustom *customError
			Expect(errors.As(err, &errCustom)).To(BeFalse())
		})

		It("Should return true and set target for matching error types", func() {
			originalErr := &customError{msg: "test"}
			err := errors.Wrap(originalErr, "wrapped")
			var errCustom *customError
			Expect(errors.As(err, &errCustom)).To(BeTrue())
			Expect(errCustom).To(Equal(originalErr))
		})

		It("Should panic if target is not a pointer", func() {
			err := errors.New("test")
			var errCustom customError
			Expect(func() { errors.As(err, errCustom) }).To(Panic())
		})
	})
})
