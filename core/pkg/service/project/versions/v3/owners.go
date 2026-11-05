// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// OwnersConfig configures a migration that moves the owners of the visualizations of
// type Type out of the ontology and into their project and range fields.
type OwnersConfig[K gorp.Key, E gorp.Entry[K]] struct {
	// Name is the migration's key.
	Name string
	// Type is the resource type of the visualizations.
	Type resource.Type
	// SetOwner sets the project or the range of e. Exactly one of them is non-nil.
	SetOwner func(e *E, project, rng *uuid.UUID)
}

// NewOwnersMigration returns a migration that sets the owner of each visualization of
// type cfg.Type from the parent edge that points at it from a project or a range. A
// project edge wins over a range edge. The migration deletes the converted edges, and
// deletes each visualization with no owner along with its ontology resource and edges.
func NewOwnersMigration[K gorp.Key, E gorp.Entry[K]](
	cfg OwnersConfig[K, E],
) gorp.Migration {
	return migrate.New(
		cfg.Name,
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			return migrateOwners(ctx, tx, cfg)
		},
	)
}

func migrateOwners[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	tx gorp.Tx,
	cfg OwnersConfig[K, E],
) error {
	edges, err := collect[string, ontology.Relationship](
		ctx,
		tx,
		func(r ontology.Relationship) bool {
			return r.To.Type == cfg.Type || r.From.Type == cfg.Type
		},
	)
	if err != nil {
		return err
	}
	var (
		projects  = make(map[string]uuid.UUID)
		ranges    = make(map[string]uuid.UUID)
		converted []string
	)
	for _, e := range edges {
		if e.Type != ontology.RelationshipTypeParentOf || e.To.Type != cfg.Type {
			continue
		}
		owners := projects
		switch e.From.Type {
		case resource.TypeProject:
		case resource.TypeRange:
			owners = ranges
		default:
			continue
		}
		k, err := uuid.Parse(e.From.Key)
		if err != nil {
			return errors.Wrapf(err, "parent of %s", e.To)
		}
		owners[e.To.Key] = k
		converted = append(converted, e.GorpKey())
	}
	entries, err := collect[K, E](ctx, tx, func(E) bool { return true })
	if err != nil {
		return err
	}
	var (
		owned     = make([]E, 0, len(entries))
		orphans   []K
		resources []string
		orphaned  = set.New[string]()
	)
	for _, e := range entries {
		key := gorp.FormatKey(e.GorpKey())
		if p, ok := projects[key]; ok {
			cfg.SetOwner(&e, &p, nil)
		} else if r, ok := ranges[key]; ok {
			cfg.SetOwner(&e, nil, &r)
		} else {
			orphans = append(orphans, e.GorpKey())
			resources = append(
				resources,
				resource.ID{Type: cfg.Type, Key: key}.String(),
			)
			orphaned.Add(key)
			continue
		}
		owned = append(owned, e)
	}
	for _, e := range edges {
		if e.To.Type == cfg.Type && orphaned.Contains(e.To.Key) ||
			e.From.Type == cfg.Type && orphaned.Contains(e.From.Key) {
			converted = append(converted, e.GorpKey())
		}
	}
	if err = gorp.WrapWriter[K, E](tx).Set(ctx, owned...); err != nil {
		return err
	}
	if err = gorp.WrapWriter[K, E](tx).Delete(ctx, orphans...); err != nil {
		return err
	}
	if err = gorp.WrapWriter[string, ontology.Resource](tx).
		Delete(ctx, resources...); err != nil {
		return err
	}
	return gorp.WrapWriter[string, ontology.Relationship](tx).Delete(ctx, converted...)
}

func collect[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	tx gorp.Tx,
	keep func(E) bool,
) (out []E, err error) {
	iter, err := gorp.WrapReader[K, E](tx).OpenIterator(gorp.IterOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	for iter.First(); iter.Valid(); iter.Next() {
		if e := iter.Value(ctx); e != nil && keep(*e) {
			out = append(out, *e)
		}
	}
	return out, iter.Error()
}
