// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

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

// Writer is used to create, update, and delete symbols within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method.
type Writer struct {
	tx         gorp.Tx
	table      *gorp.Table[Key, Symbol]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates the given symbol in s.Group, assigning a new key when s.Key is zero.
// The commit carries a create action that holds the stored symbol. Create returns an
// error wrapping validate.ErrValidation when a symbol with s.Key exists.
func (w Writer) Create(ctx context.Context, s *Symbol) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if err := validateScale(nil, *s); err != nil {
		return err
	}
	if s.Key == uuid.Nil() {
		s.Key = uuid.New()
	}
	if err := group.CheckMember(
		ctx,
		w.tx,
		s.Group,
		resource.TypeSchematicSymbol,
		nil,
	); err != nil {
		return err
	}
	if err := w.table.NewCreate().Exclusive().Entry(s).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, s.Key, NewCreateAction(CreatePayload{Symbol: *s}))
	return nil
}

// Apply applies the actions to the symbol with the given key and attaches them to the
// transaction of w, so its commit broadcasts them with the dispatch key of w. Apply
// returns an error wrapping validate.ErrValidation when an action is a create, the
// result fails to decode or validate, or the new group cannot hold a symbol, and an
// error wrapping query.ErrNotFound when no symbol has the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Symbol](key)).
		ChangeErr(func(_ gorp.Context, s Symbol) (Symbol, error) {
			next, err := Reduce(s, acts...)
			if err != nil {
				return Symbol{}, err
			}
			if err = next.Validate(); err != nil {
				return Symbol{}, err
			}
			if err = validateScale(&s, next); err != nil {
				return Symbol{}, err
			}
			if lo.FromPtr(next.Group) == lo.FromPtr(s.Group) {
				return next, nil
			}
			if err = group.CheckMember(
				ctx,
				w.tx,
				next.Group,
				resource.TypeSchematicSymbol,
				nil,
			); err != nil {
				return Symbol{}, err
			}
			return next, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}

// validateScale returns an error wrapping validate.ErrValidation when next has a scale
// that is not positive and differs from the scale of prev. A zero scale draws the
// symbol at size zero. A stored scale that the write keeps passes, so a stored zero
// does not block a rename or another edit. prev is nil for a new symbol.
func validateScale(prev *Symbol, next Symbol) error {
	if next.Data.Scale > 0 || (prev != nil && next.Data.Scale == prev.Data.Scale) {
		return nil
	}
	return errors.Wrapf(
		validate.ErrValidation,
		"data.scale: must be positive, got %v",
		next.Data.Scale,
	)
}

// CreateMany creates the given symbols. It returns an error wrapping
// validate.ErrValidation when a symbol with one of their keys exists.
func (w Writer) CreateMany(ctx context.Context, symbols *[]Symbol) error {
	for i := range *symbols {
		if err := w.Create(ctx, &(*symbols)[i]); err != nil {
			return err
		}
	}
	return nil
}

// SetGroup moves the symbols with keys into g, or out of their group when g is nil.
// Each move applies a set of the group.
func (w Writer) SetGroup(ctx context.Context, g *group.Key, keys ...Key) error {
	set := NewSetAction(SetPayload{Path: []string{"group"}, Value: g})
	for _, k := range keys {
		if err := w.Apply(ctx, k, set); err != nil {
			return err
		}
	}
	return nil
}

// Rename renames the symbol with the given key by applying a set of the name.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	return w.Apply(
		ctx,
		key,
		NewSetAction(SetPayload{Path: []string{"name"}, Value: name}),
	)
}

// Delete deletes the symbols with the given keys.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Symbol](keys...)).
		Exec(ctx, w.tx)
}
