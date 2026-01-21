// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/arc/symbol"
	"go.lsp.dev/protocol"
)

func (s *Server) FoldingRange(
	_ context.Context,
	params *protocol.FoldingRangeParams,
) ([]protocol.FoldingRange, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok || doc.IR.Symbols == nil {
		return []protocol.FoldingRange{}, nil
	}
	var ranges []protocol.FoldingRange
	collectFoldingRanges(doc.IR.Symbols, &ranges)
	return ranges, nil
}

func collectFoldingRanges(scope *symbol.Scope, ranges *[]protocol.FoldingRange) {
	if scope.AST != nil && isFoldableKind(scope.Kind) {
		start := scope.AST.GetStart()
		stop := scope.AST.GetStop()
		if start != nil && stop != nil {
			startLine := start.GetLine() - 1
			endLine := stop.GetLine() - 1
			if endLine > startLine {
				kind := foldingRangeKind(scope.Kind)
				*ranges = append(*ranges, protocol.FoldingRange{
					StartLine:      uint32(startLine),
					StartCharacter: uint32(start.GetColumn()),
					EndLine:        uint32(endLine),
					EndCharacter:   uint32(stop.GetColumn() + len(stop.GetText())),
					Kind:           kind,
				})
			}
		}
	}
	for _, child := range scope.Children {
		collectFoldingRanges(child, ranges)
	}
}

func isFoldableKind(kind symbol.Kind) bool {
	switch kind {
	case symbol.KindFunction, symbol.KindSequence, symbol.KindStage:
		return true
	default:
		return false
	}
}

func foldingRangeKind(kind symbol.Kind) protocol.FoldingRangeKind {
	switch kind {
	case symbol.KindFunction, symbol.KindSequence, symbol.KindStage:
		return protocol.RegionFoldingRange
	default:
		return protocol.RegionFoldingRange
	}
}
