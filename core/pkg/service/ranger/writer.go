// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"context"
	"slices"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Writer is used to create ranges within the DB.
type Writer struct {
	tx         gorp.Tx
	table      *gorp.Table[Key, Range]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates the given range, assigning a new key when it has none, and stores the
// labels r carries. It sets HasChildren to false, since a new range has no children.
// The parent and labels must exist. It returns an error wrapping validate.ErrValidation
// when a range with the key exists. After the transaction commits, Create broadcasts a
// create action that carries the stored range.
func (w Writer) Create(ctx context.Context, r *Range) error {
	if r.Key == uuid.Nil() {
		r.Key = uuid.New()
	}
	r.HasChildren = false
	if err := r.Validate(); err != nil {
		return err
	}
	if err := w.validate(*r); err != nil {
		return err
	}
	if err := w.table.NewCreate().Exclusive().Entry(r).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(
		w.tx,
		r.Key,
		NewCreateAction(CreatePayload{Range: *r}),
	)
	return nil
}

// CreateMany calls Create for each range, in order. It sets HasChildren on each range
// that another range in ranges names as its parent.
func (w Writer) CreateMany(ctx context.Context, ranges *[]Range) error {
	parents := make(set.Set[Key], len(*ranges))
	for i := range *ranges {
		r := &(*ranges)[i]
		if err := w.Create(ctx, r); err != nil {
			return err
		}
		if r.Parent != nil {
			parents.Add(*r.Parent)
		}
	}
	for i := range *ranges {
		(*ranges)[i].HasChildren = parents.Contains((*ranges)[i].Key)
	}
	return nil
}

// AddLabels adds labels to the range with the given key. After the transaction
// commits, AddLabels broadcasts the change as a set action. It returns
// query.ErrNotFound if no range has the key.
func (w Writer) AddLabels(ctx context.Context, key Key, labels []label.Key) error {
	return w.editLabels(ctx, key, func(stored []label.Key) []label.Key {
		return lo.Union(stored, labels)
	})
}

// ReplaceLabels makes labels the only labels of the range with the given key. After
// the transaction commits, ReplaceLabels broadcasts the change as a set action. It
// returns query.ErrNotFound if no range has the key.
func (w Writer) ReplaceLabels(ctx context.Context, key Key, labels []label.Key) error {
	return w.editLabels(ctx, key, func([]label.Key) []label.Key {
		return slices.Clone(labels)
	})
}

// RemoveLabels removes labels from the range with the given key. After the transaction
// commits, RemoveLabels broadcasts the change as a set action. It returns
// query.ErrNotFound if no range has the key.
func (w Writer) RemoveLabels(ctx context.Context, key Key, labels []label.Key) error {
	return w.editLabels(ctx, key, func(stored []label.Key) []label.Key {
		return lo.Without(stored, labels...)
	})
}

func (w Writer) editLabels(
	ctx context.Context,
	key Key,
	edit func([]label.Key) []label.Key,
) error {
	var labels []label.Key
	if err := w.table.
		NewUpdate().
		Where(gorp.MatchKeys[Key, Range](key)).
		Change(func(_ gorp.Context, r Range) Range {
			labels = edit(r.Labels)
			r.Labels = labels
			return r
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, labelsAction(labels))
	return nil
}

// labelsAction returns the action that sets the labels of a range to labels.
func labelsAction(labels []label.Key) Action {
	return NewSetAction(SetPayload{Path: []string{"labels"}, Value: labels})
}

// Apply applies the actions to the range with the given key and attaches them to the
// transaction of w, so its commit broadcasts them with the dispatch key of w. Apply
// returns an error wrapping validate.ErrValidation when an action is a create or the
// result fails to decode or validate, and an error wrapping query.ErrNotFound when no
// range has the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	if err := w.apply(ctx, key, acts); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}

// apply applies the actions to the range with the given key and stores the result.
func (w Writer) apply(ctx context.Context, key Key, acts []Action) error {
	return w.table.
		NewUpdate().
		Where(gorp.MatchKeys[Key, Range](key)).
		ChangeErr(func(_ gorp.Context, r Range) (Range, error) {
			next, err := Reduce(r, acts...)
			if err != nil {
				return Range{}, err
			}
			if err = next.Validate(); err != nil {
				return Range{}, err
			}
			if err = w.validate(next); err != nil {
				return Range{}, err
			}
			return next, nil
		}).
		Exec(ctx, w.tx)
}

// Delete deletes the ranges with the given keys, along with each range nested in them
// and the key-value pairs of every deleted range. Delete is idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	if len(keys) == 0 {
		return nil
	}
	return w.table.
		NewDelete().
		Where(gorp.MatchKeys[Key, Range](keys...)).
		Exec(ctx, w.tx)
}

func (w Writer) validate(r Range) error {
	v := validate.New("ranger.range")
	v.NotNil("key", r.Key)
	v.NonZero("time_range.start", r.TimeRange.Start)
	v.NonZero("time_range.end", r.TimeRange.End)
	v.Ternary(
		"time_range",
		r.TimeRange.Start.After(r.TimeRange.End),
		"time_range.start cannot be after time_range.end",
	)
	return v.Error()
}
