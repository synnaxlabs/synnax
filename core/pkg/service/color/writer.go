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
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
)

// Writer creates, changes, and deletes named colors in a transaction.
type Writer struct {
	tx    gorp.Tx
	table *gorp.Table[Key, Named]
}

// Create creates a named color and gives it a key if it has none. It returns an error
// wrapping validate.ErrValidation when the color is not valid or a named color with the
// key exists.
func (w Writer) Create(ctx context.Context, n *Named) error {
	if n.Key == uuid.Nil() {
		n.Key = uuid.New()
	}
	if err := n.Validate(); err != nil {
		return err
	}
	return w.table.NewCreate().Exclusive().Entry(n).Exec(ctx, w.tx)
}

// Apply applies the actions to the named color with the given key. It returns an error
// wrapping validate.ErrValidation when an action is a create or the result is not
// valid, and an error wrapping query.ErrNotFound when no named color has the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	return w.table.
		NewUpdate().
		Where(gorp.MatchKeys[Key, Named](key)).
		ChangeErr(func(_ gorp.Context, n Named) (Named, error) {
			next, err := Reduce(n, acts...)
			if err != nil {
				return Named{}, err
			}
			if err = next.Validate(); err != nil {
				return Named{}, err
			}
			return next, nil
		}).
		Exec(ctx, w.tx)
}

// Delete deletes the named colors with the given keys. A key with no named color is not
// an error.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Named](keys...)).
		Exec(ctx, w.tx)
}
