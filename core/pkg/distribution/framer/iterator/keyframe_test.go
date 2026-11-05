// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("SeekKeyframe", func() {
	DescribeTable("should seek to the last keyframe through", func(
		ctx SpecContext,
		leaseholder node.Key,
	) {
		cluster := DeferClose(testutil.OpenCluster(ctx, 2))
		dist := cluster.Nodes[1]
		idx := MustSucceed(dist.Channel.Create(ctx, []channel.Channel{{
			Name:        "Lumiere",
			IsIndex:     true,
			DataType:    telem.TimestampT,
			Leaseholder: leaseholder,
		}}))[0]
		video := MustSucceed(dist.Channel.Create(ctx, []channel.Channel{{
			Name:        "Edison",
			LocalIndex:  idx.LocalKey,
			DataType:    telem.VideoT,
			Leaseholder: leaseholder,
		}}))[0]
		keys := channel.Keys{idx.Key(), video.Key()}
		frames := make([]telem.VideoFrame, 6)
		for i := range frames {
			frames[i] = telem.VideoFrame{
				Keyframe: i%3 == 0,
				Codec:    telem.VideoCodecH264,
				Data:     []byte{byte(i)},
			}
		}
		w := MustSucceed(dist.Framer.OpenWriter(ctx, writer.Config{
			Keys:  keys,
			Start: 10 * telem.SecondTS,
			Sync:  new(true),
		}))
		Expect(w.Write(frame.NewMulti(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15),
			telem.NewVideoSeries(frames),
		}))).To(BeTrue())
		Expect(w.Commit()).To(BeNumerically(">", 0))
		Expect(w.Close()).To(Succeed())

		iter := MustSucceed(dist.Framer.OpenIterator(ctx, iterator.Config{
			Keys:   keys,
			Bounds: telem.TimeRangeMax,
		}))
		Expect(iter.SeekKeyframe(15 * telem.SecondTS)).To(BeTrue())
		Expect(iter.Next(2 * telem.Second)).To(BeTrue())
		Expect(iter.Value().Get(idx.Key()).Series[0]).
			To(telem.MatchSeriesData(telem.NewSeriesSecondsTSV(13, 14)))
		Expect(iter.Close()).To(Succeed())

		iter = MustSucceed(dist.Framer.OpenIterator(ctx, iterator.Config{
			Keys:   channel.Keys{idx.Key()},
			Bounds: telem.TimeRangeMax,
		}))
		Expect(iter.SeekKeyframe(15 * telem.SecondTS)).To(BeFalse())
		Expect(iter.Error()).To(MatchError(validate.ErrValidation))
		Expect(iter.Error()).To(MatchError(ContainSubstring(
			"cannot seek to a keyframe of a timestamp channel",
		)))
		Expect(iter.Close()).To(MatchError(validate.ErrValidation))
	},
		Entry("the gateway", node.Key(1)),
		Entry("a peer", node.Key(2)),
	)
})
