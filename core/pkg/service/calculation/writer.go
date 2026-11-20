// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Writer creates, changes, and deletes calculations in a transaction.
type Writer struct {
	tx         gorp.Tx
	table      *gorp.Table[Key, Calculation]
	channels   *channel.Service
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates the calculation and gives it a key when it has none. Its output must
// be a virtual channel that no other record writes. It returns an error wrapping
// validate.ErrValidation when the calculation is not valid or its output breaks that
// rule, and an error wrapping query.ErrNotFound when the output does not exist. The
// commit broadcasts a create action that holds the stored calculation.
func (w Writer) Create(ctx context.Context, c *Calculation) error {
	if c.Key == uuid.Nil() {
		c.Key = uuid.New()
	}
	if err := c.Validate(); err != nil {
		return err
	}
	if err := w.checkOutput(ctx, c.Output); err != nil {
		return err
	}
	if err := w.table.NewCreate().Exclusive().Entry(c).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, c.Key, NewCreateAction(CreatePayload{Calculation: *c}))
	return nil
}

// checkOutput fails when the channel with key does not exist, stores its samples, or
// has a record that writes it.
func (w Writer) checkOutput(ctx context.Context, key channel.Key) error {
	var ch channel.Channel
	if err := w.channels.NewRetrieve().
		Where(channel.MatchKeys(key)).
		Entry(&ch).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if !ch.Virtual {
		return outputError(
			"channel %s stores its samples, so no calculation writes it",
			ch,
		)
	}
	uses, err := w.channels.Uses(ctx, w.tx, []channel.Key{key})
	if err != nil {
		return err
	}
	for _, u := range uses[0] {
		if u.Direction == channel.DirectionWrite {
			return outputError("another record already writes channel %s", ch)
		}
	}
	return nil
}

func outputError(format string, ch channel.Channel) error {
	return validate.PathedError(
		errors.Wrapf(validate.ErrValidation, format, ch.Name),
		"output",
	)
}

// Apply applies the actions to the calculation with the given key, and the commit
// broadcasts them. It returns an error wrapping validate.ErrValidation when an action
// is a create or the result is not valid, and an error wrapping query.ErrNotFound when
// no calculation has the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	if err := w.table.
		NewUpdate().
		Where(gorp.MatchKeys[Key, Calculation](key)).
		ChangeErr(func(_ gorp.Context, c Calculation) (Calculation, error) {
			next, err := Reduce(c, acts...)
			if err != nil {
				return Calculation{}, err
			}
			if err = next.Validate(); err != nil {
				return Calculation{}, err
			}
			return next, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}

// Delete deletes the calculations with the given keys. A key with no calculation is not
// an error. The output of each one stays as a channel with no producer.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Calculation](keys...)).
		Exec(ctx, w.tx)
}
