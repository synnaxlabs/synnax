// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package view

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Writer is used to create and update views within the DB.
type Writer struct {
	tx         gorp.Tx
	table      *gorp.Table[Key, View]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates a view, assigning a new key when the key of view is zero. The commit
// carries a create action that holds the stored view. Create returns an error wrapping
// validate.ErrValidation when a view with the key exists.
func (w Writer) Create(ctx context.Context, view *View) error {
	if err := w.validate(*view); err != nil {
		return err
	}
	if view.GorpKey() == uuid.Nil() {
		*view = view.WithKey(uuid.New())
	}
	if err := w.table.NewCreate().Exclusive().Entry(view).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(
		w.tx,
		view.GorpKey(),
		NewCreateAction(CreatePayload{View: *view}),
	)
	return nil
}

// Apply applies the actions to the view with the given key and attaches them to the
// transaction of w, so its commit broadcasts them with the dispatch key of w. Apply
// returns an error wrapping validate.ErrValidation when an action is a create or the
// result fails to decode or validate, and an error wrapping query.ErrNotFound when no
// view has the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, View](key)).
		ChangeErr(func(_ gorp.Context, v View) (View, error) {
			next, err := Reduce(v, acts...)
			if err != nil {
				return View{}, err
			}
			if err = next.Validate(); err != nil {
				return View{}, err
			}
			if err = w.validate(next); err != nil {
				return View{}, err
			}
			return next, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}

// CreateMany creates the given views. It returns an error wrapping
// validate.ErrValidation when a view with one of their keys exists.
func (w Writer) CreateMany(ctx context.Context, views *[]View) error {
	for i := range *views {
		if err := w.Create(ctx, &(*views)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Rename renames the view with the given key by applying a set of the name. It returns
// query.ErrNotFound if no view with the key exists and a validation error if name is
// empty.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	return w.Apply(
		ctx,
		key,
		NewSetAction(SetPayload{Path: []string{"name"}, Value: name}),
	)
}

// Delete deletes the views with the given keys. Delete is idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().Where(gorp.MatchKeys[Key, View](keys...)).
		Exec(ctx, w.tx)
}

func (w Writer) validate(v View) error {
	vld := validate.New("view.view")
	vld.NotEmptyString("name", v.Base().Name)
	vld.NotEmptyString("type", v.Type())
	if err := vld.Error(); err != nil {
		return err
	}
	var err error
	switch variant := v.Variant.(type) {
	case RangeView:
		if variant.Where != nil {
			_, err = ranger.MatchWhere(*variant.Where)
		}
	case StatusView:
		if variant.Where != nil {
			_, err = status.MatchWhere[any](*variant.Where)
		}
	default:
		return errors.Newf("view %s has the unknown variant %T", v.Base().Key, variant)
	}
	return validate.PathedError(err, "where")
}
