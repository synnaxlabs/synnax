// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package query

import "fmt"

// Parameter is a type representing the key for a given option. Parameter should be unique for each option.
// If writing a new option, ensure that the key is unique, or else unpredictable behavior may occur.
type Parameter string

type Parameters map[Parameter]any

// Get returns the option with the given key. If the option is not set, returns false as its second argument.
// Unless an option is not required, it's recommended to use GetRequired instead.
func (p Parameters) Get(key Parameter) (any, bool) {
	v, ok := p[key]
	return v, ok
}

// GetRequired returns the option with the given key. Panics if the option is not set.
func (p Parameters) GetRequired(key Parameter) any {
	v, ok := p.Get(key)
	if !ok {
		panic(fmt.Sprintf("required option %s not set", key))
	}
	return v
}

// Set sets the option with the given key. Unless an option can be explicitly set multiple times, use SetOnce
// instead.
func (p Parameters) Set(key Parameter, value any) { p[key] = value }

// SetOnce sets the option with the given key. If the option is already set, it panics.
func (p Parameters) SetOnce(key Parameter, value any) {
	if _, ok := p[key]; ok {
		panic(fmt.Sprintf("option %s already set", key))
	}
	p[key] = value
}
