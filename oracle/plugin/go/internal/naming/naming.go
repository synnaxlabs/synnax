// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package naming provides name derivation and case detection utilities for Go code
// generation.
package naming

import (
	"path/filepath"
	"unicode"
)

// IsScreamingCase returns true if s is all uppercase letters (possibly with
// underscores).
func IsScreamingCase(s string) bool {
	if s == "" {
		return false
	}
	hasLetter := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			hasLetter = true
			if !unicode.IsUpper(r) {
				return false
			}
		} else if r != '_' {
			return false
		}
	}
	return hasLetter
}

// DerivePackageName extracts the package name from an output path.
// Example: "core/pkg/service/user" -> "user"
func DerivePackageName(outputPath string) string { return filepath.Base(outputPath) }

// DerivePackageAlias creates a unique alias for an imported package to avoid
// conflicts. If the base name conflicts with the current package, it prepends
// the parent directory.
func DerivePackageAlias(outputPath, currentPackage string) string {
	base := filepath.Base(outputPath)
	if base == currentPackage {
		parent := filepath.Base(filepath.Dir(outputPath))
		return parent + base
	}
	return base
}
