// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/arc/types/versions/v0"
	"github.com/synnaxlabs/x/encoding/orc"
)

var _ = Describe("Codec", func() {
	Describe("Type", func() {
		DescribeTable("should round-trip encode and decode",
			func(original v0.Type) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v0.Type
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("scalar", v0.Type{Kind: v0.KindF64, Name: "f64"}),
			Entry("channel with direction", v0.Type{
				Kind:          v0.KindChan,
				Name:          "chan",
				Elem:          &v0.Type{Kind: v0.KindF32},
				ChanDirection: v0.ChanDirectionRead,
			}),
			Entry("with unit", v0.Type{
				Kind: v0.KindF64,
				Name: "f64",
				Unit: new(v0.Unit{
					Dimensions: v0.Dimensions{Length: 1, Time: -1},
					Scale:      1,
					Name:       "m/s",
				}),
			}),
			Entry("variable with constraint", v0.Type{
				Kind:       v0.KindVariable,
				Name:       "T",
				Constraint: &v0.Type{Kind: v0.KindNumericConstant},
			}),
			Entry("function with properties", v0.Type{
				FunctionProperties: v0.FunctionProperties{
					Inputs:  v0.Params{{Name: "in", Type: v0.Type{Kind: v0.KindI32}}},
					Outputs: v0.Params{{Name: "out", Type: v0.Type{Kind: v0.KindF64}}},
					Config:  v0.Params{{Name: "rate", Type: v0.Type{Kind: v0.KindF32}}},
				},
				Kind: v0.KindFunction,
			}),
			Entry("zero value", v0.Type{}),
		)
	})

	Describe("Param", func() {
		DescribeTable("should round-trip encode and decode",
			func(original v0.Param) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v0.Param
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("fully populated", v0.Param{
				Name:  "rate",
				Type:  v0.Type{Kind: v0.KindF64, Name: "f64"},
				Value: map[string]any{"key": "value"},
			}),
			Entry("string value", v0.Param{
				Name:  "label",
				Type:  v0.Type{Kind: v0.KindString},
				Value: "hello",
			}),
			Entry("zero value", v0.Param{}),
		)
	})

	Describe("FunctionProperties", func() {
		DescribeTable("should round-trip encode and decode",
			func(original v0.FunctionProperties) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v0.FunctionProperties
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("fully populated", v0.FunctionProperties{
				Inputs:  v0.Params{{Name: "in", Type: v0.Type{Kind: v0.KindI32}}},
				Outputs: v0.Params{{Name: "out", Type: v0.Type{Kind: v0.KindF64}}},
				Config:  v0.Params{{Name: "rate", Type: v0.Type{Kind: v0.KindF32}}},
			}),
			Entry("zero values", v0.FunctionProperties{}),
		)
	})

	Describe("Unit", func() {
		DescribeTable("should round-trip encode and decode",
			func(original v0.Unit) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v0.Unit
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("fully populated", v0.Unit{
				Dimensions: v0.Dimensions{Mass: 1, Length: -1, Time: -2},
				Scale:      6894.76,
				Name:       "psi",
			}),
			Entry("zero value", v0.Unit{}),
		)
	})

	Describe("Dimensions", func() {
		DescribeTable("should round-trip encode and decode",
			func(original v0.Dimensions) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v0.Dimensions
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("fully populated", v0.Dimensions{
				Length:      1,
				Mass:        2,
				Time:        -3,
				Current:     4,
				Temperature: -5,
				Angle:       6,
				Count:       -7,
				Data:        8,
			}),
			Entry("zero value", v0.Dimensions{}),
		)
	})

	Describe("Channels", func() {
		DescribeTable(
			"should round-trip encode and decode",
			func(original v0.Channels) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v0.Channels
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("fully populated", v0.Channels{
				Read:  map[uint32]string{2: "test_1"},
				Write: map[uint32]string{3: "test_2"},
			}),
			Entry("zero values", v0.Channels{Read: nil, Write: nil}),
			Entry(
				"empty collections",
				v0.Channels{Read: map[uint32]string{}, Write: map[uint32]string{}},
			),
		)
	})
})
