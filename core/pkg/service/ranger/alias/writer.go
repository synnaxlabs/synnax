// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alias

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// Writer is used to create, change, and delete aliases.
type Writer struct {
	tx         gorp.Tx
	channel    *channel.Service
	table      *gorp.Table[string, Alias]
	dispatcher actions.Dispatcher[string, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Set sets an alias for the given channel on the specified range.
func (w Writer) Set(
	ctx context.Context,
	rng ranger.Key,
	ch channel.Key,
	al string,
) error {
	exists, err := w.channel.NewRetrieve().
		Where(channel.MatchKeys(ch)).Exists(ctx, w.tx)
	if err != nil {
		return err
	}
	if !exists {
		return errors.Wrapf(
			query.ErrNotFound,
			"[alias] - cannot alias non-existent channel %s",
			ch,
		)
	}
	a := Alias{Range: rng, Channel: ch, Alias: al}
	if err = w.table.NewCreate().Entry(&a).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, a.GorpKey(), NewCreateAction(CreatePayload{Alias: a}))
	return nil
}

// Apply applies acts to the alias with the given Gorp key and attaches them to the
// transaction of w. It returns the actions that it attaches.
func (w Writer) Apply(
	ctx context.Context,
	key string,
	acts ...Action,
) ([]Action, error) {
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[string, Alias](key)).
		ChangeErr(func(_ gorp.Context, a Alias) (Alias, error) {
			next, err := Reduce(a, acts...)
			if err != nil {
				return Alias{}, err
			}
			return next, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return acts, nil
}

// Delete deletes the alias for the given channel on the specified range.
// Delete is idempotent and will not return an error if the alias does not
// exist.
func (w Writer) Delete(
	ctx context.Context,
	rng ranger.Key,
	ch channel.Key,
) error {
	return w.table.
		NewDelete().
		Where(gorp.MatchKeys[string, Alias](Alias{Range: rng, Channel: ch}.GorpKey())).
		Exec(ctx, w.tx)
}
