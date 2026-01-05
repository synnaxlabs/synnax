// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package time_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xtime "github.com/synnaxlabs/x/time"
	"time"
)

var _ = Describe("Ticker", func() {
	Describe("ScaledTicker", func() {
		It("Should scale the duration between ticks", func() {
			t := xtime.NewScaledTicker(1*time.Millisecond, 2)
			defer t.Stop()
			Expect(<-t.C).To(Equal(2 * time.Millisecond))
			Expect(<-t.C).To(Equal(4 * time.Millisecond))
		})
	})
})
