// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

type groupsMigration struct {
	cfg ServiceConfig
}

var _ gorp.Migration = (*groupsMigration)(nil)

func (m *groupsMigration) Name() string { return "range_groups" }

func (m *groupsMigration) Run(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
	ins.L.Debug("swapping invalid time ranges")
	if err := m.swapRanges(ctx, tx); err != nil {
		return err
	}

	var topLevelGroup group.Group
	if err := m.cfg.Group.
		NewRetrieve().
		Entry(&topLevelGroup).
		WhereNames("Ranges").
		Exec(ctx, tx); err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return nil
		}
		return err
	}

	var groups []ontology.Resource
	if err := m.cfg.Ontology.
		NewRetrieve().
		WhereIDs(topLevelGroup.OntologyID()).
		TraverseTo(ontology.ChildrenTraverser).
		WhereTypes(ontology.ResourceTypeGroup).
		Entries(&groups).
		Exec(ctx, tx); err != nil {
		return err
	}
	m.cfg.L.Debug("converting groups to parent ranges", zap.Int("groups", len(groups)))

	otgWriter := m.cfg.Ontology.NewWriter(tx)
	if err := otgWriter.DeleteOutgoingRelationshipsOfType(
		ctx,
		topLevelGroup.OntologyID(),
		ontology.RelationshipTypeParentOf,
	); err != nil {
		return err
	}
	if err := m.cfg.Group.NewWriter(tx).Delete(ctx, topLevelGroup.Key); err != nil {
		return err
	}

	rangeMap, err := m.loadAllRanges(ctx, tx)
	if err != nil {
		return err
	}

	writer := gorp.WrapWriter[Key, Range](tx)

	for _, g := range groups {
		var childRanges []ontology.Resource
		// ExcludeFieldData is required because this migration runs during
		// OpenService before the range ontology service is registered. Without
		// it, the ontology would panic trying to retrieve range resources.
		if err = m.cfg.Ontology.
			NewRetrieve().
			WhereIDs(g.ID).
			TraverseTo(ontology.ChildrenTraverser).
			WhereTypes(ontology.ResourceTypeRange).
			ExcludeFieldData(true).
			Entries(&childRanges).
			Exec(ctx, tx); err != nil {
			return err
		}
		m.cfg.L.Debug(
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
		if err = m.cfg.Group.NewWriter(tx).Delete(ctx, gKey); err != nil {
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
			newParentRange.OntologyID(),
			ontology.RelationshipTypeParentOf,
			lo.Map(childRanges, func(r ontology.Resource, _ int) ontology.ID {
				return r.ID
			}),
		); err != nil {
			return err
		}
	}

	return nil
}

func (m *groupsMigration) swapRanges(
	ctx context.Context,
	tx gorp.Tx,
) (err error) {
	writer := gorp.WrapWriter[Key, Range](tx)
	iter, err := gorp.WrapReader[Key, Range](tx).OpenIterator(gorp.IterOptions{})
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	for iter.First(); iter.Valid(); iter.Next() {
		rng := iter.Value(ctx)
		if err := iter.Error(); err != nil {
			return errors.Wrap(err, "invalid range")
		}
		if rng.TimeRange.Valid() {
			continue
		}
		rng.TimeRange = rng.TimeRange.MakeValid()
		if err = writer.Set(ctx, *rng); err != nil {
			return err
		}
	}
	return err
}

func (m *groupsMigration) loadAllRanges(
	ctx context.Context,
	tx gorp.Tx,
) (map[uuid.UUID]Range, error) {
	result := make(map[uuid.UUID]Range)
	reader := gorp.WrapReader[Key, Range](tx)
	iter, err := reader.OpenIterator(gorp.IterOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	for iter.First(); iter.Valid(); iter.Next() {
		v := iter.Value(ctx)
		if err := iter.Error(); err != nil {
			return nil, err
		}
		result[v.Key] = *v
	}
	return result, err
}
