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
	"context"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"golang.org/x/sync/errgroup"
)

var _ = Describe("StartRun", func() {
	It("ends the run at cleanup and waits for it to return", func() {
		var returned atomic.Bool
		DeferCleanup(func() { Expect(returned.Load()).To(BeTrue()) })
		started := make(chan context.Context, 1)
		StartRun(func(ctx context.Context) error {
			started <- ctx
			<-ctx.Done()
			returned.Store(true)
			return nil
		})
		var runCtx context.Context
		Eventually(started).Should(Receive(&runCtx))
		Consistently(runCtx.Done()).ShouldNot(BeClosed())
	})

	It("returns once the run blocks", func() {
		var blocked atomic.Bool
		StartRun(func(ctx context.Context) error {
			blocked.Store(true)
			<-ctx.Done()
			return nil
		})
		Expect(blocked.Load()).To(BeTrue())
	})

	It("ends the run when stop is called", func() {
		var returned atomic.Bool
		stop := StartRun(func(ctx context.Context) error {
			<-ctx.Done()
			returned.Store(true)
			return nil
		})
		stop()
		Expect(returned.Load()).To(BeTrue())
	})

	It("accepts an error that satisfies the matchers", func() {
		StartRun(func(ctx context.Context) error {
			<-ctx.Done()
			return errors.New("stopped")
		}, MatchError("stopped"))
	})

	It("fails when the run returns an error and there are no matchers", func() {
		var failures *[]string
		DeferCleanup(func() { Expect(*failures).To(HaveLen(1)) })
		failures = interceptCleanupFailures()
		StartRun(func(context.Context) error { return errors.New("failed") })
	})

	Describe("From a setup node", func() {
		var runCtx context.Context
		BeforeEach(func(ctx SpecContext) {
			started := make(chan context.Context, 1)
			StartRun(func(ctx context.Context) error {
				started <- ctx
				<-ctx.Done()
				return nil
			})
			Eventually(started).WithContext(ctx).Should(Receive(&runCtx))
		})

		It("keeps the run going after the setup node ends", func() {
			Expect(runCtx.Err()).ToNot(HaveOccurred())
		})
	})
})

var _ = Describe("GoRun", func() {
	It("returns the error of the run from every stop", func(ctx SpecContext) {
		var blocked atomic.Bool
		stop := GoRun(ctx, func(ctx context.Context) error {
			blocked.Store(true)
			<-ctx.Done()
			return errors.New("stopped")
		})
		Expect(blocked.Load()).To(BeTrue())
		Expect(stop()).To(MatchError("stopped"))
		Expect(stop()).To(MatchError("stopped"))
	})

	It("returns once the goroutines that the run nests are blocked", func(
		ctx SpecContext,
	) {
		var nestedStarted atomic.Bool
		stop := GoRun(ctx, func(ctx context.Context) error {
			var g errgroup.Group
			g.Go(func() error {
				g.Go(func() error {
					nestedStarted.Store(true)
					<-ctx.Done()
					return nil
				})
				<-ctx.Done()
				return nil
			})
			return g.Wait()
		})
		Expect(nestedStarted.Load()).To(BeTrue())
		Expect(stop()).To(Succeed())
	})
})
