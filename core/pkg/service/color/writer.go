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

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Writer creates, changes, and deletes named colors in a transaction.
type Writer struct {
	tx         gorp.Tx
	table      *gorp.Table[Key, Named]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates a named color and gives it a key if it has none. It returns an error
// wrapping validate.ErrValidation when the color is not valid, its group cannot hold a
// named color, or a named color with the key exists. The commit broadcasts a create
// action that holds the stored color.
func (w Writer) Create(ctx context.Context, n *Named) error {
	if n.Key == uuid.Nil() {
		n.Key = uuid.New()
	}
	if err := check(*n); err != nil {
		return err
	}
	if err := group.CheckMember(
		ctx,
		w.tx,
		n.Group,
		resource.TypeColor,
		nil,
	); err != nil {
		return err
	}
	if err := w.table.NewCreate().Exclusive().Entry(n).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, n.Key, NewCreateAction(CreatePayload{Named: *n}))
	return nil
}

func check(n Named) error {
	if err := n.Validate(); err != nil {
		return err
	}
	if n.Position == nil {
		return nil
	}
	return validatePosition(*n.Position)
}

// CreateMany creates the named colors as Create does, stopping at the first error.
func (w Writer) CreateMany(ctx context.Context, named []Named) error {
	for i := range named {
		if err := w.Create(ctx, &named[i]); err != nil {
			return err
		}
	}
	return nil
}

// Apply applies the actions to the named color with the given key, and the commit
// broadcasts them. It returns an error wrapping validate.ErrValidation when an action
// is a create, the result is not valid, or the new group cannot hold a named color, and
// an error wrapping query.ErrNotFound when no named color has the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	if err := w.apply(ctx, key, acts); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}

func (w Writer) apply(ctx context.Context, key Key, acts []Action) error {
	return w.table.
		NewUpdate().
		Where(gorp.MatchKeys[Key, Named](key)).
		ChangeErr(func(_ gorp.Context, n Named) (Named, error) {
			next, err := Reduce(n, acts...)
			if err != nil {
				return Named{}, err
			}
			if err = check(next); err != nil {
				return Named{}, err
			}
			if lo.FromPtr(next.Group) == lo.FromPtr(n.Group) {
				return next, nil
			}
			if err = group.CheckMember(
				ctx,
				w.tx,
				next.Group,
				resource.TypeColor,
				nil,
			); err != nil {
				return Named{}, err
			}
			return next, nil
		}).
		Exec(ctx, w.tx)
}

// Delete deletes the named colors with the given keys. A key with no named color is not
// an error. It returns an error wrapping validate.ErrValidation for a built-in named
// color, and then deletes none of them.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Named](keys...)).
		Guard(func(_ gorp.Context, n Named) error {
			if isBuiltin(n.Key) {
				return errors.Wrapf(
					validate.ErrValidation,
					"cannot delete built-in color %s",
					n.Name,
				)
			}
			return nil
		}).
		Exec(ctx, w.tx)
}
