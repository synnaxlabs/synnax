// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package authority provides semantic analysis for Arc authority declarations.
package authority

import (
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
)

// Analyze validates authority blocks in a program and returns the AuthorityConfig.
// It checks:
//  1. Authority values are in range 0-255
//  2. At most one bare numeric default
//  3. Channel identifiers exist in the symbol resolver
//  4. No duplicate channel entries
//  5. Authority blocks appear before function/flow/sequence declarations
func Analyze(
	ctx acontext.Context[parser.IProgramContext],
) ir.AuthorityConfig {
	var (
		config          ir.AuthorityConfig
		seenDeclaration bool
		hasDefault      bool
		seenChannels    = make(map[string]bool)
	)
	for _, item := range ctx.AST.AllTopLevelItem() {
		authBlock := item.AuthorityBlock()
		if authBlock == nil {
			seenDeclaration = true
			continue
		}
		if seenDeclaration {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				authBlock,
				"authority declaration must appear before function, "+
					"flow, and sequence declarations",
			))
			continue
		}

		// Simple form: authority 200
		if lit := authBlock.INTEGER_LITERAL(); lit != nil {
			if hasDefault {
				ctx.Diagnostics.Add(diagnostics.Errorf(
					authBlock,
					"multiple default authority values",
				))
				continue
			}
			v, ok := parseAuthorityValue(ctx.Diagnostics, authBlock, lit.GetText())
			if ok {
				hasDefault = true
				config.Default = &v
			}
			continue
		}

		// Grouped form: authority ( ... )
		for _, entry := range authBlock.AllAuthorityEntry() {
			analyzeEntry(ctx, entry, &config, &hasDefault, seenChannels)
		}
	}
	return config
}

func analyzeEntry(
	ctx acontext.Context[parser.IProgramContext],
	entry parser.IAuthorityEntryContext,
	config *ir.AuthorityConfig,
	hasDefault *bool,
	seenChannels map[string]bool,
) {
	if id := entry.IDENTIFIER(); id != nil {
		name := id.GetText()
		if seenChannels[name] {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				entry,
				"duplicate authority for '%s'",
				name,
			))
			return
		}
		sym, err := ctx.Scope.Resolve(ctx, name)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				entry,
				"channel '%s' not found",
				name,
			))
			return
		}
		v, ok := parseAuthorityValue(ctx.Diagnostics, entry, entry.INTEGER_LITERAL().GetText())
		if !ok {
			return
		}
		seenChannels[name] = true
		if config.Channels == nil {
			config.Channels = make(map[string]uint8)
		}
		config.Channels[name] = v
		if config.Keys == nil {
			config.Keys = make(map[uint32]string)
		}
		config.Keys[uint32(sym.ID)] = name
	} else {
		if *hasDefault {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				entry,
				"multiple default authority values",
			))
			return
		}
		v, ok := parseAuthorityValue(ctx.Diagnostics, entry, entry.INTEGER_LITERAL().GetText())
		if !ok {
			return
		}
		*hasDefault = true
		config.Default = &v
	}
}

// ResolveChannelKey resolves a channel name to its key using the symbol resolver.
func ResolveChannelKey(
	ctx acontext.Context[parser.IProgramContext],
	name string,
) (uint32, bool) {
	sym, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		return 0, false
	}
	if sym.Kind != symbol.KindChannel {
		return 0, false
	}
	return uint32(sym.ID), true
}

func parseAuthorityValue(
	diag *diagnostics.Diagnostics,
	node antlr.ParserRuleContext,
	text string,
) (uint8, bool) {
	v, err := strconv.ParseUint(text, 10, 64)
	if err != nil || v > 255 {
		diag.Add(diagnostics.Errorf(
			node,
			"authority value must be an integer in range 0-255, got '%s'",
			text,
		))
		return 0, false
	}
	return uint8(v), true
}
