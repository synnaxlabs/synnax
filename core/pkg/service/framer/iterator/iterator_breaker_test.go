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
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Iterator breaker", func() {
	It("Should compute a calculation that only its record holds", func(
		ctx SpecContext,
	) {
		node := testutil.NewNode(ctx)
		iteratorSvc, channelSvc, calcSvc := openServices(ctx, node)
		index := channel.Channel{
			Name:     "brk_time",
			DataType: telem.TimestampT,
			IsIndex:  true,
		}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &index)
		})).To(Succeed())
		sensor := channel.Channel{
			Name:       "brk_sensor",
			DataType:   telem.Float32T,
			LocalIndex: index.LocalKey,
		}
		output := channel.Channel{
			Name:     "brk_output",
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			w := channelSvc.NewWriter(tx)
			if err := w.Create(ctx, &sensor); err != nil {
				return err
			}
			if err := w.Create(ctx, &output); err != nil {
				return err
			}
			return calcSvc.NewWriter(tx).Create(ctx, &calculation.Calculation{
				Output:     output.Key(),
				Expression: "return brk_sensor * 2",
			})
		})).To(Succeed())
		keys := channel.Keys{index.Key(), sensor.Key()}
		w := MustSucceed(node.Framer.OpenWriter(ctx, framer.WriterConfig{
			Start:            telem.SecondTS,
			Keys:             keys,
			EnableAutoCommit: new(true),
		}))
		Expect(w.Write(frame.NewMulti(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(1, 2, 3),
			telem.NewSeriesV[float32](1, 2, 3),
		}))).To(BeTrue())
		Expect(w.Close()).To(Succeed())
		iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
			Keys:   channel.Keys{output.Key()},
			Bounds: telem.TimeRangeMax,
		}))
		DeferCleanup(func() { Expect(iter.Close()).To(Succeed()) })
		Expect(iter.SeekFirst()).To(BeTrue())
		Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
		Expect(merged(iter.Value().Get(output.Key()).Series)).
			To(telem.MatchSeriesDataV[float32](2, 4, 6))
	})
})
