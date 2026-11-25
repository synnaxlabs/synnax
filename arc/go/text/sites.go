// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package text

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/set"
)

// ChannelSites returns the start of each token of content where a channel can be
// named, in code points. block parses content as the body of a function. A source that
// does not parse names no channel.
func ChannelSites(content string, block bool, cfg parser.Config) set.Set[int] {
	if !block {
		tree, diags := parser.Parse(content, cfg)
		if diags != nil {
			return nil
		}
		return sitesOf(tree, 0)
	}
	tree, diags := parser.ParseBlock("{"+content+"}", cfg)
	if diags != nil {
		return nil
	}
	return sitesOf(tree, 1)
}

// ExpressionSites returns the start of each token of the expression source where a
// channel can be named, in code points. A source that does not parse names no channel.
func ExpressionSites(source string, cfg parser.Config) set.Set[int] {
	tree, diags := parser.ParseExpression(source, cfg)
	if diags != nil {
		return nil
	}
	return sitesOf(tree, 0)
}

// sitesOf returns the start, less shift, of each token of tree where a channel can be
// named: a primary expression, a flow identifier, an assignment target, or an
// authority entry.
func sitesOf(tree antlr.ParseTree, shift int) set.Set[int] {
	l := &siteListener{sites: make(set.Set[int]), shift: shift}
	if tree != nil {
		antlr.ParseTreeWalkerDefault.Walk(l, tree)
	}
	return l.sites
}

type siteListener struct {
	parser.BaseArcParserListener
	sites set.Set[int]
	shift int
}

func (l *siteListener) add(nodes ...antlr.TerminalNode) {
	for _, n := range nodes {
		if n != nil {
			l.sites.Add(n.GetSymbol().GetStart() - l.shift)
		}
	}
}

func (l *siteListener) EnterPrimaryExpression(c *parser.PrimaryExpressionContext) {
	l.add(c.IDENTIFIER(), c.CHANNEL_NAME())
}

func (l *siteListener) EnterIdentifier(c *parser.IdentifierContext) {
	l.add(c.IDENTIFIER(), c.CHANNEL_NAME())
}

func (l *siteListener) EnterAssignment(c *parser.AssignmentContext) {
	l.add(c.IDENTIFIER(), c.CHANNEL_NAME())
}

func (l *siteListener) EnterAuthorityEntry(c *parser.AuthorityEntryContext) {
	l.add(c.IDENTIFIER(), c.CHANNEL_NAME())
}
