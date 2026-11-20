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

var _ = Describe("Delete checks", func() {
	It("Should delete data by the decomposed form of a channel's name", func(
		ctx SpecContext,
	) {
		// The Core stores names in NFC; a client may send the decomposed form.
		stem := UniqueChannelName()
		index := channel.Channel{
			Name:     stem + "_ñ_time",
			DataType: telem.TimestampT,
			IsIndex:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).Create(ctx, &index)
		})).To(Succeed())
		data := channel.Channel{
			Name:       stem + "_ñ",
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
			telem.NewSeriesSecondsTSV(1, 2, 3),
			telem.NewSeriesV[float32](1, 2, 3),
		}))).To(BeTrue())
		Expect(w.Close()).To(Succeed())
		Expect(apiSvc.Delete(rootCtx(ctx), db, apiframer.DeleteRequest{
			Names:  []string{stem + "_ñ_time", stem + "_ñ"},
			Bounds: telem.TimeRangeMax,
		})).Error().ToNot(HaveOccurred())
		iter := MustOpen(framerSvc.OpenIterator(ctx, svcframer.IteratorConfig{
			Keys:   keys,
			Bounds: telem.TimeRangeMax,
		}))
		Expect(iter.SeekFirst()).To(BeFalse())
	})
})
