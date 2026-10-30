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

type mockToken struct{ line, column int }

func (m *mockToken) GetSource() *antlr.TokenSourceCharStreamPair { return nil }
func (m *mockToken) GetTokenType() int                           { return 0 }
func (m *mockToken) GetChannel() int                             { return 0 }
func (m *mockToken) GetStart() int                               { return 0 }
func (m *mockToken) GetStop() int                                { return 0 }
func (m *mockToken) GetLine() int                                { return m.line }
func (m *mockToken) GetColumn() int                              { return m.column }
func (m *mockToken) GetText() string                             { return "" }
func (m *mockToken) SetText(string)                              {}
func (m *mockToken) GetTokenIndex() int                          { return 0 }
func (m *mockToken) SetTokenIndex(int)                           {}
func (m *mockToken) GetInputStream() antlr.CharStream            { return nil }
func (m *mockToken) GetTokenSource() antlr.TokenSource           { return nil }
func (m *mockToken) String() string                              { return "" }

// MockAST implements antlr.ParserRuleContext for testing purposes.
type MockAST struct {
	token *mockToken
	antlr.BaseParserRuleContext
	id int
}

// NewMockAST creates a new mock AST node with the given ID.
func NewMockAST(id int) *MockAST {
	return &MockAST{
		id:    id,
		token: &mockToken{},
	}
}

// GetStart returns the start token for this AST node.
func (m *MockAST) GetStart() antlr.Token {
	return m.token
}
