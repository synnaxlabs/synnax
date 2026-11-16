// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sync_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xsync "github.com/synnaxlabs/x/sync"
	"golang.org/x/sync/errgroup"
)

var _ = Describe("Wait", func() {
	It("Should return the error of a group that finishes first", func(ctx SpecContext) {
		var g errgroup.Group
		g.Go(func() error { return errors.New("cat") })
		Expect(xsync.Wait(ctx, &g)).To(MatchError("cat"))
	})

	It("Should return nil when the group succeeds", func(ctx SpecContext) {
		var g errgroup.Group
		g.Go(func() error { return nil })
		Expect(xsync.Wait(ctx, &g)).To(Succeed())
	})

	It("Should return the context error on a deadline", func(ctx SpecContext) {
		var g errgroup.Group
		release := make(chan struct{})
		g.Go(func() error { <-release; return nil })
		waitCtx, cancel := context.WithTimeout(ctx, 5*time.Millisecond)
		defer cancel()
		Expect(xsync.Wait(waitCtx, &g)).To(MatchError(context.DeadlineExceeded))
		close(release)
	})
})
