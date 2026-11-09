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
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

// storedDetails carries the details of a stored status unchanged in either stored
// encoding. Decoding JSON details as any would round numbers past float64 precision.
type storedDetails struct{ value any }

func (d storedDetails) MarshalJSON() ([]byte, error) { return json.Marshal(d.value) }

func (d *storedDetails) UnmarshalJSON(b []byte) error {
	d.value = json.RawMessage(bytes.Clone(b))
	return nil
}

func (d storedDetails) EncodeMsgpack(e *msgpack.Encoder) error {
	return e.Encode(d.value)
}

func (d *storedDetails) DecodeMsgpack(dec *msgpack.Decoder) (err error) {
	d.value, err = dec.DecodeInterface()
	return err
}

// Writer is used to create and update statuses within the DB. Each create or update
// attaches a set action that holds the stored status.
type Writer struct {
	tx         gorp.Tx
	table      *gorp.Table[Key, Status[any]]
	indexes    indexes
	dispatcher actions.Dispatcher[Key, Action]
}

// Set creates or updates a status. A status with an owner goes to the status its owner
// holds, whatever key it carries. Otherwise, Set uses the status's key, or a new UUID
// when the key is empty. A create stores the labels and owner s carries. An update
// keeps the stored owner and labels. Labels change only through AddLabels,
// ReplaceLabels, and RemoveLabels. On return, s holds what was stored.
func (w Writer) Set[D any](ctx context.Context, s *Status[D]) error {
	if err := w.validate(*s); err != nil {
		return err
	}
	if err := w.Resolve(ctx, s); err != nil {
		return err
	}
	return w.write(ctx, s.ToAny())
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

// RenameOwned renames the status that owner holds, keeping every other field. It
// returns query.ErrNotFound if owner holds no status and a validation error if name is
// empty.
func (w Writer) RenameOwned(ctx context.Context, owner resource.ID, name string) error {
	key, err := w.ownedKey(ctx, owner)
	if err != nil {
		return err
	}
	return w.Rename(ctx, key, name)
}

// DeleteOwned deletes the statuses that owners hold. DeleteOwned is idempotent.
func (w Writer) DeleteOwned(ctx context.Context, owners ...resource.ID) error {
	targets := make([]gorp.Target, len(owners))
	for i, o := range owners {
		targets[i] = resourceTarget(o)
	}
	keys, err := w.indexes.owner.Get(ctx, w.tx, targets...)
	if err != nil {
		return err
	}
	return w.Delete(ctx, keys...)
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

// Resolve sets the key, labels, and owner of s to the ones Set would write, without
// writing anything. Calling it again on the result changes nothing.
func (w Writer) Resolve[D any](ctx context.Context, s *Status[D]) error {
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
	s.Owner = stored.Owner
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
	return w.update(ctx, key, func(s *Status[storedDetails]) { s.Name = name })
}

// AddLabels adds labels to the status with the given key. It returns
// query.ErrNotFound if no status has the key.
func (w Writer) AddLabels(ctx context.Context, key Key, labels []label.Key) error {
	return w.update(ctx, key, func(s *Status[storedDetails]) {
		s.Labels = lo.Union(s.Labels, labels)
	})
}

// ReplaceLabels makes labels the only labels of the status with the given key. It
// returns query.ErrNotFound if no status has the key.
func (w Writer) ReplaceLabels(ctx context.Context, key Key, labels []label.Key) error {
	return w.update(ctx, key, func(s *Status[storedDetails]) {
		s.Labels = slices.Clone(labels)
	})
}

// RemoveLabels removes labels from the status with the given key. It returns
// query.ErrNotFound if no status has the key.
func (w Writer) RemoveLabels(ctx context.Context, key Key, labels []label.Key) error {
	return w.update(ctx, key, func(s *Status[storedDetails]) {
		s.Labels = lo.Without(s.Labels, labels...)
	})
}

// update rewrites the status with the given key. Its details round-trip unchanged in
// either stored encoding.
func (w Writer) update(
	ctx context.Context,
	key Key,
	change func(*Status[storedDetails]),
) error {
	s, err := w.retrieveRaw(ctx, key)
	if err != nil {
		return err
	}
	change(&s)
	return w.write(ctx, s.ToAny())
}

// write stores s and attaches a set action that holds it.
func (w Writer) write(ctx context.Context, s Status[any]) error {
	if err := w.table.NewCreate().Entry(&s).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, s.Key, NewSetAction(SetPayload{Status: s}))
	return nil
}

func (w Writer) retrieveRaw(
	ctx context.Context,
	key Key,
) (Status[storedDetails], error) {
	var s Status[storedDetails]
	err := gorp.NewRetrieve[Key, Status[storedDetails]]().
		Where(gorp.MatchKeys[Key, Status[storedDetails]](key)).
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
