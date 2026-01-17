// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolver

import (
	"fmt"
	"strings"
)

// FormatterConfig defines language-specific formatting rules for types.
type FormatterConfig struct {
	// ArrayFormat formats an array type. If nil, uses default "[]elemType" style.
	ArrayFormat func(elemType string) string
	// FixedArrayFormat formats a fixed-size array. If nil, falls back to ArrayFormat.
	FixedArrayFormat func(elemType string, size int64) string
	// MapFormat formats a map type. If nil, uses default "map[K]V" style.
	MapFormat func(keyType, valType string) string
	// QualifiedSep is the separator between qualifier and type name (e.g., "." or "::").
	QualifiedSep string
	// GenericOpen is the opening bracket for generics (e.g., "<" or "[").
	GenericOpen string
	// GenericClose is the closing bracket for generics (e.g., ">" or "]").
	GenericClose string
	// FallbackTypeName is the type to use when resolution fails (e.g., "any", "void").
	FallbackTypeName string
	// SkipGenerics if true, FormatGeneric ignores type arguments and returns base name.
	SkipGenerics bool
}

// ConfigurableFormatter implements TypeFormatter using a FormatterConfig.
type ConfigurableFormatter struct {
	config FormatterConfig
}

// NewFormatter creates a TypeFormatter from the given configuration.
func NewFormatter(config FormatterConfig) TypeFormatter {
	return &ConfigurableFormatter{config: config}
}

func (f *ConfigurableFormatter) FormatQualified(qualifier, typeName string) string {
	if qualifier == "" {
		return typeName
	}
	return qualifier + f.config.QualifiedSep + typeName
}

func (f *ConfigurableFormatter) FormatGeneric(baseName string, typeArgs []string) string {
	if f.config.SkipGenerics || len(typeArgs) == 0 {
		return baseName
	}
	return fmt.Sprintf("%s%s%s%s", baseName, f.config.GenericOpen,
		strings.Join(typeArgs, ", "), f.config.GenericClose)
}

func (f *ConfigurableFormatter) FormatArray(elemType string) string {
	if f.config.ArrayFormat != nil {
		return f.config.ArrayFormat(elemType)
	}
	return "[]" + elemType
}

func (f *ConfigurableFormatter) FormatFixedArray(elemType string, size int64) string {
	if f.config.FixedArrayFormat != nil {
		return f.config.FixedArrayFormat(elemType, size)
	}
	// Fall back to regular array if not specified
	return f.FormatArray(elemType)
}

func (f *ConfigurableFormatter) FormatMap(keyType, valType string) string {
	if f.config.MapFormat != nil {
		return f.config.MapFormat(keyType, valType)
	}
	return fmt.Sprintf("map[%s]%s", keyType, valType)
}

func (f *ConfigurableFormatter) FallbackType() string {
	return f.config.FallbackTypeName
}

// Pre-configured formatters for common languages.
var (
	// GoFormatterConfig is the configuration for Go type formatting.
	GoFormatterConfig = FormatterConfig{
		QualifiedSep:     ".",
		GenericOpen:      "[",
		GenericClose:     "]",
		ArrayFormat:      func(elem string) string { return "[]" + elem },
		FixedArrayFormat: func(elem string, size int64) string { return fmt.Sprintf("[%d]%s", size, elem) },
		MapFormat:        func(k, v string) string { return fmt.Sprintf("map[%s]%s", k, v) },
		FallbackTypeName: "any",
	}

	// TSFormatterConfig is the configuration for TypeScript type formatting.
	TSFormatterConfig = FormatterConfig{
		QualifiedSep:     ".",
		GenericOpen:      "<",
		GenericClose:     ">",
		ArrayFormat:      func(elem string) string { return elem + "[]" },
		MapFormat:        func(k, v string) string { return fmt.Sprintf("Record<%s, %s>", k, v) },
		FallbackTypeName: "unknown",
	}

	// PyFormatterConfig is the configuration for Python type formatting.
	PyFormatterConfig = FormatterConfig{
		QualifiedSep:     ".",
		GenericOpen:      "[",
		GenericClose:     "]",
		ArrayFormat:      func(elem string) string { return fmt.Sprintf("list[%s]", elem) },
		MapFormat:        func(k, v string) string { return fmt.Sprintf("dict[%s, %s]", k, v) },
		FallbackTypeName: "Any",
	}

	// CppFormatterConfig is the configuration for C++ type formatting.
	CppFormatterConfig = FormatterConfig{
		QualifiedSep:     "::",
		GenericOpen:      "<",
		GenericClose:     ">",
		ArrayFormat:      func(elem string) string { return fmt.Sprintf("std::vector<%s>", elem) },
		MapFormat:        func(k, v string) string { return fmt.Sprintf("std::unordered_map<%s, %s>", k, v) },
		FallbackTypeName: "void",
	}

	// PbFormatterConfig is the configuration for Protocol Buffers type formatting.
	PbFormatterConfig = FormatterConfig{
		QualifiedSep:     ".",
		SkipGenerics:     true,                                     // Protobuf doesn't support generics
		ArrayFormat:      func(elem string) string { return elem }, // repeated fields don't change the type
		MapFormat:        func(k, v string) string { return fmt.Sprintf("map<%s, %s>", k, v) },
		FallbackTypeName: "bytes",
	}
)
