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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// Writer is used to create ranges within the DB.
type Writer struct {
	tx         gorp.Tx
	otgWriter  ontology.Writer
	table      *gorp.Table[Key, Range]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates or updates the given range, assigning a new key when it has none. An
// update keeps the stored labels and adds the ones r carries, and keeps the stored
// parent when r has none. The parent and labels must exist, and the parent must not be
// the range or one of its descendants. On return, r holds what was stored. After the
// transaction commits, Create broadcasts a create action that carries the stored range.
func (w Writer) Create(ctx context.Context, r *Range) error {
	if r.Key == uuid.Nil() {
		r.Key = uuid.New()
	}
	if err := r.Validate(); err != nil {
		return err
	}
	if err := w.validate(*r); err != nil {
		return err
	}
	var stored Range
	err := w.table.
		NewRetrieve().
		Where(gorp.MatchKeys[Key, Range](r.Key)).
		Entry(&stored).
		Exec(ctx, w.tx)
	if err == nil {
		r.Labels = lo.Union(stored.Labels, r.Labels)
		if r.Parent == nil {
			r.Parent = stored.Parent
		}
	} else if !errors.Is(err, query.ErrNotFound) {
		return err
	}
	if err = w.table.NewCreate().Entry(r).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.NotifyOnCommit(
		ctx,
		w.tx,
		r.Key,
		NewCreateAction(CreatePayload{Range: *r}),
	)
	return w.otgWriter.DefineResources(ctx, r.ResourceID())
}

// CreateMany calls Create for each range, in order.
func (w Writer) CreateMany(ctx context.Context, ranges *[]Range) error {
	for i := range *ranges {
		if err := w.Create(ctx, &(*ranges)[i]); err != nil {
			return err
		}
	}
	return nil
}

// SetLabels adds labels to the range with the given key. With replace, labels become
// the range's only labels. After the transaction commits, SetLabels broadcasts the
// change as a set action. It returns query.ErrNotFound if no range has the key.
func (w Writer) SetLabels(
	ctx context.Context,
	key Key,
	labels []label.Key,
	replace bool,
) error {
	return w.editLabels(ctx, key, func(stored []label.Key) []label.Key {
		if replace {
			return slices.Clone(labels)
		}
		return lo.Union(stored, labels)
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
	var stored Range
	if err := w.table.
		NewRetrieve().
		Where(gorp.MatchKeys[Key, Range](key)).
		Entry(&stored).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	act := NewSetAction(SetPayload{
		Path:  []string{"labels"},
		Value: edit(stored.Labels),
	})
	if err := w.apply(ctx, key, []Action{act}); err != nil {
		return err
	}
	w.dispatcher.NotifyOnCommit(ctx, w.tx, key, act)
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
// and the aliases and key-value pairs of every deleted range. Delete is idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	if len(keys) == 0 {
		return nil
	}
	if err := w.table.
		NewDelete().
		Where(gorp.MatchKeys[Key, Range](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DeleteResources(ctx, ResourceIDs(keys)...)
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
