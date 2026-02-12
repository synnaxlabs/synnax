// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"fmt"
	"reflect"
	"strings"
)

// CustomTypeName is an interface that allows types to provide their own custom type
// name. Types implementing this interface will return their custom name instead of the
// reflection-based name.
type CustomTypeName interface {
	// CustomTypeName returns the custom name of the type.
	CustomTypeName() string
}

// Name returns the type name of T. If T implements CustomTypeName interface, it returns
// the custom name. Otherwise, it returns the Go type name using reflection.
func Name[T any]() string {
	var t T
	if ct, ok := any(t).(CustomTypeName); ok {
		return ct.CustomTypeName()
	}
	typ := reflect.TypeOf((*T)(nil)).Elem()
	if n := typ.Name(); n != "" {
		return n
	}
	return typ.String()
}

// PluralName returns the plural form of the type name for T. It handles common English
// pluralization rules:
//   - Words ending in 'y' change to 'ies'
//   - Words ending in 's', 'x', 'z', 'ch', or 'sh' add 'es'
//   - All other words add 's'
func PluralName[T any]() string {
	name := Name[T]()
	// Handle special cases and irregular plurals
	switch {
	case len(name) == 0:
		return name
	case name[len(name)-1] == 'y':
		// Words ending in 'y' typically change to 'ies'
		return name[:len(name)-1] + "ies"
	case name[len(name)-1] == 's' ||
		name[len(name)-1] == 'x' ||
		name[len(name)-1] == 'z' ||
		strings.HasSuffix(name, "ch") ||
		strings.HasSuffix(name, "sh"):
		// Words ending in s, x, z, ch, sh add 'es'
		return name + "es"
	default:
		// Default case: just add 's'
		return name + "s"
	}
}

// PackageName extracts the package name from a reflect.Type. It returns the last
// component of the package path, which is typically the package name. For example,
// given a type from "github.com/user/project/pkg", it returns "pkg". If the type has no
// package path or the path is empty, it returns "unknown".
func PackageName(t reflect.Type) string {
	pkgPath := t.PkgPath()
	parts := strings.Split(pkgPath, "/")
	if len(parts) == 0 {
		return "unknown"
	}
	return parts[len(parts)-1]
}

// ValueName returns the semantic name of a type, including information about whether
// pointers are nil. For non-pointer types, it behaves the same as before.
func ValueName(v reflect.Value) string {
	if !v.IsValid() {
		return "nil"
	}
	t := v.Type()
	switch t.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return "*" + t.Elem().String() + " (nil)"
		}
		return "*" + t.Elem().String()
	case reflect.Slice:
		if v.IsNil() {
			return "[]" + t.Elem().String() + " (nil)"
		}
		return "[]" + t.Elem().String()
	case reflect.Map:
		if v.IsNil() {
			return fmt.Sprintf("map[%s]%s (nil)", t.Key().String(), t.Elem().String())
		}
		return fmt.Sprintf("map[%s]%s", t.Key().String(), t.Elem().String())
	case reflect.Chan:
		if v.IsNil() {
			var dir string
			switch t.ChanDir() {
			case reflect.SendDir:
				dir = "chan<- "
			case reflect.RecvDir:
				dir = "<-chan "
			default:
				dir = "chan "
			}
			return dir + t.Elem().String() + " (nil)"
		}
		var dir string
		switch t.ChanDir() {
		case reflect.SendDir:
			dir = "chan<- "
		case reflect.RecvDir:
			dir = "<-chan "
		default:
			dir = "chan "
		}
		return dir + t.Elem().String()
	case reflect.Func:
		if v.IsNil() {
			return "func (nil)"
		}
		return "func"
	case reflect.Interface:
		if v.IsNil() {
			if t.Name() != "" {
				return t.String()
			}
			return "any (nil)"
		}
		if t.Name() != "" {
			return t.String()
		}
		return "any"
	default:
		if t.Name() != "" {
			if t.PkgPath() != "" {
				return PackageName(t) + "." + t.Name()
			}
			return t.Name()
		}
		return t.String()
	}
}
