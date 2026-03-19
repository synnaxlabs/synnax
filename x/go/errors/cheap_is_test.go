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
	"io"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("CheapIs", func() {
	var (
		errBase    = errors.New("base")
		errChild   = errors.Wrap(errBase, "child")
		errUnknown = errors.New("unknown")
	)

	Describe("sentinel matching", func() {
		It("Should match a sentinel against itself", func() {
			Expect(errors.CheapIs(errBase, errBase)).To(BeTrue())
		})
		It("Should match a wrapped error against its sentinel", func() {
			Expect(errors.CheapIs(errChild, errBase)).To(BeTrue())
		})
		It("Should not match a sentinel against its child", func() {
			Expect(errors.CheapIs(errBase, errChild)).To(BeFalse())
		})
		It("Should not match unrelated errors", func() {
			Expect(errors.CheapIs(errUnknown, errBase)).To(BeFalse())
		})
	})

	Describe("multi-wrap", func() {
		It("Should match through multiple layers of wrapping", func() {
			wrapped := errors.Wrap(errors.Wrap(errBase, "layer1"), "layer2")
			Expect(errors.CheapIs(wrapped, errBase)).To(BeTrue())
		})
		It("Should match an intermediate sentinel in the chain", func() {
			wrapped := errors.Wrap(errChild, "outer")
			Expect(errors.CheapIs(wrapped, errChild)).To(BeTrue())
			Expect(errors.CheapIs(wrapped, errBase)).To(BeTrue())
		})
	})

	Describe("join errors", func() {
		It("Should match a sentinel inside a joined error", func() {
			joined := errors.Join(errBase, errUnknown)
			Expect(errors.CheapIs(joined, errBase)).To(BeTrue())
			Expect(errors.CheapIs(joined, errUnknown)).To(BeTrue())
		})
		It("Should match a wrapped sentinel inside a joined error", func() {
			joined := errors.Join(errChild, errUnknown)
			Expect(errors.CheapIs(joined, errBase)).To(BeTrue())
		})
	})

	Describe("nil handling", func() {
		It("Should return true for nil == nil", func() {
			Expect(errors.CheapIs(nil, nil)).To(BeTrue())
		})
		It("Should return false for nil ref with non-nil error", func() {
			Expect(errors.CheapIs(errBase, nil)).To(BeFalse())
		})
		It("Should return false for nil error with non-nil ref", func() {
			Expect(errors.CheapIs(nil, errBase)).To(BeFalse())
		})
	})

	Describe("encode round-trip", func() {
		It("Should match after encode→decode because decode re-wraps sentinel", func() {
			original := errors.Wrap(query.ErrNotFound, "channel 123")
			pld := errors.Encode(ctx, original, false)
			decoded := errors.Decode(ctx, pld)
			Expect(errors.CheapIs(decoded, query.ErrNotFound)).To(BeTrue())
			Expect(errors.CheapIs(decoded, query.ErrQuery)).To(BeTrue())
		})
	})

	Describe("stdlib errors", func() {
		It("Should match io.EOF", func() {
			wrapped := errors.Wrap(io.EOF, "stream ended")
			Expect(errors.CheapIs(wrapped, io.EOF)).To(BeTrue())
		})
	})

	Describe("zero allocations", func() {
		sentinel := errors.New("sentinel")
		wrapped := errors.Wrap(errors.Wrap(sentinel, "inner"), "outer")
		It("Should not allocate when matching a wrapped sentinel", func() {
			Expect(testing.AllocsPerRun(100, func() {
				errors.CheapIs(wrapped, sentinel)
			})).To(BeZero())
		})
		It("Should not allocate when the error does not match", func() {
			other := errors.New("other")
			Expect(testing.AllocsPerRun(100, func() {
				errors.CheapIs(wrapped, other)
			})).To(BeZero())
		})
	})
})
