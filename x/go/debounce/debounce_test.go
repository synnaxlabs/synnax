// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package debounce_test

import (
	"context"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/debounce"
)

var _ = Describe("Debouncer", func() {
	Describe("Trigger", func() {
		It("Should fire the callback after the delay", func() {
			var called atomic.Int32
			d := debounce.New(20*time.Millisecond, 0, func(ctx context.Context) {
				called.Add(1)
			})
			d.Trigger()
			Eventually(func() int32 { return called.Load() }).Should(Equal(int32(1)))
		})

		It("Should coalesce rapid triggers into a single invocation", func() {
			var called atomic.Int32
			d := debounce.New(20*time.Millisecond, 0, func(ctx context.Context) {
				called.Add(1)
			})
			for i := 0; i < 10; i++ {
				d.Trigger()
			}
			Eventually(func() int32 { return called.Load() }).Should(Equal(int32(1)))
			Consistently(func() int32 { return called.Load() }, 50*time.Millisecond).Should(Equal(int32(1)))
		})

		It("Should reset the timer on each trigger", func() {
			var called atomic.Int32
			d := debounce.New(30*time.Millisecond, 0, func(ctx context.Context) {
				called.Add(1)
			})
			d.Trigger()
			time.Sleep(15 * time.Millisecond)
			d.Trigger()
			time.Sleep(15 * time.Millisecond)
			Expect(called.Load()).To(Equal(int32(0)))
			Eventually(func() int32 { return called.Load() }).Should(Equal(int32(1)))
		})

		It("Should cancel in-flight work when re-triggered", func() {
			var cancelled atomic.Bool
			d := debounce.New(5*time.Millisecond, 0, func(ctx context.Context) {
				<-ctx.Done()
				cancelled.Store(true)
			})
			d.Trigger()
			time.Sleep(10 * time.Millisecond)
			d.Trigger()
			Eventually(func() bool { return cancelled.Load() }).Should(BeTrue())
		})
	})

	Describe("MaxDelay", func() {
		It("Should fire before the trailing delay when max delay is exceeded", func() {
			var called atomic.Int32
			d := debounce.New(50*time.Millisecond, 80*time.Millisecond, func(ctx context.Context) {
				called.Add(1)
			})
			d.Trigger()
			time.Sleep(40 * time.Millisecond)
			d.Trigger()
			time.Sleep(40 * time.Millisecond)
			d.Trigger()
			Eventually(func() int32 { return called.Load() }).Should(BeNumerically(">=", int32(1)))
		})

		It("Should fire immediately when max delay is fully elapsed", func() {
			var called atomic.Int32
			d := debounce.New(50*time.Millisecond, 30*time.Millisecond, func(ctx context.Context) {
				called.Add(1)
			})
			d.Trigger()
			time.Sleep(35 * time.Millisecond)
			d.Trigger()
			time.Sleep(10 * time.Millisecond)
			Expect(called.Load()).To(BeNumerically(">=", int32(1)))
		})
	})

	Describe("Stop", func() {
		It("Should prevent a pending trigger from firing", func() {
			var called atomic.Int32
			d := debounce.New(20*time.Millisecond, 0, func(ctx context.Context) {
				called.Add(1)
			})
			d.Trigger()
			d.Stop()
			Consistently(func() int32 { return called.Load() }, 50*time.Millisecond).Should(Equal(int32(0)))
		})

		It("Should cancel in-flight work", func() {
			var cancelled atomic.Bool
			d := debounce.New(5*time.Millisecond, 0, func(ctx context.Context) {
				<-ctx.Done()
				cancelled.Store(true)
			})
			d.Trigger()
			time.Sleep(10 * time.Millisecond)
			d.Stop()
			Eventually(func() bool { return cancelled.Load() }).Should(BeTrue())
		})

		It("Should be safe to call multiple times", func() {
			d := debounce.New(20*time.Millisecond, 0, func(ctx context.Context) {})
			d.Stop()
			d.Stop()
		})

		It("Should reset max delay tracking", func() {
			var called atomic.Int32
			d := debounce.New(20*time.Millisecond, 100*time.Millisecond, func(ctx context.Context) {
				called.Add(1)
			})
			d.Trigger()
			d.Stop()
			time.Sleep(10 * time.Millisecond)
			d.Trigger()
			Eventually(func() int32 { return called.Load() }).Should(Equal(int32(1)))
		})
	})
})
