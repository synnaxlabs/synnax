// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
)

// RelationshipTypeLabeledBy indicates that a resource holds a label. From is labeled by
// To.
const RelationshipTypeLabeledBy RelationshipType = "labeled_by"

// RootID is the ID of the root of the ontology.
var RootID = resource.ID{Type: resource.TypeBuiltin, Key: "root"}

// RetrieveRelationships returns the stored relationships for which keep returns true.
func RetrieveRelationships(
	ctx context.Context,
	tx gorp.Tx,
	keep func(Relationship) bool,
) (out []Relationship, err error) {
	iter, err := gorp.WrapReader[string, Relationship](tx).
		OpenIterator(gorp.IterOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	for iter.First(); iter.Valid(); iter.Next() {
		if r := iter.Value(ctx); r != nil && keep(*r) {
			out = append(out, *r)
		}
	}
	return out, iter.Error()
}

// DeleteRelationships deletes rels. A relationship that is not stored is skipped.
func DeleteRelationships(ctx context.Context, tx gorp.Tx, rels ...Relationship) error {
	keys := make([]string, len(rels))
	for i, r := range rels {
		keys[i] = r.GorpKey()
	}
	return gorp.WrapWriter[string, Relationship](tx).Delete(ctx, keys...)
}

// ResourceExists reports whether a resource with id is stored.
func ResourceExists(ctx context.Context, tx gorp.Tx, id resource.ID) (bool, error) {
	r := gorp.WrapReader[string, Resource](tx)
	_, err := r.Get(ctx, id.String())
	if errors.Is(err, query.ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}

// FieldConfig configures a migration that moves relationships into a field of the
// entries of one table.
type FieldConfig[K gorp.Key, E gorp.Entry[K]] struct {
	// Name is the key of the migration.
	Name string
	// Converts reports whether the migration moves rel into a field.
	Converts func(rel Relationship) bool
	// Holder returns the key of the entry that holds rel, in gorp.FormatKey form.
	Holder func(rel Relationship) string
	// Set moves rel into a field of e and reports whether e changed. tx holds the
	// migrated entries, so Set can check that the target of rel exists.
	Set func(ctx context.Context, tx gorp.Tx, e *E, rel Relationship) (bool, error)
}

// NewFieldMigration returns a migration that moves each relationship that cfg.Converts
// into a field of the entry that holds it, in relationship key order. It then deletes
// every relationship it converts, including those whose holder has no entry.
func NewFieldMigration[K gorp.Key, E gorp.Entry[K]](
	cfg FieldConfig[K, E],
) gorp.Migration {
	return migrate.New(
		cfg.Name,
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			return migrateFields(ctx, tx, cfg)
		},
	)
}

func migrateFields[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	tx gorp.Tx,
	cfg FieldConfig[K, E],
) error {
	rels, err := RetrieveRelationships(ctx, tx, cfg.Converts)
	if err != nil || len(rels) == 0 {
		return err
	}
	var entries []E
	if err = gorp.NewRetrieve[K, E]().Entries(&entries).Exec(ctx, tx); err != nil {
		return err
	}
	byKey := make(map[string]int, len(entries))
	for i, e := range entries {
		byKey[gorp.FormatKey(e.GorpKey())] = i
	}
	changed := make(set.Set[int])
	for _, rel := range rels {
		i, ok := byKey[cfg.Holder(rel)]
		if !ok {
			continue
		}
		moved, err := cfg.Set(ctx, tx, &entries[i], rel)
		if err != nil {
			return err
		}
		if moved {
			changed.Add(i)
		}
	}
	updated := make([]E, 0, len(changed))
	for i := range changed {
		updated = append(updated, entries[i])
	}
	if err = gorp.WrapWriter[K, E](tx).Set(ctx, updated...); err != nil {
		return err
	}
	return DeleteRelationships(ctx, tx, rels...)
}

// DefineResources stores a resource for each of ids.
func DefineResources(ctx context.Context, tx gorp.Tx, ids ...resource.ID) error {
	resources := make([]Resource, len(ids))
	for i, id := range ids {
		resources[i] = Resource{ID: id}
	}
	return gorp.WrapWriter[string, Resource](tx).Set(ctx, resources...)
}

// DeleteResources deletes the resources with ids and every relationship from or to
// them.
func DeleteResources(ctx context.Context, tx gorp.Tx, ids ...resource.ID) error {
	deleted := set.New(ids...)
	rels, err := RetrieveRelationships(ctx, tx, func(r Relationship) bool {
		return deleted.Contains(r.From) || deleted.Contains(r.To)
	})
	if err != nil {
		return err
	}
	if err = DeleteRelationships(ctx, tx, rels...); err != nil {
		return err
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = id.String()
	}
	return gorp.WrapWriter[string, Resource](tx).Delete(ctx, keys...)
}

// Drop deletes every stored resource and relationship.
func Drop(ctx context.Context, tx gorp.Tx) error {
	if err := gorp.NewDelete[string, Resource]().Exec(ctx, tx); err != nil {
		return err
	}
	return gorp.NewDelete[string, Relationship]().Exec(ctx, tx)
}
