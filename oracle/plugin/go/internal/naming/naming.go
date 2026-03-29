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
	"strings"
	"unicode"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/keywords"
	"github.com/synnaxlabs/oracle/resolution"
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

// ToPascalCase converts a name to PascalCase, preserving Go acronym conventions
// (e.g. "id" → "ID").
func ToPascalCase(s string) string {
	if IsScreamingCase(s) {
		return s
	}
	result := lo.PascalCase(s)
	result = strings.ReplaceAll(result, "Id", "ID")
	return result
}

// GetFieldName returns the Go field name for a schema field. It checks for a
// @go name override first, then falls back to ToPascalCase of the field name.
func GetFieldName(f resolution.Field) string {
	if override := domain.GetStringFromField(f, "go", "name"); override != "" {
		return override
	}
	return ToPascalCase(f.Name)
}

// GetGoName returns the Go name for a type. It checks for a @go name
// override first, then falls back to ToPascalCase of the schema type name.
func GetGoName(t resolution.Type) string {
	if override := domain.GetStringFromType(t, "go", "name"); override != "" {
		return override
	}
	return ToPascalCase(t.Name)
}

// LowerFirst lowercases the leading uppercase run of a string, handling
// acronyms correctly (e.g., "HTTPClient" -> "httpClient", "Key" -> "key").
// The result is escaped if it collides with a Go keyword.
func LowerFirst(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	i := 0
	for i < len(runes) && unicode.IsUpper(runes[i]) {
		i++
	}
	if i == 0 {
		return s
	}
	if i == 1 {
		runes[0] = unicode.ToLower(runes[0])
	} else if i == len(runes) {
		for j := range runes {
			runes[j] = unicode.ToLower(runes[j])
		}
	} else {
		for j := 0; j < i-1; j++ {
			runes[j] = unicode.ToLower(runes[j])
		}
	}
	return keywords.Escape(string(runes))
}

// DerivePackageName extracts the package name from an output path.
// Example: "core/pkg/service/user" -> "user"
func DerivePackageName(outputPath string) string { return filepath.Base(outputPath) }

// DerivePackageAlias creates a unique alias for an imported package to avoid
// conflicts. For migration version packages (e.g., "graph/migrations/v53"), the
// grandparent directory name is prepended to distinguish between packages at the
// same version across different source packages. Otherwise, if the base name
// conflicts with the current package, it prepends the parent directory.
func DerivePackageAlias(outputPath, currentPackage string) string {
	base := filepath.Base(outputPath)
	parent := filepath.Base(filepath.Dir(outputPath))
	if parent == "migrations" {
		grandparent := filepath.Base(filepath.Dir(filepath.Dir(outputPath)))
		return grandparent + base
	}
	if base == currentPackage {
		return parent + base
	}
	return base
}
