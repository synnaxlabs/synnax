// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import "github.com/samber/lo"

// Find searches for a function by key. Returns the function and true if found,
// or zero value and false otherwise.
func (f Functions) Find(key string) (Function, bool) {
	return lo.Find(f, func(fn Function) bool { return fn.Key == key })
}

// Get returns the function with the given key. Panics if not found.
func (f Functions) Get(key string) Function { return lo.Must(f.Find(key)) }
