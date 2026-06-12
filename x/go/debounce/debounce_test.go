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
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/debounce"
	. "github.com/synnaxlabs/x/testutil"
	xtime "github.com/synnaxlabs/x/time"
)

func noopCallback(context.Context) {}

var _ = Describe("Debounce", func() {
	Describe("Config", func() {
		Describe("Override", func() {
			It("Should override Delay when the override is non-zero", func() {
				base := debounce.Config{Delay: 10 * time.Millisecond}
				o := debounce.Config{Delay: 20 * time.Millisecond}
				Expect(base.Override(o).Delay).To(Equal(20 * time.Millisecond))
			})
			It("Should preserve Delay when the override is zero", func() {
				base := debounce.Config{Delay: 10 * time.Millisecond}
				Expect(base.Override(debounce.Config{}).Delay).
					To(Equal(10 * time.Millisecond))
			})
			It("Should override MaxDelay when the override is non-zero", func() {
				base := debounce.Config{MaxDelay: 10 * time.Millisecond}
				o := debounce.Config{MaxDelay: 20 * time.Millisecond}
				Expect(base.Override(o).MaxDelay).To(Equal(20 * time.Millisecond))
			})
			It("Should preserve MaxDelay when the override is zero", func() {
				base := debounce.Config{MaxDelay: 10 * time.Millisecond}
				Expect(base.Override(debounce.Config{}).MaxDelay).
					To(Equal(10 * time.Millisecond))
			})
			It("Should override Clock when the override is non-nil", func() {
				base := debounce.Config{Clock: xtime.Real}
				fake := &xtime.Fake{}
				Expect(base.Override(debounce.Config{Clock: fake}).Clock).
					To(Equal(xtime.Clock(fake)))
			})
			It("Should preserve Clock when the override is nil", func() {
				base := debounce.Config{Clock: xtime.Real}
				Expect(base.Override(debounce.Config{}).Clock).NotTo(BeNil())
			})
			It("Should override Callback when the override is non-nil", func() {
				var called atomic.Bool
				base := debounce.Config{Callback: noopCallback}
				o := debounce.Config{
					Callback: func(context.Context) { called.Store(true) },
				}
				base.Override(o).Callback(context.Background())
				Expect(called.Load()).To(BeTrue())
			})
			It("Should preserve Callback when the override is nil", func() {
				base := debounce.Config{Callback: noopCallback}
				Expect(base.Override(debounce.Config{}).Callback).NotTo(BeNil())
			})
		})

		Describe("Validate", func() {
			It("Should fail when Delay is zero", func() {
				cfg := debounce.Config{
					Clock:    xtime.Real,
					Callback: noopCallback,
				}
				Expect(cfg.Validate()).To(MatchError(ContainSubstring("delay")))
			})
			It("Should fail when Delay is negative", func() {
				cfg := debounce.Config{
					Delay:    -1,
					Clock:    xtime.Real,
					Callback: noopCallback,
				}
				Expect(cfg.Validate()).To(MatchError(ContainSubstring("delay")))
			})
			It("Should fail when MaxDelay is negative", func() {
				cfg := debounce.Config{
					Delay:    time.Millisecond,
					MaxDelay: -1,
					Clock:    xtime.Real,
					Callback: noopCallback,
				}
				Expect(cfg.Validate()).To(MatchError(ContainSubstring("max_delay")))
			})
			It("Should fail when Clock is nil", func() {
				cfg := debounce.Config{
					Delay:    time.Millisecond,
					Callback: noopCallback,
				}
				Expect(cfg.Validate()).To(MatchError(ContainSubstring("clock")))
			})
			It("Should fail when Callback is nil", func() {
				cfg := debounce.Config{
					Delay: time.Millisecond,
					Clock: xtime.Real,
				}
				Expect(cfg.Validate()).To(MatchError(ContainSubstring("callback")))
			})
			It("Should succeed when Delay, Clock, and Callback are all set", func() {
				cfg := debounce.Config{
					Delay:    time.Millisecond,
					Clock:    xtime.Real,
					Callback: noopCallback,
				}
				Expect(cfg.Validate()).To(Succeed())
			})
		})
	})

	Describe("Trigger", func() {
		It("Should fire the callback once the delay has elapsed", func() {
			clk := &xtime.Fake{}
			var called atomic.Int32
			d := MustSucceed(debounce.New(debounce.Config{
				Delay:    5 * time.Millisecond,
				Clock:    clk,
				Callback: func(context.Context) { called.Add(1) },
			}))
			DeferCleanup(d.Stop)
			d.Trigger()
			Consistently(called.Load, time.Millisecond*20).Should(Equal(int32(0)))
			clk.Advance(5 * time.Millisecond)
			Eventually(called.Load).Should(Equal(int32(1)))
		})

		It("Should coalesce rapid triggers into a single invocation", func() {
			clk := &xtime.Fake{}
			var called atomic.Int32
			d := MustSucceed(debounce.New(debounce.Config{
				Delay:    20 * time.Millisecond,
				Clock:    clk,
				Callback: func(context.Context) { called.Add(1) },
			}))
			DeferCleanup(d.Stop)
			for range 10 {
				d.Trigger()
			}
			clk.Advance(20 * time.Millisecond)
			Eventually(called.Load).Should(Equal(int32(1)))
			clk.Advance(time.Hour)
			Consistently(called.Load, time.Millisecond*20).Should(Equal(int32(1)))
		})

		It("Should cancel in-flight work when re-triggered", func() {
			clk := &xtime.Fake{}
			var (
				once      sync.Once
				started   = make(chan struct{})
				cancelled atomic.Bool
			)
			d := MustSucceed(debounce.New(debounce.Config{
				Delay: 5 * time.Millisecond,
				Clock: clk,
				Callback: func(ctx context.Context) {
					once.Do(func() { close(started) })
					<-ctx.Done()
					cancelled.Store(true)
				},
			}))
			DeferCleanup(d.Stop)
			d.Trigger()
			clk.Advance(5 * time.Millisecond)
			Eventually(started).Should(BeClosed())
			d.Trigger()
			Eventually(cancelled.Load).Should(BeTrue())
		})
	})

	Describe("MaxDelay", func() {
		It("Should clamp the delay when the cap would otherwise be exceeded", func() {
			clk := &xtime.Fake{}
			var called atomic.Int32
			d := MustSucceed(debounce.New(debounce.Config{
				Delay:    time.Hour,
				MaxDelay: 30 * time.Millisecond,
				Clock:    clk,
				Callback: func(context.Context) { called.Add(1) },
			}))
			DeferCleanup(d.Stop)
			d.Trigger()
			clk.Advance(15 * time.Millisecond)
			d.Trigger()
			clk.Advance(30 * time.Millisecond)
			d.Trigger()
			Eventually(called.Load).Should(Equal(int32(1)))
		})

		It("Should fire immediately when the cap is fully elapsed", func() {
			clk := &xtime.Fake{}
			var called atomic.Int32
			d := MustSucceed(debounce.New(debounce.Config{
				Delay:    time.Hour,
				MaxDelay: 30 * time.Millisecond,
				Clock:    clk,
				Callback: func(context.Context) { called.Add(1) },
			}))
			DeferCleanup(d.Stop)
			d.Trigger()
			clk.Advance(time.Hour)
			d.Trigger()
			Eventually(called.Load).Should(Equal(int32(1)))
		})

		It("Should not clamp the delay when only one trigger fits inside the cap", func() {
			clk := &xtime.Fake{}
			var called atomic.Int32
			d := MustSucceed(debounce.New(debounce.Config{
				Delay:    5 * time.Millisecond,
				MaxDelay: time.Hour,
				Clock:    clk,
				Callback: func(context.Context) { called.Add(1) },
			}))
			DeferCleanup(d.Stop)
			d.Trigger()
			clk.Advance(5 * time.Millisecond)
			Eventually(called.Load).Should(Equal(int32(1)))
		})
	})

	Describe("Stop", func() {
		It("Should prevent a pending trigger from firing", func() {
			clk := &xtime.Fake{}
			var called atomic.Int32
			d := MustSucceed(debounce.New(debounce.Config{
				Delay:    20 * time.Millisecond,
				Clock:    clk,
				Callback: func(context.Context) { called.Add(1) },
			}))
			d.Trigger()
			d.Stop()
			clk.Advance(time.Hour)
			Consistently(called.Load, time.Millisecond*20).Should(Equal(int32(0)))
		})

		It("Should cancel in-flight work", func() {
			clk := &xtime.Fake{}
			var (
				once      sync.Once
				started   = make(chan struct{})
				cancelled atomic.Bool
			)
			d := MustSucceed(debounce.New(debounce.Config{
				Delay: 5 * time.Millisecond,
				Clock: clk,
				Callback: func(ctx context.Context) {
					once.Do(func() { close(started) })
					<-ctx.Done()
					cancelled.Store(true)
				},
			}))
			d.Trigger()
			clk.Advance(5 * time.Millisecond)
			Eventually(started).Should(BeClosed())
			d.Stop()
			Eventually(cancelled.Load).Should(BeTrue())
		})

		It("Should be safe to call multiple times", func() {
			d := MustSucceed(debounce.New(debounce.Config{
				Delay:    20 * time.Millisecond,
				Callback: noopCallback,
			}))
			d.Stop()
			d.Stop()
		})

		It("Should reset MaxDelay tracking so the next Trigger is not clamped", func() {
			clk := &xtime.Fake{}
			var called atomic.Int32
			d := MustSucceed(debounce.New(debounce.Config{
				Delay:    5 * time.Millisecond,
				MaxDelay: time.Hour,
				Clock:    clk,
				Callback: func(context.Context) { called.Add(1) },
			}))
			DeferCleanup(d.Stop)
			d.Trigger()
			clk.Advance(5 * time.Millisecond)
			Eventually(called.Load).Should(Equal(int32(1)))
			d.Stop()
			clk.Advance(2 * time.Hour)
			d.Trigger()
			clk.Advance(4 * time.Millisecond)
			Consistently(called.Load, time.Millisecond*20).Should(Equal(int32(1)))
			clk.Advance(time.Millisecond)
			Eventually(called.Load).Should(Equal(int32(2)))
		})
	})
})
