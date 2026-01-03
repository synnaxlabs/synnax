// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil provides common testing utilities for the analyzer package.
package testutil

import "github.com/antlr4-go/antlr/v4"

// MockToken implements antlr.Token for testing purposes.
type MockToken struct{ line, column int }

func (m *MockToken) GetSource() *antlr.TokenSourceCharStreamPair { return nil }
func (m *MockToken) GetTokenType() int                           { return 0 }
func (m *MockToken) GetChannel() int                             { return 0 }
func (m *MockToken) GetStart() int                               { return 0 }
func (m *MockToken) GetStop() int                                { return 0 }
func (m *MockToken) GetLine() int                                { return m.line }
func (m *MockToken) GetColumn() int                              { return m.column }
func (m *MockToken) GetText() string                             { return "" }
func (m *MockToken) SetText(string)                              {}
func (m *MockToken) GetTokenIndex() int                          { return 0 }
func (m *MockToken) SetTokenIndex(int)                           {}
func (m *MockToken) GetInputStream() antlr.CharStream            { return nil }
func (m *MockToken) GetTokenSource() antlr.TokenSource           { return nil }
func (m *MockToken) String() string                              { return "" }

// MockAST implements antlr.ParserRuleContext for testing purposes.
type MockAST struct {
	antlr.BaseParserRuleContext
	id    int
	token *MockToken
}

// NewMockAST creates a new mock AST node with the given ID.
func NewMockAST(id int) *MockAST {
	return &MockAST{
		id:    id,
		token: &MockToken{},
	}
}

// GetStart returns the start token for this AST node.
func (m *MockAST) GetStart() antlr.Token {
	return m.token
}
