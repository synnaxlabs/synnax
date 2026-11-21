// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package marshal

import (
	"slices"

	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	"github.com/synnaxlabs/oracle/resolution"
)

// pinScope resolves a type to the version that the types a builder is inside pin.
type pinScope struct {
	// pins resolves a stored reference to the dependency version that an owner's
	// version file imports.
	pins versioning.Pins
	// owners are the qualified names of the types the builder is inside, innermost
	// last.
	owners []string
}

// enter makes qualifiedName the innermost owner until the returned function runs.
func (s *pinScope) enter(qualifiedName string) func() {
	s.owners = append(s.owners, qualifiedName)
	return func() { s.owners = s.owners[:len(s.owners)-1] }
}

// path returns the @go output path of typ at the version that the innermost owner
// pinning typ's schema imports. A struct pins no version of its own schema, so its
// fields of that schema take the version that the owner holding the struct pins.
func (s *pinScope) path(typ resolution.Type) (string, bool) {
	for _, owner := range slices.Backward(s.owners) {
		if p, ok := s.pins.Path(owner, typ); ok {
			return p, true
		}
	}
	return "", false
}
