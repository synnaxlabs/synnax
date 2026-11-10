// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
)

// Writer wraps a transaction to create, update, and delete labels.
type Writer struct {
	tx         gorp.Tx
	table      *gorp.Table[Key, Label]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates a new label, assigning it a unique key if one is not provided. It
// returns an error wrapping validate.ErrValidation when a label with the key exists.
// The commit carries a create action that holds the stored label.
func (w Writer) Create(ctx context.Context, l *Label) error {
	if l.Key == uuid.Nil() {
		l.Key = uuid.New()
	}
	if err := l.Validate(); err != nil {
		return err
	}
	if err := w.table.NewCreate().Exclusive().Entry(l).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(
		w.tx,
		l.Key,
		NewCreateAction(CreatePayload{Label: *l}),
	)
	return nil
}

// Apply applies the actions to the label with the given key and attaches them to the
// transaction of w, so its commit broadcasts them with the dispatch key of w. Apply
// returns an error wrapping validate.ErrValidation when an action is a create or the
// result fails to decode or validate, and an error wrapping query.ErrNotFound when no
// label has the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	if err := w.apply(ctx, key, acts); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}

// apply applies the actions to the label with the given key and stores the result.
func (w Writer) apply(ctx context.Context, key Key, acts []Action) error {
	return w.table.
		NewUpdate().
		Where(gorp.MatchKeys[Key, Label](key)).
		ChangeErr(func(_ gorp.Context, l Label) (Label, error) {
			next, err := Reduce(l, acts...)
			if err != nil {
				return Label{}, err
			}
			if err = next.Validate(); err != nil {
				return Label{}, err
			}
			return next, nil
		}).
		Exec(ctx, w.tx)
}

// CreateMany creates multiple labels in a single transaction. It returns an error
// wrapping validate.ErrValidation when a label with one of their keys exists.
func (w Writer) CreateMany(ctx context.Context, labels *[]Label) error {
	for i := range *labels {
		if err := w.Create(ctx, &(*labels)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Delete removes a label from the database. Delete is idempotent, and will not return
// an error if the label does not exist.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Label](keys...)).
		Exec(ctx, w.tx)
}
