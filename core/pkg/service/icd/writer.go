// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
)

// Writer creates, updates, and deletes ICDs within a transaction.
type Writer struct {
	tx         gorp.Tx
	table      *gorp.Table[Key, ICD]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates the ICD. Create assigns keys to the ICD, its messages, and their
// fields where absent. It returns a validation error when an ICD with its key exists,
// and a path-scoped validation error when the ICD breaks a rule of its schema. Problems
// that Check finds do not stop the save. Create attaches a create action that carries
// the stored ICD to the transaction, so its commit broadcasts the action.
func (w Writer) Create(ctx context.Context, ic *ICD) error {
	if ic.Key == uuid.Nil() {
		ic.Key = uuid.New()
	}
	assignKeys(ic)
	if err := ic.Validate(); err != nil {
		return err
	}
	if err := w.table.NewCreate().Exclusive().Entry(ic).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, ic.Key, NewCreateAction(CreatePayload{ICD: *ic}))
	return nil
}

// CreateMany creates each of the given ICDs as Create does.
func (w Writer) CreateMany(ctx context.Context, ics *[]ICD) error {
	for i := range *ics {
		if err := w.Create(ctx, &(*ics)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Import replaces the messages of the ICD with the given key by those parsed from data
// in the given format. An imported message keeps the key of the message with its name,
// and each of its fields keeps the key of that message's field with its name. Problems
// that Check finds do not stop the import. Import attaches a create action that carries
// the stored ICD to the transaction and returns the ICD. It returns a path-scoped
// validation error when data does not parse or the result breaks a rule of the ICD's
// schema, and an error wrapping query.ErrNotFound when no ICD has the key.
func (w Writer) Import(
	ctx context.Context,
	key Key,
	format Format,
	data []byte,
) (ICD, error) {
	messages, err := parse(format, data)
	if err != nil {
		return ICD{}, err
	}
	var ic ICD
	if err = w.table.
		NewUpdate().
		Where(gorp.MatchKeys[Key, ICD](key)).
		ChangeErr(func(_ gorp.Context, existing ICD) (ICD, error) {
			ic = merge(existing, messages)
			assignKeys(&ic)
			if err := ic.Validate(); err != nil {
				return ICD{}, err
			}
			return ic, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return ICD{}, err
	}
	w.dispatcher.Attach(w.tx, key, NewCreateAction(CreatePayload{ICD: ic}))
	return ic, nil
}

// Delete deletes the ICDs with the given keys. Delete is idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, ICD](keys...)).
		Exec(ctx, w.tx)
}

// Apply applies the actions to the ICD with the given key, stores the result, and
// attaches the actions to the transaction, so its commit broadcasts them. Problems that
// Check finds do not stop it. It returns an error wrapping validate.ErrValidation when
// an action is a create or the ICD cannot hold its value, a path-scoped validation
// error when the result breaks a rule of the ICD's schema, and an error wrapping
// query.ErrNotFound when no ICD has the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	if err := w.table.
		NewUpdate().
		Where(gorp.MatchKeys[Key, ICD](key)).
		ChangeErr(func(_ gorp.Context, ic ICD) (ICD, error) {
			next, err := Reduce(ic, acts...)
			if err != nil {
				return ICD{}, err
			}
			if err = next.Validate(); err != nil {
				return ICD{}, err
			}
			return next, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}

// assignKeys gives a new key to every message and field that has none. It skips text
// fields with no encoding and JSON fields with no type, which Check rejects.
func assignKeys(ic *ICD) {
	for i := range ic.Messages {
		m := &ic.Messages[i]
		if m.Key == uuid.Nil() {
			m.Key = uuid.New()
		}
		m.Payload.UpdateFieldBases(func(fb *BaseField) {
			if fb.Key == uuid.Nil() {
				fb.Key = uuid.New()
			}
		})
	}
}
