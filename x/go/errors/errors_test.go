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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
)

var _ = Describe("Is", func() {
	DescribeTable("IsAny", func(err error, errs []error, expected bool) {
		Expect(errors.IsAny(err, errs...)).To(BeEquivalentTo(expected))
	},
		Entry("Should return false if no errors are given", fmt.Errorf("test"), []error{}, false),
		Entry("Should return false if no errors are the same as the given error", fmt.Errorf("test"), []error{fmt.Errorf("test1"), fmt.Errorf("test2")}, false),
		Entry("Should return true if any of the errors are the same as the given error", fmt.Errorf("test"), []error{fmt.Errorf("test1"), fmt.Errorf("test")}, true),
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
})
