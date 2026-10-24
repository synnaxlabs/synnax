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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/testutil"
)

var _ = Describe("SuiteConfig", func() {
	It("Should randomize the order of every spec", func() {
		Expect(testutil.SuiteConfig().RandomizeAllSpecs).To(BeTrue())
	})

	It("Should fail the suite on a pending spec", func() {
		Expect(testutil.SuiteConfig().FailOnPending).To(BeTrue())
	})

	It("Should keep the parallel process that the command line set", func() {
		cfg := testutil.SuiteConfig()
		Expect(cfg.ParallelProcess).To(Equal(GinkgoParallelProcess()))
	})
})
