// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"slices"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	label "github.com/synnaxlabs/synnax/pkg/service/label/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	v1 "github.com/synnaxlabs/synnax/pkg/service/status/versions/v1"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// namespace is the UUIDv5 namespace of the keys that MigrateKey derives. Changing it
// gives each node a different key for the same status.
var namespace = uuid.MustParse("4a3c5e0e-6d0b-5f43-9a51-3c7e1f2b8d64")

const labeledBy ontology.RelationshipType = "labeled_by"

var ownerTypes = []resource.Type{
	resource.TypeRack,
	resource.TypeTask,
	resource.TypeDevice,
	resource.TypeChannel,
}

// MigrateKey returns the key and owner that the migration gives a status stored under
// key. A key that is a rack, task, device, or channel resource ID becomes a UUIDv5 of
// the key, owned by that resource. Any other key is returned unchanged, with no owner.
func MigrateKey(key Key) (Key, *resource.ID) {
	id, err := resource.ParseID(key)
	if err != nil || !slices.Contains(ownerTypes, id.Type) {
		return key, nil
	}
	h := sha1.New()
	h.Write(namespace[:])
	h.Write([]byte(key))
	var u uuid.UUID
	copy(u[:], h.Sum(nil))
	u[6] = (u[6] & 0x0f) | 0x50
	u[8] = (u[8] & 0x3f) | 0x80
	return u.String(), &id
}

// Migration moves each status to its MigrateKey key and owner, and moves its labels
// from labeled_by ontology relationships into the labels field. It deletes a status
// whose owner has no ontology resource, as an owner delete now deletes its status, and
// drops labels with no ontology resource. It removes the owner's key from the details,
// such as details.task on a task status. The ontology resource of each moved status,
// and each relationship that touches it, follow the new key.
var Migration = migrate.New("status_owner_labels", migrateStatuses)

type stored = Status[json.RawMessage]

func migrateStatuses(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
	old, err := collect(ctx, gorp.WrapReader[Key, v1.Status[json.RawMessage]](tx))
	if err != nil {
		return err
	}
	rels, err := collect(ctx, gorp.WrapReader[string, ontology.Relationship](tx))
	if err != nil {
		return err
	}
	var (
		labels    = make(map[resource.ID][]label.Key)
		resources = gorp.WrapReader[string, ontology.Resource](tx)
		relWriter = gorp.WrapWriter[string, ontology.Relationship](tx)
	)
	for _, rel := range rels {
		if rel.Type != labeledBy || rel.From.Type != resource.TypeStatus {
			continue
		}
		if err = relWriter.Delete(ctx, rel.GorpKey()); err != nil {
			return err
		}
		key, err := uuid.Parse(rel.To.Key)
		if err != nil {
			return errors.Wrapf(err, "label of status %s", rel.From.Key)
		}
		found, err := exists(ctx, resources, rel.To)
		if err != nil {
			return err
		}
		if found {
			labels[rel.From] = append(labels[rel.From], key)
		}
	}
	var (
		writer  = gorp.WrapWriter[Key, stored](tx)
		moved   = make(map[resource.ID]resource.ID)
		dropped = make(map[resource.ID]bool)
	)
	for _, o := range old {
		id := resource.ID{Type: resource.TypeStatus, Key: o.Key}
		key, owner := MigrateKey(o.Key)
		if owner != nil {
			found, err := exists(ctx, resources, *owner)
			if err != nil {
				return err
			}
			if !found {
				dropped[id] = true
				if err = writer.Delete(ctx, o.Key); err != nil {
					return err
				}
				continue
			}
		}
		next, err := autoMigrateStatus(ctx, o)
		if err != nil {
			return err
		}
		if owner != nil {
			if next.Details, err = dropField(
				next.Details,
				string(owner.Type),
			); err != nil {
				return errors.Wrapf(err, "details of status %s", o.Key)
			}
		}
		next.Key, next.Owner, next.Labels = key, owner, labels[id]
		if err = writer.Set(ctx, next); err != nil {
			return err
		}
		if key == o.Key {
			continue
		}
		if err = writer.Delete(ctx, o.Key); err != nil {
			return err
		}
		moved[id] = resource.ID{Type: resource.TypeStatus, Key: key}
	}
	if err = moveResources(ctx, tx, moved, dropped); err != nil {
		return err
	}
	return moveRelationships(ctx, relWriter, rels, moved, dropped)
}

func moveResources(
	ctx context.Context,
	tx gorp.Tx,
	moved map[resource.ID]resource.ID,
	dropped map[resource.ID]bool,
) error {
	var (
		reader = gorp.WrapReader[string, ontology.Resource](tx)
		writer = gorp.WrapWriter[string, ontology.Resource](tx)
	)
	for id := range dropped {
		if err := writer.Delete(ctx, id.String()); err != nil {
			return err
		}
	}
	for from, to := range moved {
		res, err := reader.Get(ctx, from.String())
		if errors.Is(err, query.ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		res.ID = to
		if err = writer.Set(ctx, res); err != nil {
			return err
		}
		if err = writer.Delete(ctx, from.String()); err != nil {
			return err
		}
	}
	return nil
}

func moveRelationships(
	ctx context.Context,
	writer *gorp.Writer[string, ontology.Relationship],
	rels []ontology.Relationship,
	moved map[resource.ID]resource.ID,
	dropped map[resource.ID]bool,
) error {
	for _, rel := range rels {
		if rel.Type == labeledBy && rel.From.Type == resource.TypeStatus {
			continue
		}
		oldKey := rel.GorpKey()
		if dropped[rel.From] || dropped[rel.To] {
			if err := writer.Delete(ctx, oldKey); err != nil {
				return err
			}
			continue
		}
		from, fromMoved := moved[rel.From]
		to, toMoved := moved[rel.To]
		if !fromMoved && !toMoved {
			continue
		}
		if fromMoved {
			rel.From = from
		}
		if toMoved {
			rel.To = to
		}
		if err := writer.Set(ctx, rel); err != nil {
			return err
		}
		if err := writer.Delete(ctx, oldKey); err != nil {
			return err
		}
	}
	return nil
}

// dropField removes the field name from details when details is a JSON object.
func dropField(details json.RawMessage, name string) (json.RawMessage, error) {
	if !bytes.HasPrefix(bytes.TrimSpace(details), []byte("{")) {
		return details, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(details, &fields); err != nil {
		return nil, err
	}
	if _, ok := fields[name]; !ok {
		return details, nil
	}
	delete(fields, name)
	return json.Marshal(fields)
}

func exists(
	ctx context.Context,
	r gorp.Reader[string, ontology.Resource],
	id resource.ID,
) (bool, error) {
	_, err := r.Get(ctx, id.String())
	if errors.Is(err, query.ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}

func collect[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	r gorp.Reader[K, E],
) (out []E, err error) {
	iter, err := r.OpenIterator(gorp.IterOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	for iter.First(); iter.Valid(); iter.Next() {
		if e := iter.Value(ctx); e != nil {
			out = append(out, *e)
		}
	}
	return out, iter.Error()
}
