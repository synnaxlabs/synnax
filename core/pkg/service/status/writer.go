// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"context"
	"encoding/json"
	"slices"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// Writer is used to create and update statuses within the DB. A Writer with no
// transaction runs each call in a transaction of its own.
type Writer struct {
	svc     *Service
	tx      gorp.Tx
	table   *gorp.Table[Key, Status[any]]
	indexes indexes
}

// inTx calls f with w, or with a Writer on a transaction of its own when w has no
// transaction.
func (w Writer) inTx(ctx context.Context, f func(Writer) error) error {
	if db, ok := w.tx.(*gorp.DB); ok {
		return db.WithTx(ctx, func(tx gorp.Tx) error { return f(w.svc.NewWriter(tx)) })
	}
	return f(w)
}

// Set creates or updates a status. A status with an owner goes to the status its owner
// holds, whatever key it carries. Otherwise, Set uses the status's key, or a new UUID
// when the key is empty. A create stores the labels s carries. An update keeps the
// stored labels, which change only through AddLabels, ReplaceLabels, and RemoveLabels,
// and keeps the stored owner when s has none. On return, s holds what was stored.
func (w Writer) Set[D any](ctx context.Context, s *Status[D]) error {
	return w.inTx(ctx, func(w Writer) error {
		if err := w.validate(*s); err != nil {
			return err
		}
		if err := w.resolve(ctx, s); err != nil {
			return err
		}
		return w.table.NewCreate().Entry(new(s.ToAny())).Exec(ctx, w.tx)
	})
}

// SetMany calls Set for each status, in order.
func (w Writer) SetMany[D any](ctx context.Context, statuses *[]Status[D]) error {
	if statuses == nil {
		return nil
	}
	return w.inTx(ctx, func(w Writer) error {
		for i := range *statuses {
			if err := w.Set(ctx, &(*statuses)[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

// RenameOwned renames the status that owner holds, keeping every other field. It
// returns query.ErrNotFound if owner holds no status and a validation error if name is
// empty.
func (w Writer) RenameOwned(ctx context.Context, owner resource.ID, name string) error {
	return w.inTx(ctx, func(w Writer) error {
		key, err := w.ownedKey(ctx, owner)
		if err != nil {
			return err
		}
		return w.Rename(ctx, key, name)
	})
}

// DeleteOwned deletes the statuses that owners hold. DeleteOwned is idempotent.
func (w Writer) DeleteOwned(ctx context.Context, owners ...resource.ID) error {
	targets := make([]gorp.Target, len(owners))
	for i, o := range owners {
		targets[i] = resourceTarget(o)
	}
	return w.inTx(ctx, func(w Writer) error {
		keys, err := w.indexes.owner.Get(ctx, w.tx, targets...)
		if err != nil {
			return err
		}
		return w.Delete(ctx, keys...)
	})
}

func (w Writer) ownedKey(ctx context.Context, owner resource.ID) (Key, error) {
	keys, err := w.indexes.owner.Get(ctx, w.tx, resourceTarget(owner))
	if err != nil {
		return "", err
	}
	if len(keys) == 0 {
		return "", errors.Wrapf(query.ErrNotFound, "%s holds no status", owner)
	}
	return keys[0], nil
}

func (w Writer) resolve[D any](ctx context.Context, s *Status[D]) error {
	if s.Owner != nil {
		key, err := w.ownedKey(ctx, *s.Owner)
		if err == nil {
			s.Key = key
		} else if !errors.Is(err, query.ErrNotFound) {
			return err
		}
	}
	if s.Key == "" {
		s.Key = uuid.New().String()
		return nil
	}
	stored, err := w.retrieveRaw(ctx, s.Key)
	if errors.Is(err, query.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	s.Labels = stored.Labels
	if s.Owner == nil {
		s.Owner = stored.Owner
	}
	return nil
}

// Rename renames the status with the given key, keeping every other field. It returns
// query.ErrNotFound if no status with the key exists and a validation error if name is
// empty.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	v := validate.New("status.status")
	v.NotEmptyString("name", name)
	if err := v.Error(); err != nil {
		return err
	}
	return w.update(ctx, key, func(s *Status[json.RawMessage]) { s.Name = name })
}

// AddLabels adds labels to the status with the given key. It returns
// query.ErrNotFound if no status has the key.
func (w Writer) AddLabels(ctx context.Context, key Key, labels []label.Key) error {
	return w.update(ctx, key, func(s *Status[json.RawMessage]) {
		s.Labels = lo.Union(s.Labels, labels)
	})
}

// ReplaceLabels makes labels the only labels of the status with the given key. It
// returns query.ErrNotFound if no status has the key.
func (w Writer) ReplaceLabels(ctx context.Context, key Key, labels []label.Key) error {
	return w.update(ctx, key, func(s *Status[json.RawMessage]) {
		s.Labels = slices.Clone(labels)
	})
}

// RemoveLabels removes labels from the status with the given key. It returns
// query.ErrNotFound if no status has the key.
func (w Writer) RemoveLabels(ctx context.Context, key Key, labels []label.Key) error {
	return w.update(ctx, key, func(s *Status[json.RawMessage]) {
		s.Labels = lo.Without(s.Labels, labels...)
	})
}

// update rewrites the status with the given key. Raw details round-trip byte for byte,
// where decoding them as any would round numbers past float64 precision.
func (w Writer) update(
	ctx context.Context,
	key Key,
	change func(*Status[json.RawMessage]),
) error {
	return w.inTx(ctx, func(w Writer) error {
		s, err := w.retrieveRaw(ctx, key)
		if err != nil {
			return err
		}
		change(&s)
		return w.table.NewCreate().Entry(new(s.ToAny())).Exec(ctx, w.tx)
	})
}

func (w Writer) retrieveRaw(
	ctx context.Context,
	key Key,
) (Status[json.RawMessage], error) {
	var s Status[json.RawMessage]
	err := gorp.NewRetrieve[Key, Status[json.RawMessage]]().
		Where(gorp.MatchKeys[Key, Status[json.RawMessage]](key)).
		Entry(&s).
		Exec(ctx, w.tx)
	return s, err
}

// Delete deletes the statuses with the given keys. Delete is idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Status[any]](keys...)).
		Exec(ctx, w.tx)
}

func (w Writer) validate[D any](s Status[D]) error {
	v := validate.New("status.status")
	v.Positive("time", s.Time)
	v.NotEmptyString("variant", s.Variant)
	return v.Error()
}
