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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Time filter", func() {
	It(
		"Should read no time beside an untimed channel that no timed channel needs",
		func(
			ctx SpecContext,
		) {
			node := testutil.NewNode(ctx)
			iteratorSvc, channelSvc := openServices(ctx, node)
			idx := &channel.Channel{
				Name:     "filter_index",
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, idx)
			})).To(Succeed())
			timed := &channel.Channel{
				Name:       "filter_timed",
				DataType:   telem.Float32T,
				LocalIndex: idx.LocalKey,
			}
			untimed := &channel.Channel{
				Name:       "filter_untimed",
				DataType:   telem.Float32T,
				LocalIndex: idx.LocalKey,
			}
			for _, ch := range []*channel.Channel{timed, untimed} {
				Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).Create(ctx, ch)
				})).To(Succeed())
			}
			// The timed channel holds samples at 1 to 3 seconds, and the untimed one at
			// 10
			// to 12 seconds.
			for start, data := range map[telem.TimeStamp]channel.Key{
				telem.SecondTS:      timed.Key(),
				10 * telem.SecondTS: untimed.Key(),
			} {
				keys := channel.Keys{idx.Key(), data}
				w := MustSucceed(node.Framer.OpenWriter(ctx, framer.WriterConfig{
					Start: start,
					Keys:  keys,
				}))
				s := start / telem.SecondTS
				Expect(w.Write(frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesSecondsTSV(s, s+1, s+2),
					telem.NewSeriesV[float32](1, 2, 3),
				}))).To(BeTrue())
				Expect(w.Close()).To(Succeed())
			}
			iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
				Keys:        channel.Keys{timed.Key(), untimed.Key()},
				UntimedKeys: channel.Keys{untimed.Key()},
				Bounds: telem.TimeRange{
					Start: 10 * telem.SecondTS,
					End:   telem.TimeStampMax,
				},
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
			Expect(iter.Value().KeysSlice()).To(Equal([]channel.Key{untimed.Key()}))
			Expect(iter.Close()).To(Succeed())
		},
	)
})
