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
	It("Should add no bytes to a frame that changes no agreed value", func(
		ctx SpecContext,
	) {
		newCodec := func() *codec.Codec {
			return codec.NewStatic(channel.Keys{6}, []telem.DataType{telem.Float32T})
		}
		data := keyed{6, clocked(telem.NewSeriesV[float32](2), 5)}
		baseline := newCodec()
		MustSucceed(baseline.Encode(ctx, multi(keyed{6, clocked(
			telem.NewSeriesV[float32](1), 5,
		)})))
		want := MustSucceed(baseline.Encode(ctx, multi(data)))
		Expect(want).To(Equal(frameBytes("3F 01000000 01000000 00000040")))

		enc := newCodec()
		MustSucceed(enc.Encode(ctx, multi(
			keyed{5, timestamps(5, 1)},
			keyed{6, clocked(telem.NewSeriesV[float32](1), 5)},
		)))
		Expect(MustSucceed(enc.Encode(ctx, multi(data)))).To(Equal(want))
	})
})
