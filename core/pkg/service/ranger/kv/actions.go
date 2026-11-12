// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// pair.
func (CreatePayload) Handle(Pair) (Pair, error) {
	return Pair{}, actions.CoreOnlyError("a pair create")
}

// Handle applies the set to the pair's value.
func (p SetPayload) Handle(
	pair Pair,
) (Pair, error) {
	return edit(pair, patch.Set(p).Apply)
}

// Handle applies the remove to the pair's value.
func (p RemovePayload) Handle(pair Pair) (Pair, error) {
	return edit(pair, patch.Remove(p).Apply)
}

// Handle applies the insert to the pair's value.
func (p InsertPayload) Handle(pair Pair) (Pair, error) {
	return edit(pair, patch.Insert(p).Apply)
}

// Handle applies the move to the pair's value.
func (p MovePayload) Handle(pair Pair) (Pair, error) {
	return edit(pair, patch.Move(p).Apply)
}

// document is the part of a pair that path actions edit.
type document struct {
	Value string `json:"value"`
}

// edit applies a path action to a copy of the pair's document. A document that gains a
// member it does not declare, such as a path to the pair's key, is rejected.
func edit(p Pair, apply func(map[string]any) error) (Pair, error) {
	d, err := patch.Edit(document{Value: p.Value}, apply)
	if err != nil {
		return Pair{}, err
	}
	p.Value = d.Value
	return p, nil
}
