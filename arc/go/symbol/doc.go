// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import "github.com/synnaxlabs/x/lsp/doc"

// Arc renders a fenced code block tagged with the "arc" language. Used by
// STL symbol declarations and the LSP layer to embed Arc source examples
// in hover and diagnostic markdown.
func Arc(content string) doc.Block { return doc.Code("arc", content) }

// WithDoc returns a copy of s with Doc set to the given blocks. Lets STL
// packages attach documentation in a chained var declaration so the Doc
// is in place before NewModule and Deprecate copy the symbol.
func (s Symbol) WithDoc(blocks ...doc.Block) Symbol {
	s.Doc = doc.New(blocks...)
	return s
}

// Document replaces s's Doc with the given blocks in place and returns s
// for chaining. The pointer-receiver companion to WithDoc, used for
// symbols obtained as *Symbol — typically module symbols built via
// NewModule.
func (s *Symbol) Document(blocks ...doc.Block) *Symbol {
	s.Doc = doc.New(blocks...)
	return s
}
