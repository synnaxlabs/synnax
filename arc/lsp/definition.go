// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lsp

import (
	"context"

	"go.lsp.dev/protocol"
)

func (s *Server) Definition(ctx context.Context, params *protocol.DefinitionParams) ([]protocol.Location, error) {
	s.mu.RLock()
	doc, ok := s.documents[params.TextDocument.URI]
	s.mu.RUnlock()
	if !ok || doc.IR.Symbols == nil {
		return nil, nil
	}
	word := s.getWordAtPosition(doc.Content, params.Position)
	if word == "" {
		return nil, nil
	}
	searchPos := params.Position
	scopeAtCursor := s.findScopeAtPosition(doc.IR.Symbols, searchPos)
	sym, err := scopeAtCursor.Resolve(ctx, word)
	if err != nil {
		return nil, nil
	}
	location := s.symbolToLocation(params.TextDocument.URI, sym)
	if location == nil {
		return nil, nil
	}
	return []protocol.Location{*location}, nil
}
