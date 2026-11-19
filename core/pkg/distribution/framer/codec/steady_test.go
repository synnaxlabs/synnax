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
	// declared returns a codec whose state declared clock 5 for key 6 in its first
	// frame.
	declared := func(ctx SpecContext) *codec.Codec {
		enc := codec.NewStatic(channel.Keys{6}, []telem.DataType{telem.Float32T})
		MustSucceed(enc.Encode(ctx, multi(
			keyed{5, timestamps(5, 1)},
			keyed{6, clocked(telem.NewSeriesV[float32](1), 5)},
		)))
		return enc
	}

	It("Should add no bytes to a frame that carries its clock with its data", func(
		ctx SpecContext,
	) {
		enc := declared(ctx)
		closed := MustSucceed(enc.Encode(ctx, multi(
			keyed{5, timestamps(5, 1)},
			keyed{6, clocked(telem.NewSeriesV[float32](2), 5)},
		)))
		// Flags, the sequence number, the shared length, and the data of the two
		// series: no extension byte and no key.
		Expect(closed).To(HaveLen(1 + 4 + 4 + 8 + 4))
		Expect(closed[0] & 0x80).To(BeZero())
	})

	It("Should key each series of a frame that leaves out a declared clock", func(
		ctx SpecContext,
	) {
		enc := declared(ctx)
		open := MustSucceed(enc.Encode(ctx, multi(
			keyed{6, clocked(telem.NewSeriesV[float32](2), 5)},
		)))
		Expect(open).To(Equal(frameBytes("3E 01000000 01000000 06000000 00000040")))
	})
})
