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
	"cmp"
	"slices"
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/filename"
	"github.com/synnaxlabs/x/set"
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

// SortResources orders resources by name and then by ID. Bundle walks sort each
// directory's members with it before claiming file names, so Claims assigns numeric
// suffixes deterministically across exports.
func SortResources(resources []ontology.Resource) {
	slices.SortFunc(resources, func(a, b ontology.Resource) int {
		return cmp.Or(
			cmp.Compare(a.Name, b.Name),
			cmp.Compare(a.ID.String(), b.ID.String()),
		)
	})
}

// Claims tracks the file names taken in one bundle directory. Names are folded before
// comparison so a bundle extracts safely on case-insensitive file systems.
type Claims struct {
	// taken holds the folded names in use, reserved or claimed.
	taken set.Set[string]
}

// NewClaims returns a claim set for one bundle directory. reserved lists file names
// that no member may take.
func NewClaims(reserved ...string) *Claims {
	c := &Claims{taken: make(set.Set[string], len(reserved))}
	for _, name := range reserved {
		c.taken.Add(filename.Fold(name))
	}
	return c
}

// Claim records a member's file name and returns the name to export under. A name that
// folds to a taken name gains a numeric suffix before extension, so extension must be
// the one fileName carries; a directory passes "".
func (c *Claims) Claim(fileName, extension string) string {
	name := fileName
	for n := 1; c.taken.Contains(filename.Fold(name)); n++ {
		name = filename.WithSuffix(fileName, " ("+strconv.Itoa(n)+")", extension)
	}
	c.taken.Add(filename.Fold(name))
	return name
}
