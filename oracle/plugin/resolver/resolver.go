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
	AddImport(category, path, alias string)
}

// TypeFormatter formats type strings for a specific language.
type TypeFormatter interface {
	// FormatQualified formats a qualified type name (e.g., "pkg.Type" or "pkg::Type").
	FormatQualified(qualifier, typeName string) string
	// FormatGeneric formats a generic type with arguments (e.g., "Type[T]" or
	// "Type<T>").
	FormatGeneric(baseName string, typeArgs []string) string
	// FormatArray formats a dynamic array type (e.g., "[]T" or "std::vector<T>").
	FormatArray(elemType string) string
	// FormatFixedArray formats a fixed-size array type (e.g., "[4]byte" or
	// "std::array<T, 4>").
	FormatFixedArray(elemType string, size int64) string
	// FormatMap formats a map type (e.g., "map[K]V" or "std::unordered_map<K, V>").
	FormatMap(keyType, valType string) string
	// FallbackType returns the fallback type for unresolved references (e.g., "any" or
	// "void").
	FallbackType() string
}

// ImportResolver resolves import paths for cross-namespace type references.
type ImportResolver interface {
	// ResolveImport returns the import information for a type in a different namespace.
	// Returns (importPath, qualifier, shouldImport).
	ResolveImport(
		outputPath string,
		ctx *Context,
	) (importPath, qualifier string, shouldImport bool)
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
		return r.resolveNamedType(resolved, ctx)
	case resolution.AliasForm:
		return r.resolveAliasType(resolved, typeRef.TypeArgs, ctx)
	case resolution.UnionForm:
		return r.resolveNamedType(resolved, ctx)
	default:
		return r.Formatter.FallbackType()
	}
}

// resolveNamedType resolves a union or distinct type, which is referenced by bare
// name.
func (r *Resolver) resolveNamedType(resolved resolution.Type, ctx *Context) string {
	return r.qualify(
		ctx.GetTypeName(resolved), ctx.IsSameOutput(resolved),
		ctx.GetOutputPath(resolved), ctx,
	)
}

// resolveEnumType resolves an enum type, whose output may live in a dedicated enum
// path.
func (r *Resolver) resolveEnumType(resolved resolution.Type, ctx *Context) string {
	return r.qualify(
		ctx.GetTypeName(resolved), ctx.IsSameOutputEnum(resolved),
		ctx.GetEnumOutputPath(resolved), ctx,
	)
}

// resolveStructType resolves a struct type to a language-specific string.
func (r *Resolver) resolveStructType(
	resolved resolution.Type,
	typeArgs []resolution.TypeRef,
	ctx *Context,
) string {
	var params []resolution.TypeParam
	if form, ok := resolved.Form.(resolution.StructForm); ok {
		params = form.TypeParams
	}
	baseType := r.Formatter.FormatGeneric(
		ctx.GetTypeName(resolved), r.resolveTypeArgs(params, typeArgs, ctx),
	)
	return r.qualify(
		baseType, ctx.IsSameOutput(resolved), ctx.GetOutputPath(resolved), ctx,
	)
}

// resolveAliasType resolves an alias type to a language-specific string. Unlike
// expanding the target, this uses the alias name directly. The alias's own type params,
// not the target's, determine which type args the output language exposes.
func (r *Resolver) resolveAliasType(
	resolved resolution.Type,
	typeArgs []resolution.TypeRef,
	ctx *Context,
) string {
	var params []resolution.TypeParam
	if form, ok := resolved.Form.(resolution.AliasForm); ok {
		params = form.TypeParams
	}
	baseType := r.Formatter.FormatGeneric(
		ctx.GetTypeName(resolved), r.resolveTypeArgs(params, typeArgs, ctx),
	)
	return r.qualify(
		baseType, ctx.IsSameOutput(resolved), ctx.GetOutputPath(resolved), ctx,
	)
}

// resolveTypeArgs renders type arguments, skipping those bound to defaulted params
// when the language substitutes defaults.
func (r *Resolver) resolveTypeArgs(
	params []resolution.TypeParam,
	typeArgs []resolution.TypeRef,
	ctx *Context,
) []string {
	var args []string
	for i, arg := range typeArgs {
		if ctx.SubstituteDefaultedTypeParams && i < len(params) &&
			params[i].HasDefault() {
			continue
		}
		args = append(args, r.ResolveTypeRef(arg, ctx))
	}
	return args
}

// qualify renders a resolved type's name relative to the current output: bare
// when the type lives in the same output, qualified through the language's import
// machinery otherwise. An empty outputPath means the type has no home in this
// language.
func (r *Resolver) qualify(
	rendered string, sameOutput bool, outputPath string, ctx *Context,
) string {
	if sameOutput {
		return rendered
	}
	if outputPath == "" {
		return r.Formatter.FallbackType()
	}
	importPath, qualifier, shouldImport := r.ImportResolver.ResolveImport(
		outputPath, ctx,
	)
	if shouldImport {
		r.ImportAdder.AddImport("internal", importPath, qualifier)
	}
	return r.Formatter.FormatQualified(qualifier, rendered)
}
