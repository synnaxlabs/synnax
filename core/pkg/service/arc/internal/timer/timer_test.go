// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package timer_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/arc/internal/timer"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Now", func() {
	It("Should advance at the rate of the system clock", func() {
		start, wall := timer.Now(), time.Now()
		time.Sleep(50 * time.Millisecond)
		Expect(timer.Now() - start).
			To(BeNumerically("~", time.Since(wall), time.Millisecond))
	})
})

var _ = Describe("Timer", func() {
	var t *timer.Timer
	BeforeEach(func() { t = MustOpen(timer.New()) })

	It("Should not fire before Reset", func() {
		Consistently(t.C).WithTimeout(20 * time.Millisecond).ShouldNot(Receive())
	})

	It("Should fire on its deadline and not before", func() {
		Expect(t.Reset(50 * time.Millisecond)).To(Succeed())
		Consistently(t.C).
			WithTimeout(45 * time.Millisecond).
			WithPolling(time.Millisecond).
			ShouldNot(Receive())
		Eventually(t.C).WithTimeout(time.Second).Should(Receive())
	})

	DescribeTable("Should fire right away for a span of zero or less",
		func(d time.Duration) {
			Expect(t.Reset(d)).To(Succeed())
			Eventually(t.C).WithTimeout(50 * time.Millisecond).Should(Receive())
		},
		Entry("zero", time.Duration(0)),
		Entry("negative", -time.Second),
	)

	It("Should fire once per Reset", func() {
		Expect(t.Reset(time.Millisecond)).To(Succeed())
		Eventually(t.C).WithTimeout(time.Second).Should(Receive())
		Consistently(t.C).WithTimeout(20 * time.Millisecond).ShouldNot(Receive())
	})

	It("Should fire at the span of the latest Reset", func() {
		Expect(t.Reset(time.Hour)).To(Succeed())
		Expect(t.Reset(5 * time.Millisecond)).To(Succeed())
		Eventually(t.C).WithTimeout(time.Second).Should(Receive())
		Expect(t.Reset(5 * time.Millisecond)).To(Succeed())
		Expect(t.Reset(time.Hour)).To(Succeed())
		Consistently(t.C).WithTimeout(30 * time.Millisecond).ShouldNot(Receive())
	})

	It("Should not fire after Stop", func() {
		Expect(t.Reset(10 * time.Millisecond)).To(Succeed())
		Expect(t.Stop()).To(Succeed())
		Consistently(t.C).WithTimeout(40 * time.Millisecond).ShouldNot(Receive())
	})

	It("Should drop a pending fire on Stop", func() {
		Expect(t.Reset(0)).To(Succeed())
		Eventually(func() int { return len(t.C) }).Should(Equal(1))
		Expect(t.Stop()).To(Succeed())
		Expect(t.C).ToNot(Receive())
	})

	It("Should wake more often than once per millisecond", func() {
		const (
			period = 500 * time.Microsecond
			fires  = 400
		)
		start := time.Now()
		for range fires {
			Expect(t.Reset(period)).To(Succeed())
			Eventually(t.C).
				WithTimeout(time.Second).
				WithPolling(10 * time.Microsecond).
				Should(Receive())
		}
		Expect(time.Since(start)).To(BeNumerically("<", fires*875*time.Microsecond))
	})
})
