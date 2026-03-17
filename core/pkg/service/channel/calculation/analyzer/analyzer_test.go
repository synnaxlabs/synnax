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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel/calculation/analyzer"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var ctx = context.Background()

var _ = Describe("Analyze", func() {

	Describe("Type Inference", func() {
		It("Should infer the correct type for integer literal expressions", func() {
			a := analyzer.New(symbol.MapResolver{})
			ch := channel.Channel{Name: "calc", Expression: "return 1 + 2"}
			dt := MustSucceed(a.Analyze(ctx, ch))
			Expect(dt).To(Equal(telem.Int64T))
		})

		It("Should infer the correct type for float literal expressions", func() {
			a := analyzer.New(symbol.MapResolver{})
			ch := channel.Channel{Name: "calc", Expression: "return 1.0 + 2.0"}
			dt := MustSucceed(a.Analyze(ctx, ch))
			Expect(dt).To(Equal(telem.Float64T))
		})

		It("Should infer the correct type when referencing a float32 channel", func() {
			r := symbol.MapResolver{
				"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10},
			}
			a := analyzer.New(r)
			ch := channel.Channel{Name: "calc", Expression: "return sensor * 2.0"}
			dt := MustSucceed(a.Analyze(ctx, ch))
			Expect(dt).To(Equal(telem.Float32T))
		})

		It("Should infer the correct type when referencing an int64 channel", func() {
			r := symbol.MapResolver{
				"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.I64()), ID: 10},
			}
			a := analyzer.New(r)
			ch := channel.Channel{Name: "calc", Expression: "return sensor + 1"}
			dt := MustSucceed(a.Analyze(ctx, ch))
			Expect(dt).To(Equal(telem.Int64T))
		})

		It("Should infer the correct type when referencing multiple channels", func() {
			r := symbol.MapResolver{
				"a": {Name: "a", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 10},
				"b": {Name: "b", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 20},
			}
			a := analyzer.New(r)
			ch := channel.Channel{Name: "calc", Expression: "return a + b"}
			dt := MustSucceed(a.Analyze(ctx, ch))
			Expect(dt).To(Equal(telem.Float64T))
		})
	})

	Describe("Channel Caching", func() {
		It("Should resolve a previously analyzed channel by name", func() {
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
			dt := MustSucceed(a.Analyze(ctx, calc))
			Expect(dt).To(Equal(telem.Float64T))
		})

		It("Should cache multiple channels and resolve a chain of dependencies", func() {
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
			dt := MustSucceed(a.Analyze(ctx, third))
			Expect(dt).To(Equal(telem.Float64T))
		})

		It("Should still resolve by name when the channel has key 0", func() {
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
			dt := MustSucceed(a.Analyze(ctx, calc))
			Expect(dt).To(Equal(telem.Float64T))
		})
	})

	Describe("Error Handling", func() {
		It("Should return an error for invalid syntax", func() {
			a := analyzer.New(symbol.MapResolver{})
			ch := channel.Channel{Name: "calc", Expression: "return {{invalid"}
			Expect(a.Analyze(ctx, ch)).Error().To(MatchError(ContainSubstring("extraneous input")))
		})

		It("Should return an error for an undefined channel reference", func() {
			a := analyzer.New(symbol.MapResolver{})
			ch := channel.Channel{Name: "calc", Expression: "return nonexistent + 1"}
			Expect(a.Analyze(ctx, ch)).Error().To(HaveOccurred())
		})

		It("Should return telem.UnknownT on parse error", func() {
			a := analyzer.New(symbol.MapResolver{})
			ch := channel.Channel{Name: "calc", Expression: "return {{invalid"}
			dt, err := a.Analyze(ctx, ch)
			Expect(err).To(MatchError(ContainSubstring("extraneous input")))
			Expect(dt).To(Equal(telem.UnknownT))
		})

		It("Should return telem.UnknownT on analysis error", func() {
			a := analyzer.New(symbol.MapResolver{})
			ch := channel.Channel{Name: "calc", Expression: "return nonexistent + 1"}
			dt, err := a.Analyze(ctx, ch)
			Expect(err).To(HaveOccurred())
			Expect(dt).To(Equal(telem.UnknownT))
		})
	})

	Describe("Resolver Fallback", func() {
		It("Should fall back to the underlying resolver for symbols not in the temp cache", func() {
			r := symbol.MapResolver{
				"external": {Name: "external", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 100},
			}
			a := analyzer.New(r)
			ch := channel.Channel{Name: "calc", Expression: "return external * 2.0"}
			dt := MustSucceed(a.Analyze(ctx, ch))
			Expect(dt).To(Equal(telem.Float64T))
		})
	})
})
