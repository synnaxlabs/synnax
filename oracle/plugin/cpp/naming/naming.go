// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package naming holds C++ name derivations shared between the cpp/types and
// cpp/json plugins, so the header and its JSON codec cannot disagree on a name.
package naming

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/plugin/internal/casing"
	"github.com/synnaxlabs/oracle/resolution"
)

// VariantTypeName returns the C++ struct name for one variant of a discriminated
// union (e.g. union "Scale" + value "linear" -> "ScaleLinear", union "AIChannel" +
// value "ai_voltage" -> "AIVoltageChannel"). cpp/types declares the variant
// struct under this name and cpp/json emits its parse/to_json bodies, so both
// MUST derive it identically; routing through this one function guarantees they
// cannot drift.
func VariantTypeName(unionCppName, variantValue string) string {
	return casing.VariantTypeName(unionCppName, variantValue)
}

// QualifiedVariantTypeName is VariantTypeName for a union name that may carry a
// C++ namespace qualifier (e.g. "::ni::AIChannel").
func QualifiedVariantTypeName(unionCppName, variantValue string) string {
	return casing.QualifiedVariantTypeName(unionCppName, variantValue, "::")
}

// Namespace converts a C++ output path to its fully qualified namespace. All
// three C++ plugins derive namespaces through this one function so a header,
// its JSON codec, and its proto translators cannot disagree.
func Namespace(outputPath string) string {
	parts := strings.Split(outputPath, "/")
	if len(parts) == 0 {
		return "synnax"
	}

	// Determine the top-level namespace based on the path prefix
	var topLevel string
	rest := parts[len(parts)-1:]
	switch {
	case len(parts) >= 2 && parts[0] == "x" && parts[1] == "cpp":
		topLevel = "x"
		rest = parts[2:]
	case len(parts) >= 2 && parts[0] == "client" && parts[1] == "cpp":
		topLevel = "synnax"
		rest = parts[2:]
	case len(parts) >= 2 && parts[0] == "arc" && parts[1] == "cpp":
		topLevel = "arc"
		rest = parts[2:]
	case parts[0] == "driver":
		topLevel = "driver"
		rest = parts[1:]
	default:
		topLevel = "synnax"
	}

	if len(rest) == 0 {
		return topLevel
	}
	return topLevel + "::" + strings.Join(rest, "::")
}

// PBNamespace converts a pb output path to a fully qualified C++ namespace.
// This mirrors the package derivation logic in pb/types plugin:
// - For "core/pkg/{layer}/{service}/pb" -> "::{layer}::{service}::pb"
// - For "x/go/{service}/pb" -> "::x::{service}::pb"
// - For other paths -> "::{first}::{last-before-pb}::pb"
func PBNamespace(pbOutputPath string) string {
	if pbOutputPath == "" {
		return "pb"
	}
	parts := strings.Split(pbOutputPath, "/")
	if len(parts) == 0 {
		return "pb"
	}

	// Derive namespace (directory before /pb, or last component)
	namespace := parts[len(parts)-1]
	if namespace == "pb" && len(parts) >= 2 {
		namespace = parts[len(parts)-2]
	}

	// Derive layer prefix (mirrors deriveLayerPrefix in pb/types)
	var prefix string
	if len(parts) >= 3 && parts[0] == "core" && parts[1] == "pkg" {
		prefix = parts[2] // e.g., "distribution", "service", "api"
	} else if len(parts) >= 1 && parts[0] != "" {
		prefix = parts[0] // e.g., "x", "freighter"
	} else {
		prefix = "synnax"
	}

	return fmt.Sprintf("::%s::%s::pb", prefix, namespace)
}

// PBName returns the type's @pb name override, or its declared name.
func PBName(s resolution.Type) string {
	if domain, ok := s.Domains["pb"]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
		}
	}
	return s.Name
}

// ScreamingSnake renders an enum constant name. It deliberately keeps
// lo.SnakeCase's word splitting so generated constants stay stable.
func ScreamingSnake(s string) string { return strings.ToUpper(lo.SnakeCase(s)) }
