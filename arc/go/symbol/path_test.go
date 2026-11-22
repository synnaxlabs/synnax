// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
)

// countingResolver counts the lookups that reach a StaticResolver.
type countingResolver struct {
	StaticResolver
	names, keys int
}

func (r *countingResolver) Resolve(
	ctx context.Context,
	name string,
) (*symbol.Symbol, error) {
	r.names++
	return r.StaticResolver.Resolve(ctx, name)
}

func (r *countingResolver) ResolveKey(
	ctx context.Context,
	key uint32,
) (*symbol.Symbol, error) {
	r.keys++
	return r.StaticResolver.ResolveKey(ctx, key)
}

func pathChannel(name string, id int) symbol.Symbol {
	return symbol.Symbol{
		Name: name,
		Kind: symbol.KindChannel,
		Type: types.Chan(types.F64()),
		ID:   id,
	}
}

var _ = Describe("Paths", func() {
	var (
		channels *countingResolver
		root     *symbol.Symbol
		alias    *symbol.Symbol
	)
	BeforeEach(func(ctx SpecContext) {
		channels = &countingResolver{StaticResolver: StaticResolver{
			pathChannel("stand.pt_01", 1),
			pathChannel("valves.open", 2),
			pathChannel("valves.open.cmd", 3),
			pathChannel("2024", 4),
			pathChannel("other", 2024),
		}}
		valves := &symbol.Symbol{Name: "valves", Kind: symbol.KindModule}
		valves.AddChild(
			&symbol.Symbol{Name: "open", Kind: symbol.KindFunction},
			&symbol.Symbol{Name: "limit", Kind: symbol.KindConstant, Type: types.F64()},
		)
		root = symbol.NewRoot(channels, stl.NewSymbols())
		alias = MustSucceed(root.Add(ctx, symbol.Symbol{
			Name:   "valves",
			Kind:   symbol.KindModuleAlias,
			Target: valves,
		}))
		channels.names, channels.keys = 0, 0
	})
	resolve := func(ctx context.Context, n parser.Name) (*symbol.Symbol, error) {
		return root.ResolveName(ctx, n)
	}

	It("Should resolve a bare path to the channel at that path", func(ctx SpecContext) {
		Expect(resolve(ctx, parser.Name{Head: "stand", Tail: "pt_01"})).
			To(HaveField("ID", 1))
	})

	It("Should resolve a module member that no channel path matches", func(
		ctx SpecContext,
	) {
		Expect(resolve(ctx, parser.Name{Head: "valves", Tail: "limit"})).
			To(HaveField("Name", "limit"))
		Expect(alias.Used).To(BeTrue())
	})

	It("Should fail a name that is both a module member and a channel path", func(
		ctx SpecContext,
	) {
		_, err := resolve(ctx, parser.Name{Head: "valves", Tail: "open"})
		Expect(err).To(MatchError(
			"valves.open is both a member of module valves and a channel",
		))
		d := diagnostics.Error(err, nil)
		Expect(d.Code).To(Equal(codes.PathClash.Code))
		Expect(d.Notes).To(ContainElement(HaveField(
			"Message",
			Equal("import the module with `as`, or rename the channel"),
		)))
		Expect(alias.Used).To(BeFalse())
	})

	It("Should resolve a callee as a module member with no channel lookup", func(
		ctx SpecContext,
	) {
		Expect(resolve(ctx, parser.Name{Head: "valves", Tail: "open", Callee: true})).
			To(HaveField("Kind", symbol.KindFunction))
		Expect(channels.names).To(BeZero())
	})

	It("Should resolve three segments under a module only as a channel path", func(
		ctx SpecContext,
	) {
		Expect(resolve(ctx, parser.Name{Head: "valves", Tail: "open.cmd"})).
			To(HaveField("ID", 3))
	})

	It("Should report the missing member of an imported module", func(
		ctx SpecContext,
	) {
		Expect(resolve(ctx, parser.Name{Head: "valves", Tail: "nope"})).Error().
			To(MatchError("undefined symbol: nope"))
	})

	It("Should report that a path with no channel does not exist", func(
		ctx SpecContext,
	) {
		Expect(resolve(ctx, parser.Name{Head: "stand", Tail: "pt_02"})).Error().
			To(MatchError("stand.pt_02 does not exist"))
	})

	It("Should keep the import hint for a module that is not imported", func(
		ctx SpecContext,
	) {
		_, err := resolve(ctx, parser.Name{Head: "math", Tail: "avg"})
		Expect(err).To(MatchError("undefined symbol: math.avg"))
		Expect(diagnostics.Error(err, nil).Notes).To(ContainElement(HaveField(
			"Message",
			ContainSubstring(`module "math" is not imported`),
		)))
	})

	It("Should not let a variable named like the head hide the path", func(
		ctx SpecContext,
	) {
		MustSucceed(root.Add(ctx, symbol.Symbol{
			Name: "stand",
			Kind: symbol.KindVariable,
			Type: types.F64(),
		}))
		Expect(resolve(ctx, parser.Name{Head: "stand", Tail: "pt_01"})).
			To(HaveField("ID", 1))
	})

	It("Should resolve digits as the channel of that name", func(ctx SpecContext) {
		Expect(resolve(ctx, parser.Name{Head: "2024"})).
			To(HaveField("ID", 4))
	})

	It("Should find a channel by key through the key lookup only", func(
		ctx SpecContext,
	) {
		Expect(root.ResolveChannelKey(ctx, 2024)).To(HaveField("Name", "other"))
		Expect(channels.keys).To(Equal(1))
		Expect(channels.names).To(BeZero())
	})
})
