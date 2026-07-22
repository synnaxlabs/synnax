// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"encoding/json"

	"github.com/samber/lo"
)

var _ json.Marshaler = (Params)(nil)

// MarshalJSON implements the json.Marshal interface.
func (p Params) MarshalJSON() ([]byte, error) {
	if p == nil {
		return json.Marshal([]Param{})
	}
	type params []Param
	return json.Marshal(params(p))
}

// Get retrieves a parameter by name. Returns the parameter and true if found, otherwise
// returns a zero Param and false.
func (p Params) Get(name string) (Param, bool) {
	return lo.Find(p, func(param Param) bool { return param.Name == name })
}

// GetIndex returns the index of a parameter by name. Returns -1 if not found.
func (p Params) GetIndex(name string) int {
	_, i, _ := lo.FindIndexOf(p, func(param Param) bool {
		return param.Name == name
	})
	return i
}

// Has returns true if a parameter with the given name exists.
func (p Params) Has(name string) bool { _, ok := p.Get(name); return ok }

// Positional returns the params bindable by position: all params except the trigger,
// which the upstream feeds rather than the call site.
func (p Params) Positional(trigger string) Params {
	if trigger == "" {
		return p
	}
	return lo.Filter(p, func(param Param, _ int) bool { return param.Name != trigger })
}

// ValueMap returns a map of parameter names to their values.
func (p Params) ValueMap() map[string]any {
	return lo.SliceToMap(p, func(param Param) (string, any) {
		return param.Name, param.Value
	})
}

// RequiredCount returns the number of required (non-optional) parameters. A parameter
// is optional if its Value field is non-nil (has a default).
func (p Params) RequiredCount() int {
	count := 0
	for _, param := range p {
		if param.Value == nil {
			count++
		}
	}
	return count
}
