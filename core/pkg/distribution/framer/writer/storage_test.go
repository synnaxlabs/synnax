// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Unindexed Stamper", func() {
	It("Should not allocate when it stamps a frame", func() {
		st := newUnindexedStamper([]channel.Key{1, 2, 3})
		keys := []channel.Key{1, 2, 3, 1}
		series := make([]telem.Series, len(keys))
		for i := range series {
			series[i] = telem.NewSeriesV[int64](1, 2)
		}
		fr := telem.MultiFrame(keys, series)
		Expect(testing.AllocsPerRun(100, func() { st.stamp(fr) })).To(BeZero())
	})

	It("Should not stamp a series the frame excludes", func() {
		st := newUnindexedStamper([]channel.Key{1, 2})
		fr := telem.MultiFrame(
			[]channel.Key{1, 2},
			[]telem.Series{telem.NewSeriesV[int64](1, 2), telem.NewSeriesV[int64](3)},
		)
		st.stamp(fr.ExcludeKeys([]channel.Key{2}))
		Expect(fr.RawSeriesAt(0).TimeRange.Span()).To(Equal(2 * telem.Nanosecond))
		Expect(fr.RawSeriesAt(1).TimeRange.IsZero()).To(BeTrue())
	})
})
