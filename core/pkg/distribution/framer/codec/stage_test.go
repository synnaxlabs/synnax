// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package codec_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Stage", func() {
	resolver := mapResolver{1: telem.Float32T, 2: telem.Float32T, 3: telem.Float32T}

	It("Should encode under each staged state from its Apply on, in stage order", func(
		ctx SpecContext,
	) {
		enc, dec := codec.NewDynamic(resolver), codec.NewDynamic(resolver)
		Expect(enc.Update(ctx, channel.Keys{1})).To(Succeed())
		Expect(enc.Stage(ctx, channel.Keys{1, 2})).To(Succeed())
		Expect(enc.Stage(ctx, channel.Keys{1, 2, 3})).To(Succeed())
		for _, keys := range []channel.Keys{{1}, {1, 2}, {1, 2, 3}} {
			Expect(dec.Update(ctx, keys)).To(Succeed())
		}
		all := []keyed{
			{1, telem.NewSeriesV[float32](1)},
			{2, telem.NewSeriesV[float32](2)},
			{3, telem.NewSeriesV[float32](3)},
		}
		roundTrip := func() framer.Frame {
			GinkgoHelper()
			return MustSucceed(dec.Decode(MustSucceed(enc.Encode(ctx, multi(all...)))))
		}
		expectSeries(roundTrip(), all[:1]...)
		Expect(enc.Apply()).To(Succeed())
		expectSeries(roundTrip(), all[:2]...)
		Expect(enc.Apply()).To(Succeed())
		expectSeries(roundTrip(), all...)
	})

	It("Should fail an Apply that no stage holds", func(ctx SpecContext) {
		c := codec.NewDynamic(resolver)
		Expect(c.Update(ctx, channel.Keys{1})).To(Succeed())
		Expect(
			c.Apply(),
		).To(MatchError("framer codec: a request applied that no stage holds"))
	})
})
