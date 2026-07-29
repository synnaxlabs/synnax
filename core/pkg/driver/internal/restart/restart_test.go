// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package restart_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/driver/internal/restart"
	. "github.com/synnaxlabs/x/testutil"
)

// instant builds a Policy whose backoff is zero, so Decide returns without waiting.
func instant(
	ctx context.Context,
	maxRetries int,
	healthy time.Duration,
) *restart.Policy {
	return MustSucceed(restart.New(ctx, restart.Config{
		BaseInterval:  0,
		Scale:         1,
		MaxRetries:    maxRetries,
		HealthyUptime: healthy,
	}))
}

var _ = Describe("Policy", func() {
	Describe("New", func() {
		It("Should construct a policy with valid config", func() {
			Expect(restart.New(context.Background(), restart.Config{
				BaseInterval: time.Second, Scale: 1.1, MaxRetries: 10,
			})).Error().ToNot(HaveOccurred())
		})

		It("Should reject a scale below one", func() {
			Expect(restart.New(context.Background(), restart.Config{Scale: 0.5})).
				Error().To(MatchError(ContainSubstring("scale")))
		})
	})

	Describe("expected shutdown", func() {
		DescribeTable(
			"Should return Stop regardless of uptime and never consume a retry",
			func(uptime time.Duration) {
				p := instant(context.Background(), 1, time.Minute)
				Expect(p.Decide(true, uptime)).To(Equal(restart.Stop))
				// The retry budget is untouched: a later unexpected exit still restarts.
				Expect(p.Decide(false, time.Millisecond)).To(Equal(restart.Restart))
			},
			Entry("zero uptime", time.Duration(0)),
			Entry("short uptime", time.Millisecond),
			Entry("long uptime", time.Hour),
		)
	})

	Describe("unexpected exit", func() {
		It("Should restart up to MaxRetries times, then give up", func() {
			p := instant(context.Background(), 3, time.Minute)
			Expect(p.Decide(false, time.Millisecond)).To(Equal(restart.Restart))
			Expect(p.Decide(false, time.Millisecond)).To(Equal(restart.Restart))
			Expect(p.Decide(false, time.Millisecond)).To(Equal(restart.Restart))
			Expect(p.Decide(false, time.Millisecond)).To(Equal(restart.GiveUp))
			Expect(p.Decide(false, time.Millisecond)).To(Equal(restart.GiveUp))
		})

		It("Should give up immediately when MaxRetries is zero", func() {
			p := instant(context.Background(), 0, time.Minute)
			Expect(p.Decide(false, time.Millisecond)).To(Equal(restart.GiveUp))
		})
	})

	Describe("healthy-uptime reset", func() {
		It(
			"Should reset the failure counter after a run that exceeds HealthyUptime",
			func() {
				p := instant(context.Background(), 2, time.Minute)
				By("Exhausting the retry budget with rapid crashes")
				Expect(p.Decide(false, time.Millisecond)).To(Equal(restart.Restart))
				Expect(p.Decide(false, time.Millisecond)).To(Equal(restart.Restart))
				By("Restarting again after a healthy run instead of giving up")
				Expect(p.Decide(false, 2*time.Minute)).To(Equal(restart.Restart))
				By("Granting the full budget again after the reset")
				Expect(p.Decide(false, time.Millisecond)).To(Equal(restart.Restart))
				Expect(p.Decide(false, time.Millisecond)).To(Equal(restart.GiveUp))
			},
		)

		It("Should treat an exit exactly at HealthyUptime as healthy", func() {
			p := instant(context.Background(), 1, time.Minute)
			Expect(p.Decide(false, time.Millisecond)).To(Equal(restart.Restart))
			Expect(p.Decide(false, time.Minute)).To(Equal(restart.Restart))
		})

		It("Should default HealthyUptime when unset", func() {
			p := instant(context.Background(), 1, 0)
			Expect(p.Decide(false, time.Millisecond)).To(Equal(restart.Restart))
			// An uptime past the default (1m) resets, so this restarts rather than
			// giving up.
			Expect(
				p.Decide(false, 2*restart.DefaultHealthyUptime),
			).To(Equal(restart.Restart))
		})
	})

	Describe("context cancellation", func() {
		It("Should give up when the context is canceled during backoff", func() {
			ctx, cancel := context.WithCancel(context.Background())
			p := MustSucceed(restart.New(ctx, restart.Config{
				BaseInterval:  time.Hour,
				Scale:         1,
				MaxRetries:    100,
				HealthyUptime: time.Minute,
			}))
			cancel()
			Expect(p.Decide(false, time.Millisecond)).To(Equal(restart.GiveUp))
		})
	})

	Describe("backoff", func() {
		It("Should wait the base interval before returning Restart", func() {
			p := MustSucceed(restart.New(context.Background(), restart.Config{
				BaseInterval: 30 * time.Millisecond, Scale: 1, MaxRetries: 5,
				HealthyUptime: time.Minute,
			}))
			start := time.Now()
			Expect(p.Decide(false, time.Millisecond)).To(Equal(restart.Restart))
			Expect(time.Since(start)).To(BeNumerically(">=", 20*time.Millisecond))
		})
	})
})
