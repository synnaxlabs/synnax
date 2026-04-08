// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel/calculation/analyzer"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Analyze", func() {

	Describe("Type Inference", func() {
		It("Should infer the correct type for integer literal expressions", func(ctx SpecContext) {
			a := analyzer.New(symbol.MapResolver{})
			ch := channel.Channel{Name: "calc", Expression: "return 1 + 2"}
			res := MustSucceed(a.Analyze(ctx, ch))
			Expect(res.DataType).To(Equal(telem.Int64T))
		})

		It("Should infer the correct type for float literal expressions", func(ctx SpecContext) {
			a := analyzer.New(symbol.MapResolver{})
			ch := channel.Channel{Name: "calc", Expression: "return 1.0 + 2.0"}
			res := MustSucceed(a.Analyze(ctx, ch))
			Expect(res.DataType).To(Equal(telem.Float64T))
		})

		It("Should infer the correct type when referencing a float32 channel", func(ctx SpecContext) {
			r := symbol.MapResolver{
				"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10},
			}
			a := analyzer.New(r)
			ch := channel.Channel{Name: "calc", Expression: "return sensor * 2.0"}
			res := MustSucceed(a.Analyze(ctx, ch))
			Expect(res.DataType).To(Equal(telem.Float32T))
		})

		It("Should infer the correct type when referencing an int64 channel", func(ctx SpecContext) {
			r := symbol.MapResolver{
				"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.I64()), ID: 10},
			}
			a := analyzer.New(r)
			ch := channel.Channel{Name: "calc", Expression: "return sensor + 1"}
			res := MustSucceed(a.Analyze(ctx, ch))
			Expect(res.DataType).To(Equal(telem.Int64T))
		})

		It("Should infer the correct type when referencing multiple channels", func(ctx SpecContext) {
			r := symbol.MapResolver{
				"a": {Name: "a", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 10},
				"b": {Name: "b", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 20},
			}
			a := analyzer.New(r)
			ch := channel.Channel{Name: "calc", Expression: "return a + b"}
			res := MustSucceed(a.Analyze(ctx, ch))
			Expect(res.DataType).To(Equal(telem.Float64T))
		})
	})

	Describe("Deps", func() {
		It("Should return no deps for a pure literal expression", func(ctx SpecContext) {
			a := analyzer.New(symbol.MapResolver{})
			ch := channel.Channel{Name: "calc", Expression: "return 1 + 2"}
			res := MustSucceed(a.Analyze(ctx, ch))
			Expect(res.Deps).To(BeEmpty())
		})

		It("Should return the key of a single referenced channel", func(ctx SpecContext) {
			r := symbol.MapResolver{
				"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 10},
			}
			a := analyzer.New(r)
			ch := channel.Channel{Name: "calc", Expression: "return sensor * 2.0"}
			res := MustSucceed(a.Analyze(ctx, ch))
			Expect(res.Deps).To(ConsistOf(channel.Key(10)))
		})

		It("Should return keys for multiple referenced channels", func(ctx SpecContext) {
			r := symbol.MapResolver{
				"a": {Name: "a", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 10},
				"b": {Name: "b", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 20},
			}
			a := analyzer.New(r)
			ch := channel.Channel{Name: "calc", Expression: "return a + b"}
			res := MustSucceed(a.Analyze(ctx, ch))
			Expect(res.Deps).To(ConsistOf(channel.Key(10), channel.Key(20)))
		})

		It("Should not duplicate a channel referenced multiple times", func(ctx SpecContext) {
			r := symbol.MapResolver{
				"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 10},
			}
			a := analyzer.New(r)
			ch := channel.Channel{Name: "calc", Expression: "return sensor + sensor"}
			res := MustSucceed(a.Analyze(ctx, ch))
			Expect(res.Deps).To(ConsistOf(channel.Key(10)))
		})

		It("Should resolve deps from the temp cache for previously analyzed channels", func(ctx SpecContext) {
			a := analyzer.New(symbol.MapResolver{})
			first := channel.Channel{
				Name:        "first",
				Expression:  "return 1.0",
				Leaseholder: 1,
				LocalKey:    5,
			}
			MustSucceed(a.Analyze(ctx, first))
			second := channel.Channel{
				Name:       "second",
				Expression: "return first + 1.0",
			}
			res := MustSucceed(a.Analyze(ctx, second))
			Expect(res.Deps).To(ConsistOf(first.Key()))
		})
	})

	Describe("Channel Caching", func() {
		It("Should resolve a previously analyzed channel by name", func(ctx SpecContext) {
			a := analyzer.New(symbol.MapResolver{})
			sensor := channel.Channel{
				Name:        "sensor",
				Expression:  "return 1.0",
				Leaseholder: 1,
				LocalKey:    1,
			}
			MustSucceed(a.Analyze(ctx, sensor))
			calc := channel.Channel{
				Name:        "calc",
				Expression:  "return sensor + 1.0",
				Leaseholder: 1,
				LocalKey:    2,
			}
			res := MustSucceed(a.Analyze(ctx, calc))
			Expect(res.DataType).To(Equal(telem.Float64T))
		})

		It("Should cache multiple channels and resolve a chain of dependencies", func(ctx SpecContext) {
			a := analyzer.New(symbol.MapResolver{})
			first := channel.Channel{
				Name:        "first",
				Expression:  "return 1.0",
				Leaseholder: 1,
				LocalKey:    5,
			}
			MustSucceed(a.Analyze(ctx, first))
			second := channel.Channel{
				Name:        "second",
				Expression:  "return first + 1.0",
				Leaseholder: 1,
				LocalKey:    6,
			}
			MustSucceed(a.Analyze(ctx, second))
			third := channel.Channel{
				Name:        "third",
				Expression:  "return first + second",
				Leaseholder: 1,
				LocalKey:    7,
			}
			res := MustSucceed(a.Analyze(ctx, third))
			Expect(res.DataType).To(Equal(telem.Float64T))
		})

		It("Should still resolve by name when the channel has key 0", func(ctx SpecContext) {
			a := analyzer.New(symbol.MapResolver{})
			sensor := channel.Channel{
				Name:       "sensor",
				Expression: "return 1.0",
			}
			MustSucceed(a.Analyze(ctx, sensor))
			calc := channel.Channel{
				Name:       "calc",
				Expression: "return sensor + 1.0",
			}
			res := MustSucceed(a.Analyze(ctx, calc))
			Expect(res.DataType).To(Equal(telem.Float64T))
		})
	})

	Describe("Error Handling", func() {
		It("Should return an error for invalid syntax", func(ctx SpecContext) {
			a := analyzer.New(symbol.MapResolver{})
			ch := channel.Channel{Name: "calc", Expression: "return {{invalid"}
			Expect(a.Analyze(ctx, ch)).Error().To(MatchError(ContainSubstring("extraneous input")))
		})

		It("Should return an error for an undefined channel reference", func(ctx SpecContext) {
			a := analyzer.New(symbol.MapResolver{})
			ch := channel.Channel{Name: "calc", Expression: "return nonexistent + 1"}
			res, err := a.Analyze(ctx, ch)
			Expect(err).To(MatchError(ContainSubstring("undefined symbol")))
			Expect(res.Unresolved).To(ConsistOf("nonexistent"))
		})

		It("Should return zero Result on parse error", func(ctx SpecContext) {
			a := analyzer.New(symbol.MapResolver{})
			ch := channel.Channel{Name: "calc", Expression: "return {{invalid"}
			res, err := a.Analyze(ctx, ch)
			Expect(err).To(MatchError(ContainSubstring("extraneous input")))
			Expect(res).To(Equal(analyzer.Result{}))
		})

		It("Should return unresolved names on analysis error", func(ctx SpecContext) {
			a := analyzer.New(symbol.MapResolver{})
			ch := channel.Channel{Name: "calc", Expression: "return nonexistent + 1"}
			res, err := a.Analyze(ctx, ch)
			Expect(err).To(MatchError(ContainSubstring("undefined symbol")))
			Expect(res.DataType).To(Equal(telem.UnknownT))
			Expect(res.Unresolved).To(ConsistOf("nonexistent"))
		})
	})

	Describe("Derivative Operation Type Override", func() {
		It("Should infer float64 DataType when derivative is the last operation", func(ctx SpecContext) {
			r := symbol.MapResolver{
				"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 1},
			}
			a := analyzer.New(r)
			ch := channel.Channel{
				Name:       "deriv_calc",
				Expression: "return sensor",
				Operations: []channel.Operation{
					{Type: channel.OperationTypeDerivative},
				},
			}
			res := MustSucceed(a.Analyze(ctx, ch))
			Expect(res.DataType).To(Equal(telem.Float64T))
		})

		It("Should not override DataType when avg is the last operation", func(ctx SpecContext) {
			r := symbol.MapResolver{
				"sensor2": {Name: "sensor2", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 2},
			}
			a := analyzer.New(r)
			ch := channel.Channel{
				Name:       "avg_calc",
				Expression: "return sensor2",
				Operations: []channel.Operation{
					{Type: channel.OperationTypeAvg},
				},
			}
			res := MustSucceed(a.Analyze(ctx, ch))
			Expect(res.DataType).To(Equal(telem.Int32T))
		})
	})

	Describe("Resolver Fallback", func() {
		It("Should fall back to the underlying resolver for symbols not in the temp cache", func(ctx SpecContext) {
			r := symbol.MapResolver{
				"external": {Name: "external", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 100},
			}
			a := analyzer.New(r)
			ch := channel.Channel{Name: "calc", Expression: "return external * 2.0"}
			res := MustSucceed(a.Analyze(ctx, ch))
			Expect(res.DataType).To(Equal(telem.Float64T))
		})
	})
})
