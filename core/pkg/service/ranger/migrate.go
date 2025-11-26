// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

func (s *Service) migrate(ctx context.Context) error {
	return gorp.Migrator{
		Key: "sy_ranger_migration_performed",
		Migrations: []gorp.MigrationSpec{
			{Name: "range_groups", Migrate: s.migrateRangeGroups},
		},
		Force: *s.ForceMigration,
	}.Run(ctx, s.DB)
}

func (s *Service) migrateRangeGroups(ctx context.Context, tx gorp.Tx) error {
	s.L.Info("Swapping ranges to make sure the time range is valid.")
	if err := s.NewWriter(tx).swapRanges(ctx); err != nil {
		return err
	}
	var topLevelGroup group.Group
	if err := s.
		Group.
		NewRetrieve().
		Entry(&topLevelGroup).
		WhereNames("Ranges").
		Exec(ctx, tx); err != nil {
		if errors.Is(err, query.NotFound) {
			return nil
		}
		return err
	}
	s.L.Info(
		"Migrating subgroups of the Ranges group to ranges and deleting the group.",
	)
	var groups []ontology.Resource
	if err := s.
		Ontology.
		NewRetrieve().
		WhereIDs(topLevelGroup.OntologyID()).
		TraverseTo(ontology.Children).
		WhereTypes(group.OntologyType).
		Entries(&groups).
		Exec(ctx, tx); err != nil {
		return err
	}
	if err := s.Ontology.NewWriter(tx).DeleteOutgoingRelationshipsOfType(
		ctx,
		topLevelGroup.OntologyID(),
		ontology.ParentOf,
	); err != nil {
		return err
	}
	if err := s.Group.NewWriter(tx).Delete(ctx, topLevelGroup.Key); err != nil {
		return err
	}
	// Now we have subgroups. For each subgroup, we want to grab the children of that
	// group that are ranges, and create a corresponding parent range for that group
	// whose time range is the union of the children's time ranges, whose name is the
	// subgroup's name, and has children that are the children of the subgroup.
	for _, g := range groups {
		var childRanges []ontology.Resource
		if err := s.
			Ontology.
			NewRetrieve().
			WhereIDs(g.ID).
			TraverseTo(ontology.Children).
			WhereTypes(OntologyType).
			Entries(&childRanges).
			Exec(ctx, tx); err != nil {
			return err
		}
		s.L.Infof(
			"Migrating range group: %s with %d children.",
			g.Name,
			len(childRanges),
		)
		gKey, err := uuid.Parse(g.ID.Key)
		if err != nil {
			return err
		}
		if err := s.Ontology.NewWriter(tx).DeleteOutgoingRelationshipsOfType(
			ctx,
			g.ID,
			ontology.ParentOf,
		); err != nil {
			return err
		}
		if err := s.Group.NewWriter(tx).Delete(ctx, gKey); err != nil {
			return err
		}
		if len(childRanges) == 0 {
			continue
		}
		tr := telem.TimeRange{Start: telem.TimeStampMax, End: telem.TimeStampMin}
		for _, r := range childRanges {
			rKey, err := uuid.Parse(r.ID.Key)
			if err != nil {
				return err
			}
			var r Range
			if err := s.
				NewRetrieve().
				WhereKeys(rKey).
				Entry(&r).
				Exec(ctx, tx); err != nil {
				return err
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
		if err := s.NewWriter(tx).Create(ctx, &newParentRange); err != nil {
			return err
		}
		if err := s.
			Ontology.
			NewWriter(tx).
			DefineFromOneToManyRelationships(
				ctx,
				newParentRange.OntologyID(),
				ontology.ParentOf,
				lo.Map(childRanges, func(r ontology.Resource, _ int) ontology.ID {
					return r.ID
				}),
			); err != nil {
			return err
		}
	}
	s.L.Info("finished ranger migration")
	return nil
}
