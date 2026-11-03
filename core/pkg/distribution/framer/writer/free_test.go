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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Free Writer", func() {
	It("Should not allocate when it aligns and stamps a frame", func() {
		w := &freeWriter{
			indexes:    map[channel.Key]channel.Key{1: 1, 2: 1},
			alignments: map[channel.Key]telem.Alignment{1: 0},
			stamper:    newUnindexedStamper([]channel.Key{3}),
		}
		fr := frame.NewMulti(
			channel.Keys{1, 2, 3, 3},
			[]telem.Series{
				telem.NewSeriesSecondsTSV(10*telem.SecondTS, 11*telem.SecondTS),
				telem.NewSeriesV[float32](1, 2),
				telem.NewSeriesV[int64](1, 2),
				telem.NewSeriesV[int64](3),
			},
		)
		Expect(testing.AllocsPerRun(100, func() { w.alignFrame(fr) })).To(BeZero())
	})
})
