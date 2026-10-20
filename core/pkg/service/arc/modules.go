// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
)

// retrieveModulePaths returns the import path of every module below the Arcs group,
// sorted. A path joins the group names from the Arcs group down to the module with ".".
func (s *Service) retrieveModulePaths(
	ctx context.Context,
	tx gorp.Tx,
) ([]string, error) {
	type frame struct {
		id     ontology.ID
		prefix string
	}
	var (
		frontier = []frame{{id: s.group.OntologyID()}}
		visited  = set.New(s.group.OntologyID())
		ids      []ontology.ID
		prefixes = map[ontology.ID]string{}
	)
	for len(frontier) > 0 {
		f := frontier[0]
		frontier = frontier[1:]
		groups, err := s.retrieveChildren(ctx, tx, f.id, ontology.ResourceTypeGroup, false)
		if err != nil {
			return nil, err
		}
		for _, g := range groups {
			if visited.Contains(g.ID) {
				continue
			}
			visited.Add(g.ID)
			frontier = append(frontier, frame{id: g.ID, prefix: f.prefix + g.Name + "."})
		}
		arcs, err := s.retrieveChildren(ctx, tx, f.id, ontology.ResourceTypeArc, true)
		if err != nil {
			return nil, err
		}
		for _, a := range arcs {
			if _, seen := prefixes[a.ID]; seen {
				continue
			}
			prefixes[a.ID] = f.prefix
			ids = append(ids, a.ID)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	keys, err := KeysFromOntologyIDs(ids)
	if err != nil {
		return nil, err
	}
	var entries []Arc
	if err = s.NewRetrieve().
		Where(MatchKeys(keys...)).
		Entries(&entries).
		Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return nil, err
	}
	paths := make([]string, 0, len(entries))
	for _, a := range entries {
		if a.Type != TypeModule {
			continue
		}
		paths = append(paths, prefixes[a.OntologyID()]+a.Name)
	}
	slices.Sort(paths)
	return paths, nil
}

// retrieveChildren returns the children of parent with the given type, sorted. Field
// data is skipped when excludeData is true, which leaves Name empty.
func (s *Service) retrieveChildren(
	ctx context.Context,
	tx gorp.Tx,
	parent ontology.ID,
	typ ontology.ResourceType,
	excludeData bool,
) ([]ontology.Resource, error) {
	var children []ontology.Resource
	if err := s.cfg.Ontology.NewRetrieve().
		WhereIDs(parent).
		TraverseTo(ontology.ChildrenTraverser).
		WhereTypes(typ).
		ExcludeFieldData(excludeData).
		Entries(&children).
		Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return nil, err
	}
	imex.SortResources(children)
	return children, nil
}
