// Copyright 2024 Synnax Labs, Inc.
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
})
