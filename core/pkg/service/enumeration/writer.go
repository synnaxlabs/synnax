// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package enumeration

import (
	"context"
	"strconv"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Writer creates, changes, and deletes enumerations in a transaction.
type Writer struct {
	tx         gorp.Tx
	table      *gorp.Table[Key, Enumeration]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

func (w Writer) create(ctx context.Context, e *Enumeration) error {
	if e.Key == uuid.Nil() {
		e.Key = uuid.New()
	}
	if err := check(*e); err != nil {
		return err
	}
	if err := w.table.NewCreate().Exclusive().Entry(e).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, e.Key, NewCreateAction(CreatePayload{Enumeration: *e}))
	return nil
}

func check(e Enumeration) error {
	if err := e.Validate(); err != nil {
		return err
	}
	keys := make(set.Set[string], len(e.Entries))
	values := make(set.Set[float64], len(e.Entries))
	for i, entry := range e.Entries {
		path := []string{"entries", strconv.Itoa(i)}
		if keys.Contains(entry.Key) {
			return validate.PathedError(errors.Wrapf(
				validate.ErrValidation,
				"another entry has the key %q",
				entry.Key,
			), append(path, "key")...)
		}
		if values.Contains(entry.Value) {
			return validate.PathedError(errors.Wrapf(
				validate.ErrValidation,
				"another entry has the value %v",
				entry.Value,
			), append(path, "value")...)
		}
		keys.Add(entry.Key)
		values.Add(entry.Value)
	}
	return nil
}

// CreateMany creates the enumerations and gives each one with no key a key, stopping at
// the first error. It returns an error wrapping validate.ErrValidation when an
// enumeration is not valid, two of its entries share a key or a value, or an
// enumeration with its key exists. The commit broadcasts a create action that holds
// each stored enumeration.
func (w Writer) CreateMany(ctx context.Context, enumerations []Enumeration) error {
	for i := range enumerations {
		if err := w.create(ctx, &enumerations[i]); err != nil {
			return err
		}
	}
	return nil
}

// Apply applies the actions to the enumeration with the given key, and the commit
// broadcasts them. It returns an error wrapping validate.ErrValidation when an action
// is a create or the result fails the checks of CreateMany, and an error wrapping
// query.ErrNotFound when no enumeration has the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	if err := w.table.
		NewUpdate().
		Where(gorp.MatchKeys[Key, Enumeration](key)).
		ChangeErr(func(_ gorp.Context, e Enumeration) (Enumeration, error) {
			next, err := Reduce(e, acts...)
			if err != nil {
				return Enumeration{}, err
			}
			if err = check(next); err != nil {
				return Enumeration{}, err
			}
			return next, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}

// Replace moves each record that uses the enumeration with key from to the enumeration
// with key to, then deletes from. It returns an error wrapping validate.ErrValidation
// when from is built in or equals to, and one wrapping query.ErrNotFound when from or
// to does not exist.
func (w Writer) Replace(ctx context.Context, from, to Key) error {
	if err := w.table.Retarget(ctx, w.tx, from, to); err != nil {
		return err
	}
	return w.Delete(ctx, from)
}

// Delete deletes the enumerations with the given keys. A key with no enumeration is not
// an error. It returns an error wrapping validate.ErrValidation for a built-in
// enumeration or one that a record uses, and then deletes none of them.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Enumeration](keys...)).
		Guard(func(_ gorp.Context, e Enumeration) error {
			if isBuiltin(e.Key) {
				return errors.Wrapf(
					validate.ErrValidation,
					"cannot delete built-in enumeration %s",
					e.Name,
				)
			}
			return nil
		}).
		Exec(ctx, w.tx)
}
