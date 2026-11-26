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

var _ = Describe("Delete by a kept stored name", func() {
	It("Should delete the data of a channel by the exact name that it holds", func(
		ctx SpecContext,
	) {
		index := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.TimestampT,
			IsIndex:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).Create(ctx, &index)
		})).To(Succeed())
		// A v0.62 Core with name validation off stored this name as sent, not in NFC.
		// The upgrade keeps it.
		stored := index
		stored.Name = index.Name + "_é"
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, stored)).
			To(Succeed())
		keys := channel.Keys{index.Key()}
		w := MustSucceed(framerSvc.OpenWriter(ctx, svcframer.WriterConfig{
			Start: telem.SecondTS,
			Keys:  keys,
		}))
		Expect(w.Write(frame.NewMulti(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(1, 2, 3),
		}))).To(BeTrue())
		Expect(w.Close()).To(Succeed())
		Expect(apiSvc.Delete(rootCtx(ctx), db, apiframer.DeleteRequest{
			Names:  []string{stored.Name},
			Bounds: telem.TimeRangeMax,
		})).Error().ToNot(HaveOccurred())
		iter := MustSucceed(framerSvc.OpenIterator(ctx, svcframer.IteratorConfig{
			Keys:   keys,
			Bounds: telem.TimeRangeMax,
		}))
		Expect(iter.SeekFirst()).To(BeFalse())
		Expect(iter.Close()).To(Succeed())
	})
})
