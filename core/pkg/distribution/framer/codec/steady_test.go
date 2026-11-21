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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Steady state", func() {
	// declaring returns a codec that requests 6 and has declared clock 5.
	declaring := func(ctx SpecContext) *codec.Codec {
		GinkgoHelper()
		c := codec.NewStatic(channel.Keys{6}, []telem.DataType{telem.Float32T})
		MustSucceed(c.Encode(ctx, multi(
			keyed{5, timestamps(5, 1)},
			keyed{6, clocked(telem.NewSeriesV[float32](1), 5)},
		)))
		return c
	}

	It("Should add no bytes to a closed frame, which carries each clock", func(
		ctx SpecContext,
	) {
		closed := multi(
			keyed{5, timestamps(5, 2)},
			keyed{6, clocked(telem.NewSeriesV[float32](2), 5)},
		)
		requested := codec.NewStatic(
			channel.Keys{5, 6},
			[]telem.DataType{telem.TimestampT, telem.Float32T},
		)
		MustSucceed(requested.Encode(ctx, multi(
			keyed{5, timestamps(5, 1)},
			keyed{6, clocked(telem.NewSeriesV[float32](1), 5)},
		)))
		want := MustSucceed(requested.Encode(ctx, closed))
		Expect(MustSucceed(declaring(ctx).Encode(ctx, closed))).To(Equal(want))
	})

	It("Should key each series of a frame that leaves out a declared clock", func(
		ctx SpecContext,
	) {
		open := multi(keyed{6, clocked(telem.NewSeriesV[float32](2), 5)})
		Expect(MustSucceed(declaring(ctx).Encode(ctx, open))).
			To(Equal(frameBytes("3E 01000000 01000000 06000000 00000040")))
	})
})
