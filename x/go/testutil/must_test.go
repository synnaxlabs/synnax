// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Must", func() {
	Describe("MustSucceed", func() {
		It("should return the value when no error occurs", func() {
			result := testutil.MustSucceed("test", nil)
			Expect(result).To(Equal("test"))
		})
		It("should panic when an error occurs", func() {
			err := InterceptGomegaFailure(func() {
				testutil.MustSucceed("", errors.New("test error"))
				Fail("should not reach this line")
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("test error"))
		})
	})
	Describe("MustSucceed2", func() {
		It("should return both values when no error occurs", func() {
			a, b := testutil.MustSucceed2("first", 2, nil)
			Expect(a).To(Equal("first"))
			Expect(b).To(Equal(2))
		})
		It("should fail when an error occurs", func() {
			err := InterceptGomegaFailure(func() {
				testutil.MustSucceed2("first", 2, errors.New("test error"))
				Fail("should not reach this line")
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("test error"))
		})
	})
})
