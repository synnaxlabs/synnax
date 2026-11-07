// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/set"
)

// Ancestors returns the parent relationships on every path from ids up to root, or up
// to RootID when root is zero. The walk does not go above root. Each relationship
// appears once. Ancestors reads the changes staged against tx, falling back to the
// underlying database when tx is nil.
func (o *Ontology) Ancestors(
	tx gorp.Tx,
	root resource.ID,
	ids ...resource.ID,
) ([]Relationship, error) {
	tx = o.cfg.DB.OverrideTx(tx)
	if root.IsZero() {
		root = RootID
	}
	visited := set.New(root)
	frontier := make([]resource.ID, 0, len(ids))
	for _, id := range ids {
		if !visited.Contains(id) {
			visited.Add(id)
			frontier = append(frontier, id)
		}
	}
	var rels []Relationship
	for len(frontier) > 0 {
		keys, err := o.relIndexes.byTo.Get(tx, frontier...)
		if err != nil {
			return nil, err
		}
		frontier = frontier[:0]
		for _, key := range keys {
			rel, err := ParseRelationship(key)
			if err != nil {
				return nil, err
			}
			if rel.Type != RelationshipTypeParentOf {
				continue
			}
			rels = append(rels, rel)
			if !visited.Contains(rel.From) {
				visited.Add(rel.From)
				frontier = append(frontier, rel.From)
			}
		}
	}
	return rels, nil
}

// descendants answers whether IDs descend from a root, and keeps each answer for the
// next ID.
type descendants struct {
	o       *Ontology
	tx      gorp.Tx
	root    resource.ID
	inside  set.Set[resource.ID]
	outside set.Set[resource.ID]
	// err is the first failure of a walk. Once it is set, except reports true for every
	// ID.
	err error
}

func (o *Ontology) newDescendants(tx gorp.Tx, root resource.ID) *descendants {
	return &descendants{
		o:       o,
		tx:      tx,
		root:    root,
		inside:  make(set.Set[resource.ID]),
		outside: make(set.Set[resource.ID]),
	}
}

// except reports whether id does not descend from the root.
func (d *descendants) except(id resource.ID) bool {
	if d.err != nil {
		return true
	}
	inside, err := d.descends(id)
	if err != nil {
		d.err = err
		return true
	}
	return !inside
}

func (d *descendants) descends(id resource.ID) (bool, error) {
	if d.inside.Contains(id) {
		return true, nil
	}
	if d.outside.Contains(id) {
		return false, nil
	}
	visited := set.New(id)
	for frontier := []resource.ID{id}; len(frontier) > 0; {
		keys, err := d.o.relIndexes.byTo.Get(d.tx, frontier...)
		if err != nil {
			return false, err
		}
		frontier = frontier[:0]
		for _, key := range keys {
			rel, err := ParseRelationship(key)
			if err != nil {
				return false, err
			}
			if rel.Type != RelationshipTypeParentOf || visited.Contains(rel.From) {
				continue
			}
			if rel.From == d.root || d.inside.Contains(rel.From) {
				d.inside.Add(id)
				return true, nil
			}
			visited.Add(rel.From)
			// A parent known to be outside has no path to the root.
			if !d.outside.Contains(rel.From) {
				frontier = append(frontier, rel.From)
			}
		}
	}
	// The walk found every ancestor of each visited ID, and none is the root.
	d.outside.Add(visited.Slice()...)
	return false, nil
}
