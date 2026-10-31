// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package instance gives each use of a generic struct over a plain value, such as a
// primitive, an enum, or a map, its own protobuf message. Protobuf has no generics, and
// a google.protobuf.Any holds only a message, so such a use cannot share the generic
// struct's message.
package instance

import (
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/resolution"
)

// Instance is a use of a generic struct whose type arguments are concrete and include
// at least one that an Any cannot hold.
type Instance struct {
	// Name is the message name: the type arguments' names, then the struct's name.
	// filter.Set<uuid> is UuidSet, and Box<map<string, Valve>> is StringValveMapBox. An
	// argument from another namespace takes that namespace's name first, so
	// channel.Key is ChannelKey.
	Name string
	// Generic is the generic struct.
	Generic resolution.Type
	// Ref is the use of Generic, with its type arguments. It holds no aliases, so a use
	// through an alias is the same instance as a use of its target.
	Ref resolution.TypeRef
	// Fields are the fields of Generic with the type arguments substituted.
	Fields []resolution.Field
}

// Of returns the instance that ref is, named for a use in namespace. It returns false
// when ref is not such a use, so it shares the generic struct's message.
func Of(
	ref resolution.TypeRef,
	namespace string,
	table *resolution.Table,
) (Instance, bool) {
	ref = dealias(ref, table)
	generic, ok := ref.Resolve(table)
	if !ok {
		return Instance{}, false
	}
	form, ok := generic.Form.(resolution.StructForm)
	if !ok || !form.IsGeneric() || len(ref.TypeArgs) == 0 ||
		lo.SomeBy(ref.TypeArgs, holdsTypeParam) ||
		lo.EveryBy(ref.TypeArgs, func(arg resolution.TypeRef) bool {
			return fitsAny(arg, table)
		}) {
		return Instance{}, false
	}
	args := resolution.TypeArgMap(form.TypeParams, ref.TypeArgs)
	return Instance{
		Name:    name(ref, namespace, table),
		Generic: generic,
		Ref:     ref,
		Fields: lo.Map(
			resolution.UnifiedFields(generic, table),
			func(f resolution.Field, _ int) resolution.Field {
				f.Type = resolution.SubstituteTypeRef(f.Type, args)
				return f
			},
		),
	}, true
}

func dealias(ref resolution.TypeRef, table *resolution.Table) resolution.TypeRef {
	if _, target, ok := resolution.Dealias(ref, table); ok {
		ref = target
	}
	ref.TypeArgs = lo.Map(
		ref.TypeArgs,
		func(arg resolution.TypeRef, _ int) resolution.TypeRef {
			return dealias(arg, table)
		},
	)
	return ref
}

func holdsTypeParam(ref resolution.TypeRef) bool {
	return ref.IsTypeParam() || lo.SomeBy(ref.TypeArgs, holdsTypeParam)
}

// fitsAny reports whether a google.protobuf.Any can hold ref: a struct, a union, a
// distinct type over one, any, or nil for no value.
func fitsAny(ref resolution.TypeRef, table *resolution.Table) bool {
	if ref.Name == "nil" || ref.Name == "any" {
		return true
	}
	target, _, ok := resolution.Dealias(ref, table)
	if !ok {
		return false
	}
	switch form := target.Form.(type) {
	case resolution.StructForm, resolution.UnionForm:
		return true
	case resolution.DistinctForm:
		return fitsAny(form.Base, table)
	}
	return false
}

func name(ref resolution.TypeRef, namespace string, table *resolution.Table) string {
	var b strings.Builder
	for _, arg := range ref.TypeArgs {
		b.WriteString(name(arg, namespace, table))
	}
	typ, ok := ref.Resolve(table)
	switch {
	case !ok:
		b.WriteString(lo.PascalCase(ref.Name))
	case len(ref.TypeArgs) > 0 || typ.Namespace == "" || typ.Namespace == namespace:
		b.WriteString(lo.PascalCase(typ.Name))
	default:
		b.WriteString(lo.PascalCase(typ.Namespace) + lo.PascalCase(typ.Name))
	}
	return b.String()
}
