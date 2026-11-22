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

	"github.com/synnaxlabs/arc/parser"
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
	return fmt.Sprintf("not exported: '%s' in module %q", e.Name, e.Module)
}

// pathClashError is returned for a dotted name that is both a member of an imported
// module and the path of a channel.
type pathClashError struct {
	// Path is the dotted name.
	Path string
	// Module is the name of the module that holds the member.
	Module string
}

// Error implements the error interface.
func (e *pathClashError) Error() string {
	return fmt.Sprintf("%s (module %s)", e.Path, e.Module)
}

// GetHint implements the hint interface of diagnostics.
func (e *pathClashError) GetHint() string {
	return "import the module with `as`, or rename the channel"
}

// UndefinedSymbolError is returned when a symbol cannot be resolved. It lazily
// computes "did you mean" suggestions only when GetHint() is called, avoiding
// expensive search operations on the common path where the error is discarded
// (e.g., during Add conflict checking).
type UndefinedSymbolError struct {
	ctx   context.Context
	Name  string
	scope *Symbol
	// channel is true when only a channel could have matched Name.
	channel bool
	// keyword is true for a dotted name with a keyword segment, which Arc cannot read as
	// the path of a channel.
	keyword bool
}

// Error implements the error interface.
func (e *UndefinedSymbolError) Error() string {
	if e.channel {
		return e.Name + " does not exist"
	}
	return "undefined symbol: " + e.Name
}

// keywordHint returns the hint for path, a dotted name with a keyword segment.
func keywordHint(path string) string {
	folder, name, _ := strings.Cut(path, ".")
	if folder == parser.LiteralAUTHORITY {
		return fmt.Sprintf(
			"folder %q cannot be read in Arc yet. rename the folder, or use the "+
				"channel's own name: %s",
			folder,
			name,
		)
	}
	return fmt.Sprintf("channel name %q cannot be read in Arc yet. rename the channel", name)
}

func isUndefined(err error) bool {
	_, undefined := errors.AsType[*UndefinedSymbolError](err)
	return undefined || errors.Is(err, query.ErrNotFound)
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
	if e.keyword {
		return keywordHint(e.Name)
	}
	if !e.channel && e.scope.IsAmbientModule(head) {
		return fmt.Sprintf(
			"module %q is not imported. add `import %s` at the top of the file",
			head,
			head,
		)
	}
	keep := keepAll
	if e.channel {
		keep = func(s *Symbol) bool { return s.Kind == KindChannel }
	}
	suggestions := e.scope.suggestSimilar(e.ctx, e.Name, 2, keep)
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
