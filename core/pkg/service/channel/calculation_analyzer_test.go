// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// failingResolver fails every lookup with context.Canceled.
type failingResolver struct{}

func (failingResolver) Resolve(context.Context, string) (*symbol.Symbol, error) {
	return nil, context.Canceled
}

func (failingResolver) Search(context.Context, string) ([]*symbol.Symbol, error) {
	return nil, context.Canceled
}

var _ = Describe("Analyze", func() {
	It("Should return an error of the resolver", func(ctx SpecContext) {
		a := channel.NewCalculationAnalyzer(failingResolver{})
		res, err := a.Analyze(ctx, channel.Channel{
			Name:       "calc",
			Expression: "return sensor + 1",
		})
		Expect(err).To(MatchError(context.Canceled))
		Expect(res.Unresolved).To(BeEmpty())
	})

	Describe("Type Inference", func() {
		It(
			"Should infer the correct type for integer literal expressions",
			func(ctx SpecContext) {
				a := channel.NewCalculationAnalyzer(StaticResolver{})
				ch := channel.Channel{Name: "calc", Expression: "return 1 + 2"}
				res := MustSucceed(a.Analyze(ctx, ch))
				Expect(res.ChanDataType).To(Equal(telem.Int64T))
			},
		)

		It(
			"Should infer the correct type for float literal expressions",
			func(ctx SpecContext) {
				a := channel.NewCalculationAnalyzer(StaticResolver{})
				ch := channel.Channel{Name: "calc", Expression: "return 1.0 + 2.0"}
				res := MustSucceed(a.Analyze(ctx, ch))
				Expect(res.ChanDataType).To(Equal(telem.Float64T))
			},
		)

		It(
			"Should infer the correct type when referencing a float32 channel",
			func(ctx SpecContext) {
				r := StaticResolver{
					{
						Name: "sensor",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F32()),
						ID:   10,
					},
				}
				a := channel.NewCalculationAnalyzer(r)
				ch := channel.Channel{Name: "calc", Expression: "return sensor * 2.0"}
				res := MustSucceed(a.Analyze(ctx, ch))
				Expect(res.ChanDataType).To(Equal(telem.Float32T))
			},
		)

		It(
			"Should infer the correct type when referencing an int64 channel",
			func(ctx SpecContext) {
				r := StaticResolver{
					{
						Name: "sensor",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I64()),
						ID:   10,
					},
				}
				a := channel.NewCalculationAnalyzer(r)
				ch := channel.Channel{Name: "calc", Expression: "return sensor + 1"}
				res := MustSucceed(a.Analyze(ctx, ch))
				Expect(res.ChanDataType).To(Equal(telem.Int64T))
			},
		)

		It(
			"Should infer the correct type when referencing multiple channels",
			func(ctx SpecContext) {
				r := StaticResolver{
					{
						Name: "a",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   10,
					},
					{
						Name: "b",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   20,
					},
				}
				a := channel.NewCalculationAnalyzer(r)
				ch := channel.Channel{Name: "calc", Expression: "return a + b"}
				res := MustSucceed(a.Analyze(ctx, ch))
				Expect(res.ChanDataType).To(Equal(telem.Float64T))
			},
		)

		It(
			"resolves a channel name containing dashes when enabled",
			func(ctx SpecContext) {
				r := StaticResolver{
					{
						Name: "ox-pt-1",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F32()),
						ID:   10,
					},
				}
				a := channel.NewCalculationAnalyzer(
					r,
					parser.Config{AllowDashedNames: true},
				)
				ch := channel.Channel{Name: "calc", Expression: "return ox-pt-1 * 2.0"}
				res := MustSucceed(a.Analyze(ctx, ch))
				Expect(res.ChanDataType).To(Equal(telem.Float32T))
			},
		)
	})

	Describe("Deps", func() {
		It(
			"Should return no deps for a pure literal expression",
			func(ctx SpecContext) {
				a := channel.NewCalculationAnalyzer(StaticResolver{})
				ch := channel.Channel{Name: "calc", Expression: "return 1 + 2"}
				res := MustSucceed(a.Analyze(ctx, ch))
				Expect(res.Deps).To(BeEmpty())
			},
		)

		It(
			"Should return the key of a single referenced channel",
			func(ctx SpecContext) {
				r := StaticResolver{
					{
						Name: "sensor",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   10,
					},
				}
				a := channel.NewCalculationAnalyzer(r)
				ch := channel.Channel{Name: "calc", Expression: "return sensor * 2.0"}
				res := MustSucceed(a.Analyze(ctx, ch))
				Expect(res.Deps).To(ConsistOf(channel.Key(10)))
			},
		)

		It(
			"Should return keys for multiple referenced channels",
			func(ctx SpecContext) {
				r := StaticResolver{
					{
						Name: "a",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   10,
					},
					{
						Name: "b",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   20,
					},
				}
				a := channel.NewCalculationAnalyzer(r)
				ch := channel.Channel{Name: "calc", Expression: "return a + b"}
				res := MustSucceed(a.Analyze(ctx, ch))
				Expect(res.Deps).To(ConsistOf(channel.Key(10), channel.Key(20)))
			},
		)

		It(
			"Should not duplicate a channel referenced multiple times",
			func(ctx SpecContext) {
				r := StaticResolver{
					{
						Name: "sensor",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   10,
					},
				}
				a := channel.NewCalculationAnalyzer(r)
				ch := channel.Channel{
					Name:       "calc",
					Expression: "return sensor + sensor",
				}
				res := MustSucceed(a.Analyze(ctx, ch))
				Expect(res.Deps).To(ConsistOf(channel.Key(10)))
			},
		)

		It(
			"Should resolve deps from the temp cache for previously analyzed channels",
			func(ctx SpecContext) {
				a := channel.NewCalculationAnalyzer(StaticResolver{})
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
			},
		)
	})

	Describe("Channel Caching", func() {
		It(
			"Should resolve a previously analyzed channel by name",
			func(ctx SpecContext) {
				a := channel.NewCalculationAnalyzer(StaticResolver{})
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
				Expect(res.ChanDataType).To(Equal(telem.Float64T))
			},
		)

		It(
			"Should cache multiple channels and resolve a chain of dependencies",
			func(ctx SpecContext) {
				a := channel.NewCalculationAnalyzer(StaticResolver{})
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
				Expect(res.ChanDataType).To(Equal(telem.Float64T))
			},
		)

		It(
			"Should still resolve by name when the channel has key 0",
			func(ctx SpecContext) {
				a := channel.NewCalculationAnalyzer(StaticResolver{})
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
				Expect(res.ChanDataType).To(Equal(telem.Float64T))
			},
		)
	})

	Describe("Error Handling", func() {
		It("Should return an error for invalid syntax", func(ctx SpecContext) {
			a := channel.NewCalculationAnalyzer(StaticResolver{})
			ch := channel.Channel{Name: "calc", Expression: "return {{invalid"}
			Expect(
				a.Analyze(ctx, ch),
			).Error().
				To(MatchError(ContainSubstring("extraneous input")))
		})

		It(
			"Should return an error for an undefined channel reference",
			func(ctx SpecContext) {
				a := channel.NewCalculationAnalyzer(StaticResolver{})
				ch := channel.Channel{
					Name:       "calc",
					Expression: "return nonexistent + 1",
				}
				res, err := a.Analyze(ctx, ch)
				Expect(err).To(MatchError(ContainSubstring("undefined symbol")))
				Expect(res.Unresolved).To(ConsistOf("nonexistent"))
			},
		)

		It(
			"Should return zero CalculationAnalysisResult on parse error",
			func(ctx SpecContext) {
				a := channel.NewCalculationAnalyzer(StaticResolver{})
				ch := channel.Channel{Name: "calc", Expression: "return {{invalid"}
				Expect(a.Analyze(ctx, ch)).Error().
					To(MatchError(ContainSubstring("extraneous input")))
			},
		)

		It("Should return unresolved names on analysis error", func(ctx SpecContext) {
			a := channel.NewCalculationAnalyzer(StaticResolver{})
			ch := channel.Channel{Name: "calc", Expression: "return nonexistent + 1"}
			res, err := a.Analyze(ctx, ch)
			Expect(err).To(MatchError(ContainSubstring("undefined symbol")))
			Expect(res.ChanDataType).To(Equal(telem.UnknownT))
			Expect(res.Unresolved).To(ConsistOf("nonexistent"))
		})
	})

	Describe("Derivative Operation Type Override", func() {
		It(
			"Should infer float64 DataType when derivative is the last operation",
			func(ctx SpecContext) {
				r := StaticResolver{
					{
						Name: "sensor",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   1,
					},
				}
				a := channel.NewCalculationAnalyzer(r)
				ch := channel.Channel{
					Name:       "deriv_calc",
					Expression: "return sensor",
					Operations: []channel.Operation{
						{Type: channel.OperationTypeDerivative},
					},
				}
				res := MustSucceed(a.Analyze(ctx, ch))
				Expect(res.ChanDataType).To(Equal(telem.Float64T))
				Expect(res.ExpressionReturnType).To(Equal(types.I32()))
			},
		)

		It(
			"Should not override DataType when avg is the last operation",
			func(ctx SpecContext) {
				r := StaticResolver{
					{
						Name: "sensor2",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   2,
					},
				}
				a := channel.NewCalculationAnalyzer(r)
				ch := channel.Channel{
					Name:       "avg_calc",
					Expression: "return sensor2",
					Operations: []channel.Operation{
						{Type: channel.OperationTypeAvg},
					},
				}
				res := MustSucceed(a.Analyze(ctx, ch))
				Expect(res.ChanDataType).To(Equal(telem.Int32T))
				Expect(res.ExpressionReturnType).To(Equal(types.I32()))
			},
		)
	})

	Describe("Resolver Fallback", func() {
		It(
			"Should fall back to the underlying resolver for symbols not in the temp cache",
			func(ctx SpecContext) {
				r := StaticResolver{
					{
						Name: "external",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   100,
					},
				}
				a := channel.NewCalculationAnalyzer(r)
				ch := channel.Channel{Name: "calc", Expression: "return external * 2.0"}
				res := MustSucceed(a.Analyze(ctx, ch))
				Expect(res.ChanDataType).To(Equal(telem.Float64T))
			},
		)
	})
	Describe("Dependency Removed During Analysis", func() {
		DescribeTable(
			"Should never report an analyzed channel without a data type",
			func(ctx SpecContext, found int) {
				r := &disappearingResolver{found: found}
				a := channel.NewCalculationAnalyzer(r)
				res, err := a.Analyze(ctx, channel.Channel{
					Name:       "calc",
					Expression: "return 100 / base",
				})
				if err == nil {
					Expect(res.ChanDataType).To(Equal(telem.Int64T))
				}
			},
			disappearingEntries(),
		)

		It(
			"Should resolve a dependency the same way throughout one analysis",
			func(ctx SpecContext) {
				r := &disappearingResolver{found: 1}
				a := channel.NewCalculationAnalyzer(r)
				res := MustSucceed(a.Analyze(ctx, channel.Channel{
					Name:       "calc",
					Expression: "return 100 / base",
				}))
				Expect(res.ChanDataType).To(Equal(telem.Int64T))
				Expect(r.calls).To(Equal(1))
			},
		)
	})
})

// disappearingResolver resolves "base" for its first found lookups, then reports it
// missing, as a rename that commits during analysis would.
type disappearingResolver struct {
	found int
	calls int
}

func (r *disappearingResolver) Resolve(
	_ context.Context,
	name string,
) (*symbol.Symbol, error) {
	if name != "base" {
		return nil, query.ErrNotFound
	}
	r.calls++
	if r.calls > r.found {
		return nil, query.ErrNotFound
	}
	return &symbol.Symbol{
		Name: "base",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.I64()),
		ID:   10,
	}, nil
}

func (r *disappearingResolver) Search(
	context.Context,
	string,
) ([]*symbol.Symbol, error) {
	return nil, nil
}

func disappearingEntries() []TableEntry {
	entries := make([]TableEntry, 0, 20)
	for found := range 20 {
		entries = append(entries, Entry(fmt.Sprintf("found %d times", found), found))
	}
	return entries
}
