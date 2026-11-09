// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"github.com/synnaxlabs/x/patch"
)

// Handle applies the set to the range.
func (p SetPayload) Handle(r Range) (Range, error) {
	return apply(r, patch.Set(p))
}

// Handle applies the remove to the range.
func (p RemovePayload) Handle(r Range) (Range, error) {
	return apply(r, patch.Remove(p))
}

// Handle applies the insert to the range.
func (p InsertPayload) Handle(r Range) (Range, error) {
	return apply(r, patch.Insert(p))
}

// Handle applies the move to the range.
func (p MovePayload) Handle(r Range) (Range, error) {
	return apply(r, patch.Move(p))
}

func apply[A patch.Action](r Range, a A) (Range, error) {
	c, err := patch.Begin(a)
	if err == nil {
		err = r.Patch(c)
	}
	if err != nil {
		return Range{}, err
	}
	return r, nil
}
