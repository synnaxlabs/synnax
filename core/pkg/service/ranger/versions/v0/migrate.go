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
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// MigrationConfig is the configuration for newMigration.
type MigrationConfig struct {
	// Group is the group service the legacy range groups are folded into.
	Group *group.Service
	alamos.Instrumentation
}

// newMigration returns the v0 migration, which swaps invalid time ranges and folds
// legacy range groups into the group service.
func newMigration(cfg MigrationConfig) gorp.Migration {
	return migrate.New(
		"range_groups_1",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			if migrated, err := alreadyMigrated(tx); err != nil || migrated {
				return err
			}
			ins.L.Debug("swapping invalid time ranges")
			ranges, err := swapInvalidTimeRanges(ctx, tx)
			if err != nil {
				return err
			}
			return foldGroups(ctx, tx, cfg, ranges)
		},
	)
}

// swapInvalidTimeRanges makes the time range of every stored range valid, and returns
// every range keyed by its key.
func swapInvalidTimeRanges(
	ctx context.Context,
	tx gorp.Tx,
) (ranges map[Key]Range, err error) {
	writer := gorp.WrapWriter[Key, Range](tx)
	iter, err := gorp.WrapReader[Key, Range](tx).OpenIterator(gorp.IterOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = errors.Combine(err, iter.Close()); err != nil {
			ranges = nil
		}
	}()
	ranges = make(map[Key]Range)
	for iter.First(); iter.Valid(); iter.Next() {
		rng := iter.Value()
		if err = iter.Error(); err != nil {
			return nil, errors.Wrap(err, "invalid range")
		}
		if !rng.TimeRange.Valid() {
			rng.TimeRange = rng.TimeRange.MakeValid()
			if err = writer.Set(ctx, *rng); err != nil {
				return nil, err
			}
		}
		ranges[rng.Key] = *rng
	}
	return ranges, nil
}

// foldGroups replaces each group under the "Ranges" group with a parent range of the
// group's child ranges, then deletes the groups. It does nothing when no "Ranges" group
// exists.
func foldGroups(
	ctx context.Context,
	tx gorp.Tx,
	cfg MigrationConfig,
	ranges map[Key]Range,
) error {
	var topLevelGroup group.Group
	if err := cfg.Group.
		NewRetrieve().
		Entry(&topLevelGroup).
		Where(group.MatchNames("Ranges")).
		Exec(ctx, tx); err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return nil
		}
		return err
	}
	edges, err := parentEdgesByFrom(ctx, tx)
	if err != nil {
		return err
	}
	topEdges := edges[topLevelGroup.ResourceID()]
	groups, err := retrieveChildGroups(ctx, tx, cfg.Group, topEdges)
	if err != nil {
		return err
	}
	cfg.L.Debug("converting groups to parent ranges", zap.Int("groups", len(groups)))
	if err = ontology.DeleteRelationships(ctx, tx, topEdges...); err != nil {
		return err
	}
	if err = cfg.Group.NewWriter(tx).Delete(ctx, topLevelGroup.Key); err != nil {
		return err
	}
	for _, g := range groups {
		if err = foldGroup(ctx, tx, cfg, g, edges[g.ResourceID()], ranges); err != nil {
			return err
		}
	}
	return nil
}

// retrieveChildGroups returns the groups that edges point to. An edge can name a group
// that no longer exists.
func retrieveChildGroups(
	ctx context.Context,
	tx gorp.Tx,
	svc *group.Service,
	edges []ontology.Relationship,
) ([]group.Group, error) {
	var keys []uuid.UUID
	for _, e := range edges {
		if e.To.Type != resource.TypeGroup {
			continue
		}
		k, err := uuid.Parse(e.To.Key)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	var groups []group.Group
	if err := svc.
		NewRetrieve().
		Where(group.MatchKeys(keys...)).
		Entries(&groups).
		Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return nil, err
	}
	return groups, nil
}

// foldGroup deletes g and its child edges, and creates a parent range of the ranges
// that childEdges point to.
func foldGroup(
	ctx context.Context,
	tx gorp.Tx,
	cfg MigrationConfig,
	g group.Group,
	childEdges []ontology.Relationship,
	ranges map[Key]Range,
) error {
	var childRanges []Range
	for _, e := range childEdges {
		if e.To.Type != resource.TypeRange {
			continue
		}
		rKey, err := uuid.Parse(e.To.Key)
		if err != nil {
			return err
		}
		if r, ok := ranges[rKey]; ok {
			childRanges = append(childRanges, r)
		}
	}
	cfg.L.Debug(
		"migrating range group",
		zap.String("group", g.Name),
		zap.Int("children", len(childRanges)),
	)
	if err := ontology.DeleteRelationships(ctx, tx, childEdges...); err != nil {
		return err
	}
	if err := cfg.Group.NewWriter(tx).Delete(ctx, g.Key); err != nil {
		return err
	}
	if len(childRanges) == 0 {
		return nil
	}
	return createParentRange(ctx, tx, g.Name, childRanges)
}

// createParentRange creates a range named name that spans children, and makes it the
// parent of each child.
func createParentRange(
	ctx context.Context,
	tx gorp.Tx,
	name string,
	children []Range,
) error {
	tr := telem.TimeRange{Start: telem.TimeStampMax, End: telem.TimeStampMin}
	for _, r := range children {
		if r.TimeRange.Start.Before(tr.Start) {
			tr.Start = r.TimeRange.Start
		}
		if r.TimeRange.End.After(tr.End) {
			tr.End = r.TimeRange.End
		}
	}
	parent := Range{Key: uuid.New(), Name: name, TimeRange: tr.MakeValid()}
	if err := gorp.WrapWriter[Key, Range](tx).Set(ctx, parent); err != nil {
		return err
	}
	parentID := parent.ResourceID()
	if err := ontology.DefineResources(ctx, tx, parentID); err != nil {
		return err
	}
	return gorp.WrapWriter[string, ontology.Relationship](tx).Set(
		ctx,
		lo.Map(children, func(r Range, _ int) ontology.Relationship {
			return ontology.Relationship{
				From: parentID,
				Type: ontology.RelationshipTypeParentOf,
				To:   r.ResourceID(),
			}
		})...,
	)
}

// parentEdgesByFrom returns every parent edge, keyed by the resource it comes from.
func parentEdgesByFrom(
	ctx context.Context,
	tx gorp.Tx,
) (map[resource.ID][]ontology.Relationship, error) {
	edges, err := ontology.RetrieveRelationships(
		ctx,
		tx,
		func(r ontology.Relationship) bool {
			return r.Type == ontology.RelationshipTypeParentOf
		},
	)
	if err != nil {
		return nil, err
	}
	return lo.GroupBy(edges, func(r ontology.Relationship) resource.ID {
		return r.From
	}), nil
}

// alreadyMigrated reports whether a Core up to v0.53 already folded the store's legacy
// range groups. A group named "Ranges" on such a store belongs to the user.
func alreadyMigrated(tx gorp.Tx) (bool, error) {
	_, closer, err := tx.Get([]byte("sy_ranger_migration_performed"))
	if err != nil {
		return false, errors.Skip(err, query.ErrNotFound)
	}
	if err = closer.Close(); err != nil {
		return false, err
	}
	return true, nil
}

// codecMigration re-encodes stored ranges from MessagePack to the Orc value-color
// layout.
var codecMigration = migrate.Codec[Key, Range]("msgpack_to_orc")

// NewMigrations returns the ordered set of migrations introduced at this version.
func NewMigrations(cfg MigrationConfig) []gorp.Migration {
	return []gorp.Migration{newMigration(cfg), codecMigration}
}

// NormalizeKeys re-keys Range rows stored under the pre-v0.54 key format.
var NormalizeKeys = migrate.NormalizeKeys[Key, Range]("Range")
