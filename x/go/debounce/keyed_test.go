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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/debounce"
	. "github.com/synnaxlabs/x/testutil"
	xtime "github.com/synnaxlabs/x/time"
)

var _ = Describe("Keyed", func() {
	Describe("KeyedConfig", func() {
		Describe("Validate", func() {
			It("Should reject a zero delay", func() {
				cfg := debounce.KeyedConfig[string]{
					Clock:    xtime.Real,
					Callback: func(context.Context, string) {},
				}
				Expect(cfg.Validate()).To(MatchError(ContainSubstring("delay")))
			})
			It("Should reject a nil callback", func() {
				cfg := debounce.KeyedConfig[string]{
					Delay: time.Millisecond,
					Clock: xtime.Real,
				}
				Expect(cfg.Validate()).To(MatchError(ContainSubstring("callback")))
			})
		})
	})

	Describe("NewKeyed", func() {
		It("Should return a validation error for an invalid config", func() {
			Expect(debounce.NewKeyed[string]()).Error().
				To(MatchError(ContainSubstring("delay")))
		})
	})

	Describe("Behavior", func() {
		var (
			clk   *xtime.Fake
			fired chan string
		)

		BeforeEach(func() {
			clk = &xtime.Fake{}
			fired = make(chan string, 16)
		})

		newKeyed := func(cfg debounce.KeyedConfig[string]) *debounce.Keyed[string] {
			GinkgoHelper()
			cfg.Clock = clk
			cfg.Callback = func(_ context.Context, key string) { fired <- key }
			return DeferClose(MustSucceed(debounce.NewKeyed(cfg)))
		}

		It("Should fire the callback with the key after the delay", func() {
			k := newKeyed(debounce.KeyedConfig[string]{Delay: 10 * time.Millisecond})
			k.Trigger("a")
			clk.Advance(10 * time.Millisecond)
			Eventually(fired).Should(Receive(Equal("a")))
		})

		It("Should coalesce rapid triggers into a single fire", func() {
			k := newKeyed(debounce.KeyedConfig[string]{Delay: 10 * time.Millisecond})
			k.Trigger("a")
			clk.Advance(5 * time.Millisecond)
			k.Trigger("a")
			clk.Advance(10 * time.Millisecond)
			Eventually(fired).Should(Receive(Equal("a")))
			Consistently(fired, 50*time.Millisecond, 10*time.Millisecond).
				ShouldNot(Receive())
		})

		It("Should track deadlines for different keys independently", func() {
			k := newKeyed(debounce.KeyedConfig[string]{Delay: 10 * time.Millisecond})
			k.Trigger("a")
			clk.Advance(4 * time.Millisecond)
			k.Trigger("b")
			clk.Advance(6 * time.Millisecond)
			Eventually(fired).Should(Receive(Equal("a")))
			clk.Advance(4 * time.Millisecond)
			Eventually(fired).Should(Receive(Equal("b")))
		})

		It("Should cap the total delay under sustained triggers", func() {
			k := newKeyed(debounce.KeyedConfig[string]{
				Delay:    10 * time.Millisecond,
				MaxDelay: 25 * time.Millisecond,
			})
			for range 5 {
				k.Trigger("a")
				clk.Advance(5 * time.Millisecond)
			}
			Eventually(fired).Should(Receive(Equal("a")))
		})

		It("Should discard a pending fire on Forget", func() {
			k := newKeyed(debounce.KeyedConfig[string]{Delay: 10 * time.Millisecond})
			k.Trigger("a")
			k.Forget("a")
			clk.Advance(20 * time.Millisecond)
			Consistently(fired, 50*time.Millisecond, 10*time.Millisecond).
				ShouldNot(Receive())
		})

		It("Should discard pending fires and ignore triggers after Close", func() {
			k := newKeyed(debounce.KeyedConfig[string]{Delay: 10 * time.Millisecond})
			k.Trigger("a")
			Expect(k.Close()).To(Succeed())
			k.Trigger("b")
			clk.Advance(20 * time.Millisecond)
			Consistently(fired, 50*time.Millisecond, 10*time.Millisecond).
				ShouldNot(Receive())
		})
	})
})
