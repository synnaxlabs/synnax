// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package resolver provides a shared framework for type resolution across Oracle plugins.
// It extracts the common patterns for resolving type references to language-specific strings.
package resolver

import (
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
)

// Context provides shared state during type resolution.
// It contains all the information needed to resolve types to language-specific strings.
type Context struct {
	// Table is the resolution table containing all type definitions.
	Table *resolution.Table
	// OutputPath is the output path of the current file being generated.
	OutputPath string
	// Namespace is the Oracle namespace of the current file.
	Namespace string
	// RepoRoot is the repository root path for resolving imports.
	RepoRoot string
	// DomainName is the language domain name (e.g., "go", "py", "ts", "cpp", "pb").
	DomainName string
}

// IsSameOutput checks if a resolved type is in the same output location as the current context.
// This determines whether to use qualified or unqualified type names.
func (c *Context) IsSameOutput(resolved resolution.Type) bool {
	targetOutputPath := output.GetPath(resolved, c.DomainName)
	return resolved.Namespace == c.Namespace &&
		(targetOutputPath == "" || targetOutputPath == c.OutputPath)
}

// IsSameOutputEnum checks if an enum type is in the same output location.
// Enums have special output path resolution logic.
func (c *Context) IsSameOutputEnum(resolved resolution.Type) bool {
	targetOutputPath := enum.FindOutputPath(resolved, c.Table, c.DomainName)
	return resolved.Namespace == c.Namespace &&
		(targetOutputPath == "" || targetOutputPath == c.OutputPath)
}

// GetOutputPath returns the output path for a resolved type.
func (c *Context) GetOutputPath(resolved resolution.Type) string {
	return output.GetPath(resolved, c.DomainName)
}

// GetEnumOutputPath returns the output path for an enum type.
func (c *Context) GetEnumOutputPath(resolved resolution.Type) string {
	return enum.FindOutputPath(resolved, c.Table, c.DomainName)
}

// GetNameOverride returns the language-specific name override for a type, if any.
// For example, @go name "CustomName" would return "CustomName".
func (c *Context) GetNameOverride(t resolution.Type) string {
	if domain, ok := t.Domains[c.DomainName]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
		}
	}
	return ""
}

// GetTypeName returns the type name, using any language-specific override.
func (c *Context) GetTypeName(t resolution.Type) string {
	if override := c.GetNameOverride(t); override != "" {
		return override
	}
	return t.Name
}
