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
	floats := []TableEntry{
		Entry("f32 NaN", math.NaN(), types.F32()),
		Entry("f32 +Inf", math.Inf(1), types.F32()),
		Entry("f64 NaN", math.NaN(), types.F64()),
		Entry("f64 +Inf", math.Inf(1), types.F64()),
		Entry("f64 -Inf", math.Inf(-1), types.F64()),
		Entry("f64 finite", 1.5, types.F64()),
	}

	expectFloat := func(p types.Param, value float64, t types.Type) {
		GinkgoHelper()
		var got float64
		switch v := types.DecodeValue(p.Value, t).(type) {
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

	DescribeTable("Should round trip a float through msgpack",
		func(ctx SpecContext, value float64, t types.Type) {
			param := types.Param{Name: "a", Type: t, Value: types.EncodeFloat(value)}
			var decoded types.Param
			Expect(msgpack.Codec.Decode(
				MustSucceed(msgpack.Codec.Encode(param)), &decoded,
			)).To(Succeed())
			expectFloat(decoded, value, t)
		},
		floats,
	)

	DescribeTable("Should round trip a float through proto",
		func(value float64, t types.Type) {
			param := types.Param{Name: "a", Type: t, Value: types.EncodeFloat(value)}
			decoded := MustSucceed(pb.ParamFromPB(MustSucceed(pb.ParamToPB(param))))
			expectFloat(decoded, value, t)
		},
		floats,
	)

	It("Should keep the exact encoding of a finite value", func() {
		Expect(types.EncodeFloat(1.5)).To(Equal(1.5))
		Expect(types.DecodeValue(1.5, types.F64())).To(Equal(1.5))
	})

	It("Should keep other text in a float param", func() {
		Expect(types.DecodeValue("text", types.F64())).To(Equal("text"))
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
