// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stl_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

type stubModule struct {
	symbol.MapResolver
}

func (s *stubModule) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type == "stub" {
		return nil, nil
	}
	return nil, query.ErrNotFound
}

func (s *stubModule) BindTo(_ context.Context, _ stl.HostRuntime) error {
	return nil
}

var _ stl.Module = (*stubModule)(nil)

var _ = Describe("CompoundResolver", func() {
	It("Should derive a working resolver from modules", func() {
		m1 := &stubModule{MapResolver: symbol.MapResolver{
			"foo": symbol.Symbol{Name: "foo", Type: types.I32()},
		}}
		m2 := &stubModule{MapResolver: symbol.MapResolver{
			"bar": symbol.Symbol{Name: "bar", Type: types.F64()},
		}}
		resolver := stl.CompoundResolver(m1, m2)
		sym := MustSucceed(resolver.Resolve(bCtx, "bar"))
		Expect(sym.Name).To(Equal("bar"))
		Expect(sym.Type).To(Equal(types.F64()))
	})
})

var _ = Describe("MultiFactory", func() {
	It("Should derive a working factory from modules", func() {
		m1 := &stubModule{MapResolver: symbol.MapResolver{}}
		m2 := &stubModule{MapResolver: symbol.MapResolver{}}
		factory := stl.MultiFactory(m1, m2)
		Expect(factory).To(HaveLen(2))
	})
})

var _ = Describe("ModuleResolver", func() {
	var resolver *stl.ModuleResolver

	BeforeEach(func() {
		resolver = &stl.ModuleResolver{
			Name: "math",
			Members: symbol.MapResolver{
				"abs":   symbol.Symbol{Name: "abs", Kind: symbol.KindFunction, Type: types.F64()},
				"sqrt":  symbol.Symbol{Name: "sqrt", Kind: symbol.KindFunction, Type: types.F64()},
				"floor": symbol.Symbol{Name: "floor", Kind: symbol.KindFunction, Type: types.F64()},
			},
		}
	})

	Describe("Resolve", func() {
		It("Should resolve a qualified name", func() {
			sym := MustSucceed(resolver.Resolve(bCtx, "math.abs"))
			Expect(sym.Name).To(Equal("abs"))
			Expect(sym.Type).To(Equal(types.F64()))
		})

		It("Should return ErrNotFound for a bare name", func() {
			Expect(resolver.Resolve(bCtx, "abs")).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should return ErrNotFound for a wrong prefix", func() {
			Expect(resolver.Resolve(bCtx, "string.abs")).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should return ErrNotFound for a non-existent member", func() {
			Expect(resolver.Resolve(bCtx, "math.sin")).Error().To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Search", func() {
		It("Should search with a qualified prefix", func() {
			symbols := MustSucceed(resolver.Search(bCtx, "math.sq"))
			Expect(symbols).To(HaveLen(1))
			Expect(symbols[0].Name).To(Equal("sqrt"))
		})

		It("Should return all members when term is a prefix of module name", func() {
			symbols := MustSucceed(resolver.Search(bCtx, "ma"))
			Expect(symbols).To(HaveLen(3))
		})

		It("Should return all members when term is exact module name", func() {
			symbols := MustSucceed(resolver.Search(bCtx, "math"))
			Expect(symbols).To(HaveLen(3))
		})

		It("Should return all members when term is module name with dot", func() {
			symbols := MustSucceed(resolver.Search(bCtx, "math."))
			Expect(symbols).To(HaveLen(3))
		})

		It("Should return empty for an unrelated term", func() {
			symbols := MustSucceed(resolver.Search(bCtx, "completely_different"))
			Expect(symbols).To(BeEmpty())
		})
	})
})
