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

	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// ErrImportFailed is returned for a member of an import that did not resolve. The
// import already reports why.
var ErrImportFailed = errors.Wrap(diagnostics.ErrReported, "import failed")

// notExportedError is returned when a user module holds the member but does not
// export it.
type notExportedError struct {
	// Name is the member that the module does not export.
	Name string
	// Module is the import path of the module that holds the member.
	Module string
}

// Error implements the error interface.
func (e *notExportedError) Error() string {
	return fmt.Sprintf("'%s' is not exported by module %q", e.Name, e.Module)
}

// UndefinedSymbolError is returned when a symbol cannot be resolved. It lazily
// computes "did you mean" suggestions only when GetHint() is called, avoiding
// expensive search operations on the common path where the error is discarded
// (e.g., during Add conflict checking).
type UndefinedSymbolError struct {
	ctx   context.Context
	Name  string
	scope *Symbol
}

// Error implements the error interface.
func (e *UndefinedSymbolError) Error() string {
	return fmt.Sprintf("undefined symbol: %s", e.Name)
}

func isUndefined(err error) bool {
	var undefined *UndefinedSymbolError
	return errors.Is(err, query.ErrNotFound) || errors.As(err, &undefined)
}

// GetHint implements the hint interface of diagnostics. It looks for two kinds of
// helpful information: (1) similar names visible in scope, surfaced as a
// "did you mean ...?" suggestion; and (2) when the name (or its dotted
// prefix) matches a module symbol available in the ambient prelude but
// not imported by the user, a hint to add the import.
func (e *UndefinedSymbolError) GetHint() string {
	head := e.Name
	if dot := strings.IndexByte(head, '.'); dot > 0 {
		head = head[:dot]
	}
	if e.scope.IsAmbientModule(head) {
		return fmt.Sprintf(
			"module %q is not imported. add `import %s` at the top of the file",
			head,
			head,
		)
	}
	suggestions := e.scope.SuggestSimilar(e.ctx, e.Name, 2)
	if len(suggestions) > 0 {
		return fmt.Sprintf("did you mean: %s?", strings.Join(suggestions, ", "))
	}
	return ""
}

// IsAmbientModule reports whether name matches a non-Internal KindModule
// child of an ancestor scope of kind KindAmbient reachable from s. Used to
// detect unimported references to modules available in the ambient prelude
// so callers can suggest an `import` fix. Internal modules are excluded
// because user code cannot import them.
func (s *Symbol) IsAmbientModule(name string) bool {
	for cur := s; cur != nil; cur = cur.Parent {
		if cur.Kind != KindAmbient {
			continue
		}
		if child := cur.FindChild(
			name,
		); child != nil && child.Kind == KindModule &&
			!child.Internal {
			return true
		}
	}
	return false
}
