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
	// Owned reports whether e has a project or a range.
	Owned func(e E) bool
	// SetOwner sets the project or the range of e. Exactly one of them is non-nil.
	SetOwner func(e *E, project, rng *uuid.UUID)
}

// NewOwnersMigration returns a migration that sets the owner of each visualization of
// type cfg.Type from the parent edge that points at it from a stored project or range.
// A project edge wins over a range edge, and an edge from an owner that is not stored
// counts as no edge. The migration deletes the converted edges, then deletes each
// visualization with no owner along with its ontology resource and edges.
func NewOwnersMigration[K gorp.Key, E gorp.Entry[K]](
	cfg OwnersConfig[K, E],
) gorp.Migration {
	fields := ontology.NewFieldMigration(ontology.FieldConfig[K, E]{
		Name: cfg.Name,
		Converts: func(rel ontology.Relationship) bool {
			return rel.Type == ontology.RelationshipTypeParentOf &&
				rel.To.Type == cfg.Type &&
				(rel.From.Type == resource.TypeProject || rel.From.Type == resource.TypeRange)
		},
		Holder: func(rel ontology.Relationship) string { return rel.To.Key },
		Set: func(
			ctx context.Context,
			tx gorp.Tx,
			e *E,
			rel ontology.Relationship,
		) (bool, error) {
			return setOwner(ctx, tx, cfg, e, rel)
		},
	})
	return migrate.New(
		cfg.Name,
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			if err := fields.Run(ctx, tx, ins); err != nil {
				return err
			}
			return deleteOrphans(ctx, tx, cfg)
		},
	)
}

func setOwner[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	tx gorp.Tx,
	cfg OwnersConfig[K, E],
	e *E,
	rel ontology.Relationship,
) (bool, error) {
	if rel.From.Type == resource.TypeRange && cfg.Owned(*e) {
		return false, nil
	}
	found, err := ontology.ResourceExists(ctx, tx, rel.From)
	if !found || err != nil {
		return false, err
	}
	owner, err := uuid.Parse(rel.From.Key)
	if err != nil {
		return false, errors.Wrapf(err, "owner of %s", rel.To)
	}
	if rel.From.Type == resource.TypeProject {
		cfg.SetOwner(e, &owner, nil)
	} else {
		cfg.SetOwner(e, nil, &owner)
	}
	return true, nil
}

func deleteOrphans[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	tx gorp.Tx,
	cfg OwnersConfig[K, E],
) error {
	var entries []E
	if err := gorp.NewRetrieve[K, E]().Entries(&entries).Exec(ctx, tx); err != nil {
		return err
	}
	var (
		orphans   []K
		resources []string
		orphaned  = set.New[string]()
	)
	for _, e := range entries {
		if cfg.Owned(e) {
			continue
		}
		key := gorp.FormatKey(e.GorpKey())
		orphans = append(orphans, e.GorpKey())
		resources = append(resources, resource.ID{Type: cfg.Type, Key: key}.String())
		orphaned.Add(key)
	}
	if len(orphans) == 0 {
		return nil
	}
	rels, err := ontology.RetrieveRelationships(
		ctx,
		tx,
		func(rel ontology.Relationship) bool {
			return rel.To.Type == cfg.Type && orphaned.Contains(rel.To.Key) ||
				rel.From.Type == cfg.Type && orphaned.Contains(rel.From.Key)
		},
	)
	if err != nil {
		return err
	}
	if err = gorp.WrapWriter[K, E](tx).Delete(ctx, orphans...); err != nil {
		return err
	}
	if err = gorp.WrapWriter[string, ontology.Resource](tx).
		Delete(ctx, resources...); err != nil {
		return err
	}
	return ontology.DeleteRelationships(ctx, tx, rels...)
}
