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
	"github.com/synnaxlabs/oracle/plugin/primitives"
	"github.com/synnaxlabs/oracle/resolution"
)

// ImportAdder is implemented by language-specific import managers.
// It allows the resolver to add imports when resolving cross-namespace types.
type ImportAdder interface {
	// AddImport adds an import with the given category, path, and optional alias.
	AddImport(category string, path string, alias string)
}

// TypeFormatter formats type strings for a specific language.
type TypeFormatter interface {
	// FormatQualified formats a qualified type name (e.g., "pkg.Type" or "pkg::Type").
	FormatQualified(qualifier, typeName string) string
	// FormatGeneric formats a generic type with arguments (e.g., "Type[T]" or "Type<T>").
	FormatGeneric(baseName string, typeArgs []string) string
	// FormatArray formats a dynamic array type (e.g., "[]T" or "std::vector<T>").
	FormatArray(elemType string) string
	// FormatFixedArray formats a fixed-size array type (e.g., "[4]byte" or "std::array<T, 4>").
	FormatFixedArray(elemType string, size int64) string
	// FormatMap formats a map type (e.g., "map[K]V" or "std::unordered_map<K, V>").
	FormatMap(keyType, valType string) string
	// FallbackType returns the fallback type for unresolved references (e.g., "any" or "void").
	FallbackType() string
}

// ImportResolver resolves import paths for cross-namespace type references.
type ImportResolver interface {
	// ResolveImport returns the import information for a type in a different namespace.
	// Returns (importPath, qualifier, shouldImport).
	ResolveImport(outputPath string, ctx *Context) (importPath string, qualifier string, shouldImport bool)
}

// Resolver resolves Oracle type references to language-specific type strings.
type Resolver struct {
	Formatter       TypeFormatter
	ImportResolver  ImportResolver
	ImportAdder     ImportAdder
	PrimitiveMapper primitives.Mapper
}

// ResolveTypeRef converts a TypeRef to a language-specific type string.
func (r *Resolver) ResolveTypeRef(typeRef resolution.TypeRef, ctx *Context) string {
	// Handle type parameters
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		// For languages that don't support advanced generics, substitute
		// type parameters with defaults using their default value
		if ctx.SubstituteDefaultedTypeParams && typeRef.TypeParam.HasDefault() {
			return r.ResolveTypeRef(*typeRef.TypeParam.Default, ctx)
		}
		return typeRef.TypeParam.Name
	}

	// Handle Array<T> - both dynamic [] and fixed-size [N]
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		elemType := r.ResolveTypeRef(typeRef.TypeArgs[0], ctx)
		if typeRef.ArraySize != nil {
			return r.Formatter.FormatFixedArray(elemType, *typeRef.ArraySize)
		}
		return r.Formatter.FormatArray(elemType)
	}

	// Handle Map<K, V>
	if typeRef.Name == "Map" && len(typeRef.TypeArgs) >= 2 {
		keyType := r.ResolveTypeRef(typeRef.TypeArgs[0], ctx)
		valType := r.ResolveTypeRef(typeRef.TypeArgs[1], ctx)
		return r.Formatter.FormatMap(keyType, valType)
	}

	// Handle primitives
	if primitives.IsPrimitive(typeRef.Name) {
		mapping := r.PrimitiveMapper.Map(typeRef.Name)
		for _, imp := range mapping.Imports {
			r.ImportAdder.AddImport(imp.Category, imp.Path, imp.Name)
		}
		return mapping.TargetType
	}

	// Resolve named type
	resolved, ok := typeRef.Resolve(ctx.Table)
	if !ok {
		return r.Formatter.FallbackType()
	}

	switch resolved.Form.(type) {
	case resolution.StructForm:
		return r.resolveStructType(resolved, typeRef.TypeArgs, ctx)
	case resolution.EnumForm:
		return r.resolveEnumType(resolved, ctx)
	case resolution.DistinctForm:
		return r.resolveDistinctType(resolved, ctx)
	case resolution.AliasForm:
		return r.resolveAliasType(resolved, typeRef.TypeArgs, ctx)
	default:
		return r.Formatter.FallbackType()
	}
}

// resolveStructType resolves a struct type to a language-specific string.
func (r *Resolver) resolveStructType(resolved resolution.Type, typeArgs []resolution.TypeRef, ctx *Context) string {
	typeName := ctx.GetTypeName(resolved)

	// Build type arguments, filtering out those that correspond to defaulted params
	var args []string
	if form, ok := resolved.Form.(resolution.StructForm); ok {
		for i, arg := range typeArgs {
			// Skip type args that correspond to defaulted params
			if ctx.SubstituteDefaultedTypeParams && i < len(form.TypeParams) {
				if form.TypeParams[i].HasDefault() {
					continue
				}
			}
			args = append(args, r.ResolveTypeRef(arg, ctx))
		}
	} else {
		// Not a struct form, resolve all args
		for _, arg := range typeArgs {
			args = append(args, r.ResolveTypeRef(arg, ctx))
		}
	}
	baseType := r.Formatter.FormatGeneric(typeName, args)

	// Same namespace/output -> unqualified
	if ctx.IsSameOutput(resolved) {
		return baseType
	}

	// Different namespace -> add import and qualify
	targetOutputPath := ctx.GetOutputPath(resolved)
	if targetOutputPath == "" {
		return r.Formatter.FallbackType()
	}

	importPath, qualifier, shouldImport := r.ImportResolver.ResolveImport(targetOutputPath, ctx)
	if shouldImport {
		r.ImportAdder.AddImport("internal", importPath, qualifier)
	}

	return r.Formatter.FormatQualified(qualifier, baseType)
}

// resolveEnumType resolves an enum type to a language-specific string.
func (r *Resolver) resolveEnumType(resolved resolution.Type, ctx *Context) string {
	typeName := ctx.GetTypeName(resolved)

	// Same namespace/output -> unqualified
	if ctx.IsSameOutputEnum(resolved) {
		return typeName
	}

	// Different namespace -> add import and qualify
	targetOutputPath := ctx.GetEnumOutputPath(resolved)
	if targetOutputPath == "" {
		return r.Formatter.FallbackType()
	}

	importPath, qualifier, shouldImport := r.ImportResolver.ResolveImport(targetOutputPath, ctx)
	if shouldImport {
		r.ImportAdder.AddImport("internal", importPath, qualifier)
	}

	return r.Formatter.FormatQualified(qualifier, typeName)
}

// resolveDistinctType resolves a distinct type to a language-specific string.
func (r *Resolver) resolveDistinctType(resolved resolution.Type, ctx *Context) string {
	typeName := ctx.GetTypeName(resolved)

	// Same namespace/output -> unqualified
	if ctx.IsSameOutput(resolved) {
		return typeName
	}

	// Different namespace -> add import and qualify
	targetOutputPath := ctx.GetOutputPath(resolved)
	if targetOutputPath == "" {
		return r.Formatter.FallbackType()
	}

	importPath, qualifier, shouldImport := r.ImportResolver.ResolveImport(targetOutputPath, ctx)
	if shouldImport {
		r.ImportAdder.AddImport("internal", importPath, qualifier)
	}

	return r.Formatter.FormatQualified(qualifier, typeName)
}

// resolveAliasType resolves an alias type to a language-specific string.
// Unlike expanding the target, this uses the alias name directly.
func (r *Resolver) resolveAliasType(resolved resolution.Type, typeArgs []resolution.TypeRef, ctx *Context) string {
	typeName := ctx.GetTypeName(resolved)

	// Build type arguments, filtering out those that correspond to defaulted params
	// We check the alias's own type params, not the target's, because the alias
	// determines what type args are exposed in the output language.
	var args []string
	if aliasForm, ok := resolved.Form.(resolution.AliasForm); ok {
		for i, arg := range typeArgs {
			// Skip type args that correspond to defaulted params on the alias
			if ctx.SubstituteDefaultedTypeParams && i < len(aliasForm.TypeParams) {
				if aliasForm.TypeParams[i].HasDefault() {
					continue
				}
			}
			args = append(args, r.ResolveTypeRef(arg, ctx))
		}
	}
	baseType := r.Formatter.FormatGeneric(typeName, args)

	// Same namespace/output -> unqualified
	if ctx.IsSameOutput(resolved) {
		return baseType
	}

	// Different namespace -> add import and qualify
	targetOutputPath := ctx.GetOutputPath(resolved)
	if targetOutputPath == "" {
		return r.Formatter.FallbackType()
	}

	importPath, qualifier, shouldImport := r.ImportResolver.ResolveImport(targetOutputPath, ctx)
	if shouldImport {
		r.ImportAdder.AddImport("internal", importPath, qualifier)
	}

	return r.Formatter.FormatQualified(qualifier, baseType)
}
