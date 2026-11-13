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
	"go.uber.org/zap"
)

// OwnersConfig configures a migration that moves the projects of the visualizations of
// type Type out of the ontology and into their project fields.
type OwnersConfig[K gorp.Key, E gorp.Entry[K]] struct {
	// Name is the migration's key.
	Name string
	// Type is the resource type of the visualizations.
	Type resource.Type
	// Owned reports whether e has a project.
	Owned func(e E) bool
	// NameOf returns the name of e.
	NameOf func(e E) string
	// SetOwner sets the project of e.
	SetOwner func(e *E, project uuid.UUID)
}

// NewOwnersMigration returns a migration that sets the project of each visualization of
// type cfg.Type from the parent edge that points at it from a stored project. An edge
// from a project that is not stored counts as no edge. The migration deletes the
// converted edges, then deletes each visualization with no project along with its
// ontology resource and edges, and logs a warning for each. It reads the project table,
// so the project table must open and migrate first.
func NewOwnersMigration[K gorp.Key, E gorp.Entry[K]](
	cfg OwnersConfig[K, E],
) gorp.Migration {
	fields := ontology.NewFieldMigration(ontology.FieldConfig[K, E]{
		Name: cfg.Name,
		Converts: func(rel ontology.Relationship) bool {
			return rel.Type == ontology.RelationshipTypeParentOf &&
				rel.To.Type == cfg.Type &&
				rel.From.Type == resource.TypeProject
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
			return deleteOrphans(ctx, tx, cfg, ins)
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
	owner, err := uuid.Parse(rel.From.Key)
	if err != nil {
		return false, errors.Wrapf(err, "owner of %s", rel.To)
	}
	found, err := gorp.NewRetrieve[Key, Project]().
		Where(gorp.MatchKeys[Key, Project](owner)).
		Exists(ctx, tx)
	if !found || err != nil {
		return false, err
	}
	cfg.SetOwner(e, owner)
	return true, nil
}

func deleteOrphans[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	tx gorp.Tx,
	cfg OwnersConfig[K, E],
	ins alamos.Instrumentation,
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
		ins.L.Warn(
			"deleted a visualization with no project",
			zap.String("type", string(cfg.Type)),
			zap.String("key", key),
			zap.String("name", cfg.NameOf(e)),
		)
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
