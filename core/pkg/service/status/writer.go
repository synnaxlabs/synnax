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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// Writer is used to create and update statuses within the DB. Every write touches both
// the status entry and its ontology resource, and the two land together only when the
// Writer holds a transaction spanning them.
type Writer struct {
	tx        gorp.Tx
	table     *gorp.Table[Key, Status[any]]
	indexes   indexes
	otgWriter ontology.Writer
}

// Set creates or updates a status. A status with an owner goes to the status its owner
// holds, whatever key it carries. Otherwise, Set uses the status's key, or a new UUID
// when the key is empty. An update keeps the stored labels and adds the ones s carries,
// and keeps the stored owner when s has none. On return, s holds what was stored.
func (w Writer) Set[D any](ctx context.Context, s *Status[D]) error {
	if err := w.validate(*s); err != nil {
		return err
	}
	if err := w.resolve(ctx, s); err != nil {
		return err
	}
	if err := w.table.NewCreate().Entry(new(s.ToAny())).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DefineResources(ctx, s.ResourceID())
}

// SetMany calls Set for each status, in order.
func (w Writer) SetMany[D any](ctx context.Context, statuses *[]Status[D]) error {
	if statuses == nil {
		return nil
	}
	for i := range *statuses {
		if err := w.Set(ctx, &(*statuses)[i]); err != nil {
			return err
		}
	}
	return nil
}

// KeyOf returns the key of the status that owner holds. It returns query.ErrNotFound
// when owner holds no status.
func (w Writer) KeyOf(ctx context.Context, owner resource.ID) (Key, error) {
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
		key, err := w.KeyOf(ctx, *s.Owner)
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
	s.Labels = lo.Union(stored.Labels, s.Labels)
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

// SetLabels adds labels to the status with the given key. With replace, labels become
// the status's only labels. It returns query.ErrNotFound if no status has the key.
func (w Writer) SetLabels(
	ctx context.Context,
	key Key,
	labels []label.Key,
	replace bool,
) error {
	return w.update(ctx, key, func(s *Status[json.RawMessage]) {
		if replace {
			s.Labels = slices.Clone(labels)
			return
		}
		s.Labels = lo.Union(s.Labels, labels)
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
	s, err := w.retrieveRaw(ctx, key)
	if err != nil {
		return err
	}
	change(&s)
	return w.table.NewCreate().Entry(new(s.ToAny())).Exec(ctx, w.tx)
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
	if err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Status[any]](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DeleteResources(ctx, ResourceIDs(keys)...)
}

func (w Writer) validate[D any](s Status[D]) error {
	v := validate.New("status.status")
	v.Positive("time", s.Time)
	v.NotEmptyString("variant", s.Variant)
	return v.Error()
}
