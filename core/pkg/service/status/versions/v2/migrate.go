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
	"encoding/json"
	"slices"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	v1 "github.com/synnaxlabs/synnax/pkg/service/status/versions/v1"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	xuuid "github.com/synnaxlabs/x/uuid"
)

// namespace is the UUIDv5 namespace of the keys that MigrateKey derives. A change gives
// each migrated status a different key than other Cores and clients derive.
var namespace = uuid.MustParse("4a3c5e0e-6d0b-5f43-9a51-3c7e1f2b8d64")

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
	return xuuid.NewV5(namespace, key).String(), &id
}

// Migrations move each status to its MigrateKey key and owner, then move its labels
// from labeled_by ontology relationships into the labels field. They delete a status
// whose owner has no ontology resource, and drop labels with no ontology resource. They
// remove the owner's key from the details, such as details.task on a task status. The
// ontology resource of each moved status, and each relationship that touches it,
// follow the new key.
var Migrations = []gorp.Migration{
	migrate.New("v61_status_owners", migrateOwners),
	ontology.NewFieldMigration(ontology.FieldConfig[Key, stored]{
		Name: "v61_status_labels",
		Converts: func(rel ontology.Relationship) bool {
			return rel.Type == ontology.RelationshipTypeLabeledBy &&
				rel.From.Type == resource.TypeStatus
		},
		Holder: func(rel ontology.Relationship) string { return rel.From.Key },
		Set:    moveLabel,
	}),
}

type stored = Status[json.RawMessage]

func moveLabel(
	ctx context.Context,
	tx gorp.Tx,
	s *stored,
	rel ontology.Relationship,
) (bool, error) {
	key, err := uuid.Parse(rel.To.Key)
	if err != nil {
		return false, errors.Wrapf(err, "label of status %s", rel.From.Key)
	}
	found, err := ontology.ResourceExists(ctx, tx, rel.To)
	if !found || err != nil {
		return false, err
	}
	s.Labels = append(s.Labels, key)
	return true, nil
}

func migrateOwners(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
	var old []v1.Status[json.RawMessage]
	if err := gorp.NewRetrieve[Key, v1.Status[json.RawMessage]]().
		Entries(&old).
		Exec(ctx, tx); err != nil {
		return err
	}
	rels, err := ontology.RetrieveRelationships(
		ctx,
		tx,
		func(ontology.Relationship) bool { return true },
	)
	if err != nil {
		return err
	}
	relWriter := gorp.WrapWriter[string, ontology.Relationship](tx)
	var (
		writer  = gorp.WrapWriter[Key, stored](tx)
		moved   = make(map[resource.ID]resource.ID)
		dropped = make(set.Set[resource.ID])
	)
	for _, o := range old {
		id := resource.ID{Type: resource.TypeStatus, Key: o.Key}
		key, owner := MigrateKey(o.Key)
		if owner != nil {
			found, err := ontology.ResourceExists(ctx, tx, *owner)
			if err != nil {
				return err
			}
			if !found {
				dropped.Add(id)
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
		next.Key, next.Owner = key, owner
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
	dropped set.Set[resource.ID],
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
	dropped set.Set[resource.ID],
) error {
	for _, rel := range rels {
		oldKey := rel.GorpKey()
		if dropped.Contains(rel.From) || dropped.Contains(rel.To) {
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
