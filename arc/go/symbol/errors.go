// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"context"
	"fmt"
	"strings"
)

// UndefinedSymbolError is returned when a symbol cannot be resolved. It lazily
// computes "did you mean" suggestions only when GetHint() is called, avoiding
// expensive search operations on the common path where the error is discarded
// (e.g., during Scope.Add conflict checking).
type UndefinedSymbolError struct {
	ctx   context.Context
	Name  string
	scope *Scope
}

// Error implements the error interface.
func (e *UndefinedSymbolError) Error() string {
	return fmt.Sprintf("undefined symbol: %s", e.Name)
}

// GetHint implements diagnostics.HintProvider. It lazily computes similar symbol
// suggestions using the scope's search infrastructure, which may involve expensive
// operations like Bleve full-text search.
func (e *UndefinedSymbolError) GetHint() string {
	suggestions := e.scope.SuggestSimilar(e.ctx, e.Name, 2)
	if len(suggestions) > 0 {
		return fmt.Sprintf("did you mean: %s?", strings.Join(suggestions, ", "))
	}
	return ""
}
