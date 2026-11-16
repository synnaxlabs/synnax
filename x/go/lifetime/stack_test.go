// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lifetime_test

import (
	"context"
	"runtime/pprof"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/lifetime"
)

// recorder runs a component that blocks until its context ends, and records the order
// in which the components see their context end.
type recorder struct {
	mu      sync.Mutex
	stopped []string
}

func (r *recorder) run(name string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		<-ctx.Done()
		r.mu.Lock()
		defer r.mu.Unlock()
		r.stopped = append(r.stopped, name)
		return nil
	}
}

func (r *recorder) order() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stopped
}

var _ = Describe("Stack", func() {
	It("stops the newest lifetime first and waits for each run", func(ctx SpecContext) {
		var r recorder
		s := lifetime.NewStack()
		storage := s.Go(ctx, "storage", r.run("storage"))
		service := s.Go(storage, "service", r.run("service"))
		s.Go(service, "server", r.run("server"))
		Expect(s.Stop(ctx)).To(Succeed())
		Expect(r.order()).To(Equal([]string{"server", "service", "storage"}))
	})

	It("keeps the outer lifetimes running while an inner run drains", func(
		ctx SpecContext,
	) {
		var r recorder
		s := lifetime.NewStack()
		outer := s.Go(ctx, "outer", r.run("outer"))
		var outerErr error
		s.Go(outer, "inner", func(ctx context.Context) error {
			<-ctx.Done()
			outerErr = outer.Err()
			return nil
		})
		Expect(s.Stop(ctx)).To(Succeed())
		Expect(outerErr).ToNot(HaveOccurred())
		Expect(r.order()).To(Equal([]string{"outer"}))
	})

	It("nests each lifetime inside the one before it", func(ctx SpecContext) {
		parent, cancel := context.WithCancel(ctx)
		var r recorder
		s := lifetime.NewStack()
		outer := s.Go(parent, "outer", r.run("outer"))
		s.Go(outer, "inner", r.run("inner"))
		cancel()
		Eventually(r.order).Should(ConsistOf("outer", "inner"))
		Expect(s.Stop(ctx)).To(Succeed())
	})

	It("labels each goroutine with the name of its run", func(ctx SpecContext) {
		label := make(chan string, 1)
		s := lifetime.NewStack()
		s.Go(ctx, "storage", func(ctx context.Context) error {
			v, _ := pprof.Label(ctx, "routine")
			label <- v
			<-ctx.Done()
			return nil
		})
		Eventually(label).Should(Receive(Equal("storage")))
		Expect(s.Stop(ctx)).To(Succeed())
	})

	Describe("Failed", func() {
		It("closes when a run returns an error", func(ctx SpecContext) {
			var r recorder
			s := lifetime.NewStack()
			storage := s.Go(ctx, "storage", r.run("storage"))
			s.Go(storage, "service", func(context.Context) error {
				return errors.New("disk full")
			})
			Eventually(s.Failed()).Should(BeClosed())
			Expect(s.Stop(ctx)).To(MatchError(ContainSubstring("service: disk full")))
			Expect(r.order()).To(Equal([]string{"storage"}))
		})

		It("stays open when a run returns nil before Stop", func(ctx SpecContext) {
			var (
				r        recorder
				returned = make(chan struct{})
			)
			s := lifetime.NewStack()
			storage := s.Go(ctx, "storage", func(context.Context) error {
				close(returned)
				return nil
			})
			s.Go(storage, "service", r.run("service"))
			Eventually(returned).Should(BeClosed())
			Consistently(s.Failed()).ShouldNot(BeClosed())
			Expect(r.order()).To(BeEmpty())
			Expect(s.Stop(ctx)).To(Succeed())
			Expect(r.order()).To(Equal([]string{"service"}))
		})
	})

	Describe("Deadline", func() {
		It("ends every lifetime and names the runs that did not return", func(
			ctx SpecContext,
		) {
			var (
				r       recorder
				release = make(chan struct{})
				stuck   = make(chan struct{})
			)
			DeferCleanup(func() {
				close(release)
				Eventually(stuck).Should(BeClosed())
			})
			s := lifetime.NewStack()
			storage := s.Go(ctx, "storage", r.run("storage"))
			service := s.Go(storage, "service", func(context.Context) error {
				defer close(stuck)
				<-release
				return nil
			})
			s.Go(service, "server", r.run("server"))
			stopCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()
			err := s.Stop(stopCtx)
			Expect(err).To(MatchError(context.DeadlineExceeded))
			Expect(err).To(MatchError(ContainSubstring("service")))
			Expect(err).To(MatchError(ContainSubstring("did not return")))
			Expect(err).ToNot(MatchError(ContainSubstring("server")))
			Eventually(r.order).Should(Equal([]string{"server", "storage"}))
		})

		It("joins the errors of the runs that returned", func(ctx SpecContext) {
			release := make(chan struct{})
			stuck := make(chan struct{})
			DeferCleanup(func() {
				close(release)
				Eventually(stuck).Should(BeClosed())
			})
			s := lifetime.NewStack()
			storage := s.Go(ctx, "storage", func(context.Context) error {
				defer close(stuck)
				<-release
				return nil
			})
			s.Go(storage, "server", func(ctx context.Context) error {
				<-ctx.Done()
				return errors.New("drain failed")
			})
			stopCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()
			err := s.Stop(stopCtx)
			Expect(err).To(MatchError(ContainSubstring("server: drain failed")))
			Expect(err).To(MatchError(ContainSubstring("storage did not return")))
			Expect(err).To(MatchError(context.DeadlineExceeded))
		})
	})
})
