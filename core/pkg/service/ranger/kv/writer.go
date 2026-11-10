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
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
)

// Writer is used to create, change, and delete key-value pairs.
type Writer struct {
	tx         gorp.Tx
	table      *gorp.Table[string, Pair]
	dispatcher actions.Dispatcher[string, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// SetMany sets multiple key-value pairs on the specified range.
func (w Writer) SetMany(ctx context.Context, pairs []Pair) error {
	if err := w.table.NewCreate().Entries(&pairs).Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, p := range pairs {
		w.dispatcher.Attach(w.tx, p.GorpKey(), NewCreateAction(CreatePayload{Pair: p}))
	}
	return nil
}

// Apply applies acts to the pair with the given Gorp key and attaches them to the
// transaction of w. It returns the actions that it attaches.
func (w Writer) Apply(
	ctx context.Context,
	key string,
	acts ...Action,
) ([]Action, error) {
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[string, Pair](key)).
		ChangeErr(func(_ gorp.Context, p Pair) (Pair, error) {
			next, err := Reduce(p, acts...)
			if err != nil {
				return Pair{}, err
			}
			return next, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return acts, nil
}

// Delete deletes a key-value pair from the specified range. Delete is
// idempotent and will not return an error if the key does not exist.
func (w Writer) Delete(ctx context.Context, rng ranger.Key, key string) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[string, Pair](Pair{Range: rng, Key: key}.GorpKey())).
		Exec(ctx, w.tx)
}
