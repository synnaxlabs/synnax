// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"context"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

// StartRun calls run in a goroutine with a child of ctx. A DeferCleanup ends that
// child, waits for run to return, and asserts that its error satisfies every matcher,
// or is nil when there are no matchers.
//
// ctx must outlive the spec. The SpecContext of a setup node ends with that node, so
// pass GinkgoT().Context() from BeforeEach or BeforeAll.
func StartRun(
	ctx context.Context,
	run func(ctx context.Context) error,
	matchers ...types.GomegaMatcher,
) {
	ginkgo.GinkgoHelper()
	ctx, cancel := context.WithCancel(ctx)
	errs := make(chan error, 1)
	go func() { errs <- run(ctx) }()
	matcher := gomega.Succeed()
	if len(matchers) > 0 {
		matcher = gomega.SatisfyAll(matchers...)
	}
	ginkgo.DeferCleanup(func() {
		cancel()
		var err error
		gomega.Eventually(errs).Should(gomega.Receive(&err))
		gomega.Expect(err).To(matcher)
	})
}
