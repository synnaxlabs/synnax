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

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

type MigrationConfig struct {
	Ontology *ontology.Ontology
	Group    *group.Service
	alamos.Instrumentation
}

func Migration(cfg MigrationConfig) migrate.Migration {
	return gorp.NewMigration(
		"range_groups_1",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) (err error) {
			ins.L.Debug("swapping invalid time ranges")
			var (
				writer = gorp.WrapWriter[Key, Range](tx)
				reader = gorp.WrapReader[Key, Range](tx)
			)
			iter, err := reader.OpenIterator(gorp.IterOptions{})
			if err != nil {
				return err
			}
			defer func() {
				err = errors.Combine(err, iter.Close())
			}()
			rangeMap := make(map[uuid.UUID]Range)
			for iter.First(); iter.Valid(); iter.Next() {
				rng := iter.Value(ctx)
				if err = iter.Error(); err != nil {
					return errors.Wrap(err, "invalid range")
				}
				if !rng.TimeRange.Valid() {
					rng.TimeRange = rng.TimeRange.MakeValid()
					if err = writer.Set(ctx, *rng); err != nil {
						return err
					}
				}
				rangeMap[rng.Key] = *rng
			}

			var topLevelGroup group.Group
			if err = cfg.Group.
				NewRetrieve().
				Entry(&topLevelGroup).
				Where(group.MatchNames("Ranges")).
				Exec(ctx, tx); err != nil {
				if errors.Is(err, query.ErrNotFound) {
					return nil
				}
				return err
			}

			var groups []ontology.Resource
			if err = cfg.Ontology.
				NewRetrieve().
				WhereIDs(topLevelGroup.OntologyID()).
				TraverseTo(ontology.ChildrenTraverser).
				WhereTypes(ontology.ResourceTypeGroup).
				Entries(&groups).
				Exec(ctx, tx); err != nil {
				return err
			}
			cfg.L.Debug("converting groups to parent ranges", zap.Int("groups", len(groups)))

			otgWriter := cfg.Ontology.NewWriter(tx)
			if err := otgWriter.DeleteOutgoingRelationshipsOfType(
				ctx,
				topLevelGroup.OntologyID(),
				ontology.RelationshipTypeParentOf,
			); err != nil {
				return err
			}
			if err := cfg.Group.NewWriter(tx).Delete(ctx, topLevelGroup.Key); err != nil {
				return err
			}

			for _, g := range groups {
				var childRanges []ontology.Resource
				// ExcludeFieldData is required because this migration runs during
				// OpenService before the range ontology service is registered. Without
				// it, the ontology would panic trying to retrieve range resources.
				if err = cfg.Ontology.
					NewRetrieve().
					WhereIDs(g.ID).
					TraverseTo(ontology.ChildrenTraverser).
					WhereTypes(ontology.ResourceTypeRange).
					ExcludeFieldData(true).
					Entries(&childRanges).
					Exec(ctx, tx); err != nil {
					return err
				}
				cfg.L.Debug(
					"migrating range group",
					zap.String("group", g.Name),
					zap.Int("children", len(childRanges)),
				)
				gKey, err := uuid.Parse(g.ID.Key)
				if err != nil {
					return err
				}
				if err = otgWriter.DeleteOutgoingRelationshipsOfType(
					ctx,
					g.ID,
					ontology.RelationshipTypeParentOf,
				); err != nil {
					return err
				}
				if err = cfg.Group.NewWriter(tx).Delete(ctx, gKey); err != nil {
					return err
				}
				if len(childRanges) == 0 {
					continue
				}
				tr := telem.TimeRange{Start: telem.TimeStampMax, End: telem.TimeStampMin}
				for _, cr := range childRanges {
					rKey, err := uuid.Parse(cr.ID.Key)
					if err != nil {
						return err
					}
					r, ok := rangeMap[rKey]
					if !ok {
						continue
					}
					if r.TimeRange.Start.Before(tr.Start) {
						tr.Start = r.TimeRange.Start
					}
					if r.TimeRange.End.After(tr.End) {
						tr.End = r.TimeRange.End
					}
				}
				newParentRange := Range{
					Key:       uuid.New(),
					Name:      g.Name,
					TimeRange: tr.MakeValid(),
				}
				if err = writer.Set(ctx, newParentRange); err != nil {
					return err
				}
				if err = otgWriter.DefineResource(ctx, OntologyID(newParentRange.Key)); err != nil {
					return err
				}
				if err = otgWriter.DefineFromOneToManyRelationships(
					ctx,
					OntologyID(newParentRange.Key),
					ontology.RelationshipTypeParentOf,
					lo.Map(childRanges, func(r ontology.Resource, _ int) ontology.ID {
						return r.ID
					}),
				); err != nil {
					return err
				}
			}
			return nil
		},
	)
}
