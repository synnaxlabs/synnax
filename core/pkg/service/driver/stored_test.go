// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("LastStored", func() {
	var index channel.Channel
	BeforeEach(func(ctx SpecContext) {
		index = channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.TimestampT,
			IsIndex:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &index)
		})).To(Succeed())
	})

	write := func(ctx SpecContext, seconds ...telem.TimeStamp) {
		GinkgoHelper()
		w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
			Keys:  channel.Keys{index.Key()},
			Start: seconds[0] * telem.SecondTS,
		}))
		Expect(w.Write(frame.NewMulti(
			channel.Keys{index.Key()},
			[]telem.Series{telem.NewSeriesSecondsTSV(seconds...)},
		))).To(BeTrue())
		Expect(w.Close()).To(Succeed())
	}

	It("Should return zero for an index with no data", func(ctx SpecContext) {
		Expect(driver.LastStored(ctx, framerSvc, index.Key())).
			To(Equal(telem.TimeStamp(0)))
	})

	It("Should return the last timestamp of the last write", func(ctx SpecContext) {
		write(ctx, 1, 2, 3)
		write(ctx, 20, 21, 22)
		Expect(driver.LastStored(ctx, framerSvc, index.Key())).
			To(Equal(22 * telem.SecondTS))
	})
})
