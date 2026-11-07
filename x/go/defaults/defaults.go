// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package defaults fills schema defaults into values that come from a source that
// cannot tell an absent field from a zero one, such as a legacy format or a sparse Go
// literal. A decode already applies defaults, so decoded values never need it.
package defaults

import "reflect"

type setter interface{ SetDefaults() }

// grouped is a type with a default group: a set of fields whose defaults apply only
// when every member is zero.
type grouped interface {
	SetUngroupedDefaults()
	DefaultGroups()
}

// Fill returns v with every zero field replaced by its schema default. Each value whose
// type has a SetDefaults method, at any depth, starts from its defaults, and every
// non-zero field of v replaces the default. A default group takes its defaults only
// when all of its members are zero in v.
func Fill[T any](v T) T {
	out := reflect.New(reflect.TypeFor[T]()).Elem()
	fillFresh(out, reflect.ValueOf(&v).Elem())
	return *out.Addr().Interface().(*T)
}

// fillFresh sets the zero, addressable dst to its defaults and overlays src. A
// SetDefaults method sets the defaults of the nested structs too, so only a struct
// without one fills its struct fields itself.
func fillFresh(dst, src reflect.Value) {
	if g, ok := reflect.TypeAssert[grouped](dst.Addr()); ok {
		g.SetUngroupedDefaults()
		overlay(dst, src)
		g.DefaultGroups()
		return
	}
	s, ok := reflect.TypeAssert[setter](dst.Addr())
	if ok {
		s.SetDefaults()
	}
	if ok || dst.Kind() != reflect.Struct || opaque(dst.Type()) {
		overlay(dst, src)
		return
	}
	for i := range dst.NumField() {
		if f := dst.Field(i); f.Kind() == reflect.Struct {
			fillFresh(f, src.Field(i))
		} else {
			overlay(f, src.Field(i))
		}
	}
}

// overlay writes every non-zero part of src into dst.
func overlay(dst, src reflect.Value) {
	if src.IsZero() {
		return
	}
	switch src.Kind() {
	case reflect.Struct:
		if opaque(src.Type()) {
			dst.Set(src)
			return
		}
		for i := range src.NumField() {
			overlay(dst.Field(i), src.Field(i))
		}
	case reflect.Slice:
		out := reflect.MakeSlice(src.Type(), src.Len(), src.Len())
		for i := range src.Len() {
			fillFresh(out.Index(i), src.Index(i))
		}
		dst.Set(out)
	case reflect.Map:
		out := reflect.MakeMapWithSize(src.Type(), src.Len())
		for it := src.MapRange(); it.Next(); {
			v := reflect.New(src.Type().Elem()).Elem()
			fillFresh(v, it.Value())
			out.SetMapIndex(it.Key(), v)
		}
		dst.Set(out)
	case reflect.Pointer:
		v := reflect.New(src.Type().Elem())
		fillFresh(v.Elem(), src.Elem())
		dst.Set(v)
	case reflect.Interface:
		// A union holds its variant by value behind an interface.
		v := reflect.New(src.Elem().Type()).Elem()
		fillFresh(v, src.Elem())
		dst.Set(v)
	default:
		dst.Set(src)
	}
}

// opaque reports whether t has an unexported field, such as time.Time. Fill cannot
// reach that state field by field, so it copies such a value whole.
func opaque(t reflect.Type) bool {
	for f := range t.Fields() {
		if !f.IsExported() {
			return true
		}
	}
	return false
}

// Prefill sets the defaults of *v outside its default groups, as a decode does before
// it reads its input. It leaves a value whose type has no defaults as it is.
func Prefill[T any](v *T) {
	switch s := any(v).(type) {
	case grouped:
		s.SetUngroupedDefaults()
	case setter:
		s.SetDefaults()
	}
}

// FillGroups sets the defaults of each default group of *v whose members are all zero,
// as a decode does after it reads its input. It leaves a value whose type has no
// default group as it is.
func FillGroups[T any](v *T) {
	if g, ok := any(v).(grouped); ok {
		g.DefaultGroups()
	}
}
