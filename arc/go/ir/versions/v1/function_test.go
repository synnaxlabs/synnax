// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/arc/ir/versions/v1"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Function", func() {
	Describe("Type", func() {
		It("Should return function type with all properties", func() {
			inputs := types.Params{
				{Name: "x", Type: types.I64()},
				{Name: "y", Type: types.I64()},
			}
			outputs := types.Params{{Name: "output", Type: types.I64()}}

			fn := v1.Function{
				Key:     "test",
				Inputs:  inputs,
				Outputs: outputs,
			}

			t := fn.Type()
			Expect(t.Kind).To(Equal(types.KindFunction))
			Expect(t.Inputs).To(HaveLen(2))
			Expect(t.Outputs).To(HaveLen(1))
		})
	})

	Describe("StringWithPrefix", func() {
		DescribeTable(
			"Rendering",
			func(fn v1.Function, expected string) {
				Expect(fn.StringWithPrefix("")).To(Equal(expected))
			},
			Entry("no inputs, outputs, or channels",
				v1.Function{Key: "add"},
				"add\n└── channels: (none)\n"),
			Entry("inputs without outputs",
				v1.Function{
					Key:    "add",
					Inputs: types.Params{{Name: "x", Type: types.I64()}},
				},
				"add\n├── channels: (none)\n└── inputs: x (i64)\n"),
			Entry("inputs, outputs, and channels",
				v1.Function{
					Key: "add",
					Inputs: types.Params{
						{Name: "x", Type: types.I64()},
						{Name: "y", Type: types.I64()},
					},
					Outputs: types.Params{{Name: "output", Type: types.I64()}},
					Channels: types.Channels{
						Read:  map[uint32]string{1: "sensor"},
						Write: map[uint32]string{2: "valve"},
					},
				},
				"add\n"+
					"├── channels: read [1: sensor], write [2: valve]\n"+
					"├── inputs: x (i64), y (i64)\n"+
					"└── outputs: output (i64)\n"),
		)
	})

	Describe("DecodeMsgpack", func() {
		It("Should decode legacy uppercase Go field names", func() {
			legacy := struct {
				Key      string
				Body     v1.Body
				Config   types.Params
				Inputs   types.Params
				Outputs  types.Params
				Channels types.Channels
			}{
				Key:  "fn1",
				Body: v1.Body{Raw: "return 1"},
				Inputs: types.Params{
					{Name: "x", Type: types.Type{Kind: types.KindF64}},
				},
				Channels: types.Channels{
					Read:  map[uint32]string{1: "sensor"},
					Write: map[uint32]string{2: "output"},
				},
			}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded v1.Function
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Key).To(Equal("fn1"))
			Expect(decoded.Body.Raw).To(Equal("return 1"))
			Expect(decoded.Inputs).To(HaveLen(1))
			Expect(decoded.Channels.Read).To(HaveLen(1))
		})
	})
})
