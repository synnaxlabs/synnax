// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	stderrors "errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

type fakeCloser struct {
	closed  bool
	closeFn func() error
}

func (f *fakeCloser) Close() error {
	f.closed = true
	if f.closeFn != nil {
		return f.closeFn()
	}
	return nil
}

var _ = Describe("MustOpen", func() {
	It("should return the value when the error is nil", func() {
		fc := &fakeCloser{}
		Expect(MustOpen(fc, nil)).To(BeIdenticalTo(fc))
	})

	It("should register a cleanup that closes the value", func() {
		fc := &fakeCloser{}
		DeferCleanup(func() {
			Expect(fc.closed).To(BeTrue())
		})
		MustOpen(fc, nil)
		Expect(fc.closed).To(BeFalse())
	})

	It("should fail the assertion when the error is non-nil", func() {
		failures := InterceptGomegaFailures(func() {
			MustOpen(&fakeCloser{}, stderrors.New("boom"))
		})
		Expect(failures).ToNot(BeEmpty())
	})
})
