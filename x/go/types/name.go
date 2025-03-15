// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"reflect"
	"strings"
)

// CustomTypeName is an interface that allows types to provide their own custom type name.
// Types implementing this interface will return their custom name instead of the
// reflection-based name.
type CustomTypeName interface {
	// CustomTypeName returns the custom name of the type.
	CustomTypeName() string
}

// Name returns the type name of T. If T implements CustomTypeName interface,
// it returns the custom name. Otherwise, it returns the Go type name using reflection.
func Name[T any]() string {
	var t T
	if ct, ok := any(t).(CustomTypeName); ok {
		return ct.CustomTypeName()
	}
	return reflect.TypeOf(*new(T)).Name()
}

// PluralName returns the plural form of the type name for T.
// It handles common English pluralization rules:
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
