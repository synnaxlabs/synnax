package time_test

import (
	xtime "github.com/arya-analytics/x/time"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Ticker", func() {
	Describe("ScaledTicker", func() {
		It("Should scale the duration between ticks", func() {
			t := xtime.NewScaledTicker(1*time.Millisecond, 2)
			t0 := time.Now()
			defer t.Stop()
			Expect(<-t.C).To(BeTemporally("~", t0.Add(1*time.Millisecond), 700*time.Microsecond))
			Expect(<-t.C).To(BeTemporally("~", t0.Add(3*time.Millisecond), 700*time.Microsecond))
		})
	})
})
