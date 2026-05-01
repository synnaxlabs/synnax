// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package parser

import "github.com/antlr4-go/antlr/v4"

// FindIdentifierToken walks node depth-first and returns the first
// IDENTIFIER terminal whose text equals name, or nil if none is found.
// Matching by text correctly handles tuple declarations like
// `for k, v := range ...` where two symbols share the same enclosing rule
// context. Useful when a caller has a parse-tree node and a known symbol
// name and wants to address the name's range specifically (e.g., to
// narrow a diagnostic from the full declaration to the identifier).
func FindIdentifierToken(node antlr.Tree, name string) antlr.TerminalNode {
	if node == nil || name == "" {
		return nil
	}
	for i := 0; i < node.GetChildCount(); i++ {
		child := node.GetChild(i)
		if term, ok := child.(antlr.TerminalNode); ok {
			tok := term.GetSymbol()
			if tok != nil &&
				tok.GetTokenType() == ArcParserIDENTIFIER &&
				tok.GetText() == name {
				return term
			}
			continue
		}
		if found := FindIdentifierToken(child, name); found != nil {
			return found
		}
	}
	return nil
}
