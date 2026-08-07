// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package authority_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/authority"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Authority Analyzer", func() {
	var (
		channels = []symbol.Symbol{
			{
				Name: "valve",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
				ID:   100,
			},
			{
				Name: "vent",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
				ID:   200,
			},
		}
		root *symbol.Symbol
	)
	BeforeEach(func() {
		root = NewRoot(nil, channels...)
	})

	Describe("Simple Form", func() {
		It("Should parse a simple authority declaration", func(specCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`authority 200`))
			ctx := acontext.NewRoot(specCtx, prog, root)
			config := authority.Analyze(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			Expect(config.Default).ToNot(BeNil())
			Expect(*config.Default).To(Equal(uint8(200)))
		})

		It("Should accept authority 0", func(specCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`authority 0`))
			ctx := acontext.NewRoot(specCtx, prog, root)
			config := authority.Analyze(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			Expect(config.Default).ToNot(BeNil())
			Expect(*config.Default).To(Equal(uint8(0)))
		})

		It("Should accept authority 255", func(specCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`authority 255`))
			ctx := acontext.NewRoot(specCtx, prog, root)
			config := authority.Analyze(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			Expect(config.Default).ToNot(BeNil())
			Expect(*config.Default).To(Equal(uint8(255)))
		})

		It("Should reject authority value > 255", func(specCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`authority 256`))
			ctx := acontext.NewRoot(specCtx, prog, root)
			authority.Analyze(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(ctx.Diagnostics.String()).To(ContainSubstring("0-255"))
		})
	})

	Describe("Grouped Form", func() {
		It(
			"Should parse grouped authority with default only",
			func(specCtx SpecContext) {
				prog := MustSucceed(parser.Parse(`authority (200)`))
				ctx := acontext.NewRoot(specCtx, prog, root)
				config := authority.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				Expect(config.Default).ToNot(BeNil())
				Expect(*config.Default).To(Equal(uint8(200)))
			},
		)

		It(
			"Should parse grouped authority with default and channel overrides",
			func(specCtx SpecContext) {
				prog := MustSucceed(parser.Parse(`authority (200 valve 100 vent 150)`))
				ctx := acontext.NewRoot(specCtx, prog, root)
				config := authority.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				Expect(config.Default).ToNot(BeNil())
				Expect(*config.Default).To(Equal(uint8(200)))
				Expect(config.Channels).To(HaveLen(2))
				Expect(config.Channels[100]).To(Equal(uint8(100)))
				Expect(config.Channels[200]).To(Equal(uint8(150)))
			},
		)

		It(
			"Should parse grouped authority with channel overrides only",
			func(specCtx SpecContext) {
				prog := MustSucceed(parser.Parse(`authority (valve 100)`))
				ctx := acontext.NewRoot(specCtx, prog, root)
				config := authority.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				Expect(config.Default).To(BeNil())
				Expect(config.Channels).To(HaveLen(1))
				Expect(config.Channels[100]).To(Equal(uint8(100)))
			},
		)

		It("Should parse empty grouped authority", func(specCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`authority ()`))
			ctx := acontext.NewRoot(specCtx, prog, root)
			config := authority.Analyze(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			Expect(config.Default).To(BeNil())
			Expect(config.Channels).To(BeNil())
		})
	})

	Describe("Validation", func() {
		It(
			"Should reject multiple default authority values (simple form)",
			func(specCtx SpecContext) {
				prog := MustSucceed(parser.Parse("authority 200\nauthority 100"))
				ctx := acontext.NewRoot(specCtx, prog, root)
				authority.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(
					ctx.Diagnostics.String(),
				).To(ContainSubstring("multiple default"))
			},
		)

		It(
			"Should reject multiple default authority values (grouped form)",
			func(specCtx SpecContext) {
				prog := MustSucceed(parser.Parse(`authority (200 100)`))
				ctx := acontext.NewRoot(specCtx, prog, root)
				authority.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(
					ctx.Diagnostics.String(),
				).To(ContainSubstring("multiple default"))
			},
		)

		It("Should reject duplicate channel entries", func(specCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`authority (valve 100 valve 200)`))
			ctx := acontext.NewRoot(specCtx, prog, root)
			authority.Analyze(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(ctx.Diagnostics.String()).To(ContainSubstring("duplicate"))
		})

		It("Should reject non-existent channel", func(specCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`authority (nonexistent 100)`))
			ctx := acontext.NewRoot(specCtx, prog, root)
			authority.Analyze(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(ctx.Diagnostics.String()).To(ContainSubstring("not found"))
		})

		It(
			"Should reject authority block after function declaration",
			func(specCtx SpecContext) {
				prog := MustSucceed(parser.Parse(`
				func test{} () {}
				authority 200
			`))
				ctx := acontext.NewRoot(specCtx, prog, root)
				authority.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(ctx.Diagnostics.String()).To(ContainSubstring("before"))
			},
		)

		It(
			"Should allow an import block before the authority declaration",
			func(specCtx SpecContext) {
				prog := MustSucceed(parser.Parse(`
				import time
				authority 200
			`))
				ctx := acontext.NewRoot(specCtx, prog, root)
				config := authority.Analyze(ctx)
				Expect(ctx.Diagnostics.String()).ToNot(ContainSubstring("before"))
				Expect(config.Default).ToNot(BeNil())
				Expect(*config.Default).To(Equal(uint8(200)))
			},
		)

		It(
			"Should allow multiple imports before the authority declaration",
			func(specCtx SpecContext) {
				prog := MustSucceed(parser.Parse(`
				import time
				import math
				authority 200
			`))
				ctx := acontext.NewRoot(specCtx, prog, root)
				config := authority.Analyze(ctx)
				Expect(ctx.Diagnostics.String()).ToNot(ContainSubstring("before"))
				Expect(config.Default).ToNot(BeNil())
				Expect(*config.Default).To(Equal(uint8(200)))
			},
		)

		It(
			"Should still reject authority appearing after a function even with imports first",
			func(specCtx SpecContext) {
				prog := MustSucceed(parser.Parse(`
				import time
				func test{} () {}
				authority 200
			`))
				ctx := acontext.NewRoot(specCtx, prog, root)
				authority.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(ctx.Diagnostics.String()).To(ContainSubstring("before"))
			},
		)

		It(
			"Should reject channel-specific authority value > 255",
			func(specCtx SpecContext) {
				prog := MustSucceed(parser.Parse(`authority (valve 300)`))
				ctx := acontext.NewRoot(specCtx, prog, root)
				authority.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(ctx.Diagnostics.String()).To(ContainSubstring("0-255"))
			},
		)
	})

	Describe("No Authority", func() {
		It(
			"Should return zero config when no authority blocks exist",
			func(specCtx SpecContext) {
				prog := MustSucceed(parser.Parse(`func test{} () {}`))
				ctx := acontext.NewRoot(specCtx, prog, root)
				config := authority.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				Expect(config.Default).To(BeNil())
				Expect(config.Channels).To(BeNil())
			},
		)
	})
})
