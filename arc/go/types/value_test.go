// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/arc/types/pb"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Value", func() {
	DescribeTable("Should round trip a float through proto and msgpack",
		func(ctx SpecContext, value float64, t types.Type) {
			param := types.Param{Name: "a", Type: t, Value: types.EncodeFloat(value)}
			codec := msgpack.Codec
			var fromMsgpack types.Param
			Expect(codec.Decode(
				ctx, MustSucceed(codec.Encode(ctx, param)), &fromMsgpack,
			)).To(Succeed())
			fromProto := MustSucceed(pb.ParamFromPB(MustSucceed(pb.ParamToPB(param))))
			for _, p := range []types.Param{fromMsgpack, fromProto} {
				decoded := types.DecodeValue(p.Value, t)
				var got float64
				switch v := decoded.(type) {
				case float32:
					Expect(t.Kind).To(Equal(types.KindF32))
					got = float64(v)
				case float64:
					got = v
				default:
					Fail("value did not decode to a float")
				}
				if math.IsNaN(value) {
					Expect(math.IsNaN(got)).To(BeTrue())
				} else {
					Expect(got).To(Equal(value))
				}
			}
		},
		Entry("f32 NaN", math.NaN(), types.F32()),
		Entry("f32 +Inf", math.Inf(1), types.F32()),
		Entry("f64 NaN", math.NaN(), types.F64()),
		Entry("f64 +Inf", math.Inf(1), types.F64()),
		Entry("f64 -Inf", math.Inf(-1), types.F64()),
		Entry("f64 finite", 1.5, types.F64()),
	)

	It("Should keep the exact encoding of a finite value", func() {
		Expect(types.EncodeFloat(1.5)).To(Equal(1.5))
		Expect(types.DecodeValue(1.5, types.F64())).To(Equal(1.5))
	})

	It("Should keep NaN text in a string param", func() {
		Expect(types.DecodeValue("NaN", types.String())).To(Equal("NaN"))
		Expect(types.DecodeValue("NaN", types.VarRef(types.String(), "n"))).To(
			Equal("NaN"),
		)
	})

	It("Should decode through a variable reference", func() {
		v := types.DecodeValue("Infinity", types.VarRef(types.F32(), "n"))
		Expect(v).To(Equal(float32(math.Inf(1))))
	})
})
