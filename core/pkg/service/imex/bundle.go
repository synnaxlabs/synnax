// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imex

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/filename"
	"github.com/synnaxlabs/x/validate"
)

// ManifestBaseName is the manifest's file name without an extension. The codec supplies
// the extension, so every serialization has one recognition point.
const ManifestBaseName = "manifest"

// Manifest is the body of the manifest file at a bundle's root. Membership is inferred
// from the files beside the manifest, so it names no members.
type Manifest struct {
	// Version governs the manifest schema and the bundle's layout rules.
	Version uint8 `json:"version"`
	// Type is the bundle kind, letting an endpoint reject a bundle of another kind.
	Type string `json:"type"`
	// Name is the exported resource's name, which an import gives the resource it
	// creates.
	Name string `json:"name"`
}

// Claims tracks the file names taken in one bundle directory. Names are folded before
// comparison so a bundle extracts safely on case-insensitive file systems.
type Claims struct {
	// prefix is the directory's path from the bundle root ("" for the root,
	// "a/b/" otherwise), prepended to file names in collision errors.
	prefix string
	// reserved maps folded reserved names to their display form.
	reserved map[string]string
	// claimed maps each folded file name to the resource that took it.
	claimed map[string]string
}

// NewClaims returns a claim set for the bundle directory at prefix ("" for the root,
// "a/b/" otherwise). reserved lists file names, in display form, that no member may
// take.
func NewClaims(prefix string, reserved ...string) *Claims {
	c := &Claims{
		prefix:   prefix,
		reserved: make(map[string]string, len(reserved)),
		claimed:  map[string]string{},
	}
	for _, name := range reserved {
		c.reserved[filename.Fold(name)] = name
	}
	return c
}

// Claim records fileName as taken by the resource named owner. It returns a validation
// error when fileName folds to a reserved name or to a name another resource claimed.
func (c *Claims) Claim(owner, fileName string) error {
	folded := filename.Fold(fileName)
	if display, ok := c.reserved[folded]; ok {
		return errors.Wrapf(
			validate.ErrValidation,
			"%q takes the reserved file name %q; rename it and export again",
			owner, display,
		)
	}
	if prev, ok := c.claimed[folded]; ok {
		return errors.Wrapf(
			validate.ErrValidation,
			"%q and %q both export to %q; rename one and export again",
			prev, owner, c.prefix+fileName,
		)
	}
	c.claimed[folded] = owner
	return nil
}
