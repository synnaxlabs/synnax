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

var _ = Describe("Delete by keys and names", func() {
	// written creates an index channel that holds data and returns it with a check of
	// whether it still holds data.
	written := func(ctx SpecContext) (channel.Channel, func() bool) {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.TimestampT,
			IsIndex:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		keys := channel.Keys{ch.Key()}
		w := MustSucceed(framerSvc.OpenWriter(ctx, svcframer.WriterConfig{
			Start: telem.SecondTS,
			Keys:  keys,
		}))
		Expect(w.Write(frame.NewMulti(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(1, 2, 3),
		}))).To(BeTrue())
		Expect(w.Close()).To(Succeed())
		return ch, func() bool {
			GinkgoHelper()
			iter := MustOpen(framerSvc.OpenIterator(ctx, svcframer.IteratorConfig{
				Keys:   keys,
				Bounds: telem.TimeRangeMax,
			}))
			defer func() { Expect(iter.Close()).To(Succeed()) }()
			return iter.SeekFirst()
		}
	}

	It("Should delete the data of every channel that the keys or the names hold", func(
		ctx SpecContext,
	) {
		byKey, keyHasData := written(ctx)
		byName, nameHasData := written(ctx)
		Expect(apiSvc.Delete(rootCtx(ctx), db, apiframer.DeleteRequest{
			Keys:   channel.Keys{byKey.Key()},
			Names:  []string{byName.Name},
			Bounds: telem.TimeRangeMax,
		})).Error().ToNot(HaveOccurred())
		Expect(keyHasData()).To(BeFalse())
		Expect(nameHasData()).To(BeFalse())
	})

	It("Should delete a channel that a key and a name both give, and no other", func(
		ctx SpecContext,
	) {
		both, bothHasData := written(ctx)
		byName, nameHasData := written(ctx)
		_, otherHasData := written(ctx)
		Expect(apiSvc.Delete(rootCtx(ctx), db, apiframer.DeleteRequest{
			Keys:   channel.Keys{both.Key()},
			Names:  []string{both.Name, byName.Name},
			Bounds: telem.TimeRangeMax,
		})).Error().ToNot(HaveOccurred())
		Expect(bothHasData()).To(BeFalse())
		Expect(nameHasData()).To(BeFalse())
		Expect(otherHasData()).To(BeTrue())
	})

	It("Should delete nothing when no key or name gives a channel", func(
		ctx SpecContext,
	) {
		_, otherHasData := written(ctx)
		Expect(apiSvc.Delete(rootCtx(ctx), db, apiframer.DeleteRequest{
			Bounds: telem.TimeRangeMax,
		})).Error().ToNot(HaveOccurred())
		Expect(apiSvc.Delete(rootCtx(ctx), db, apiframer.DeleteRequest{
			Names:  []string{UniqueChannelName()},
			Bounds: telem.TimeRangeMax,
		})).Error().ToNot(HaveOccurred())
		Expect(otherHasData()).To(BeTrue())
	})

	It("Should delete the data of the keys when a name beside them holds no channel",
		func(ctx SpecContext) {
			byKey, keyHasData := written(ctx)
			Expect(apiSvc.Delete(rootCtx(ctx), db, apiframer.DeleteRequest{
				Keys:   channel.Keys{byKey.Key()},
				Names:  []string{UniqueChannelName()},
				Bounds: telem.TimeRangeMax,
			})).Error().ToNot(HaveOccurred())
			Expect(keyHasData()).To(BeFalse())
		})
})
