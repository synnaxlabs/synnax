// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Untimed read keys", func() {
	var index, data channel.Channel
	BeforeEach(func(ctx SpecContext) {
		index = channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.TimestampT,
			IsIndex:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).Create(ctx, &index)
		})).To(Succeed())
		data = channel.Channel{
			Name:       UniqueChannelName(),
			DataType:   telem.Float32T,
			LocalIndex: index.LocalKey,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).Create(ctx, &data)
		})).To(Succeed())
		keys := channel.Keys{index.Key(), data.Key()}
		w := MustSucceed(framerSvc.OpenWriter(ctx, svcframer.WriterConfig{
			Start: telem.SecondTS,
			Keys:  keys,
		}))
		Expect(w.Write(frame.NewMulti(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(1, 2),
			telem.NewSeriesV[float32](1, 2),
		}))).To(BeTrue())
		Expect(w.Close()).To(Succeed())
	})

	It("Should fail a read that marks untimed a key it does not request", func(
		ctx SpecContext,
	) {
		req := apiframer.ReadRequest{
			Keys:        channel.Keys{data.Key()},
			UntimedKeys: channel.Keys{index.Key()},
			TimeRanges:  []telem.TimeRange{telem.TimeRangeMax},
		}
		res, err := apiSvc.Read(rootCtx(ctx), req)
		if err == nil {
			var iter *svcframer.Iterator
			iter, err = res.OpenIterator(rootCtx(ctx), req.Keys, res.TimeRanges[0], nil)
			if err == nil {
				Expect(iter.Close()).To(Succeed())
			}
		}
		Expect(err).To(MatchError(ContainSubstring(
			fmt.Sprintf("channel %d is not requested", index.Key()),
		)))
	})
})
