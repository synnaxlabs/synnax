// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package color

import (
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// named color.
func (CreatePayload) Handle(Named) (Named, error) {
	return Named{}, actions.CoreOnlyError("a named color create")
}

// Handle applies the set to the named color.
func (p SetPayload) Handle(n Named) (Named, error) { return apply(n, patch.Set(p)) }

// Handle applies the remove to the named color.
func (p RemovePayload) Handle(n Named) (Named, error) {
	return apply(n, patch.Remove(p))
}

// Handle applies the insert to the named color.
func (p InsertPayload) Handle(n Named) (Named, error) {
	return apply(n, patch.Insert(p))
}

// Handle applies the move to the named color.
func (p MovePayload) Handle(n Named) (Named, error) { return apply(n, patch.Move(p)) }

func apply[A patch.Action](n Named, a A) (Named, error) {
	c, err := patch.Begin(a)
	if err == nil {
		err = n.Patch(c)
	}
	if err != nil {
		return Named{}, err
	}
	return n, nil
}
