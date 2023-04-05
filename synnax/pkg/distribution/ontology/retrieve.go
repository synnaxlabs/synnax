// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology

import (
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

// Retrieve implements a set of methods for retrieving resources and traversing their
// relationships in teh ontology.
type Retrieve struct {
	query     *gorp.CompoundRetrieve[ID, Resource]
	registrar serviceRegistrar
	txn       gorp.Reader
}

func newRetrieve(registrar serviceRegistrar, txn gorp.Reader) Retrieve {
	r := Retrieve{
		query:     &gorp.CompoundRetrieve[ID, Resource]{},
		registrar: registrar,
		txn:       txn,
	}
	r.query.Next()
	return r
}

// WhereIDs filters resources by the provided keys.
func (r Retrieve) WhereIDs(keys ...ID) Retrieve {
	r.query.Current().WhereKeys(keys...)
	return r
}

// Where filters resources by the provided predicate.
func (r Retrieve) Where(filter func(r *Resource) bool) Retrieve {
	r.query.Current().Where(filter)
	return r
}

// Direction is the direction of a relationship traversal.
type Direction uint8

const (
	// Forward represents a forward traversal i.e. From -> To.
	Forward Direction = iota + 1
	// Backward represents a backward traversal i.e. To -> From.
	Backward Direction = 2
)

// GetID returns the directional ID of the relationship.
func (d Direction) GetID(rel *Relationship) ID {
	if d == Forward {
		return rel.To
	}
	if d == Backward {
		return rel.From
	}
	panic("invalid direction")
}

// Traverser is a struct that defines the traversal of a relationship between entities
// in the ontology.
type Traverser struct {
	// Filter if a function that returns true if the given Resource and Relationship
	// should be included in the traversal results.
	Filter func(res *Resource, rel *Relationship) bool
	// Direction is the direction of the traversal. See (Direction) for more.
	Direction Direction
}

var (
	// Parents traverses to the parents of a resource.
	Parents = Traverser{
		Filter: func(res *Resource, rel *Relationship) bool {
			return rel.Type == ParentOf && rel.To == res.ID
		},
		Direction: Backward,
	}
	// Children traverses to the children of a resource.
	Children = Traverser{
		Filter: func(res *Resource, rel *Relationship) bool {
			return rel.Type == ParentOf && rel.From == res.ID
		},
		Direction: Forward,
	}
)

// TraverseTo traverses to the provided relationship type. All filtering methods will
// now be applied to elements of the traversed relationship.
func (r Retrieve) TraverseTo(t Traverser) Retrieve {
	setTraverser(r.query.Current(), t)
	r.query.Next()
	return r
}

// Entry binds the entry that the Query will fill results into. Calls to Entry will
// override all previous calls to Entries or Entry.
func (r Retrieve) Entry(res *Resource) Retrieve {
	r.query.Current().Entry(res)
	return r
}

// Entries binds a slice that the Query will fill results into. Calls to Entry will
// override all previous calls to Entries or Entry.
func (r Retrieve) Entries(res *[]Resource) Retrieve {
	r.query.Current().Entries(res)
	return r
}

// Exec executes the query.
func (r Retrieve) Exec() error {
	var nextIDs []ID
	for i, clause := range r.query.Clauses {
		if i != 0 {
			clause.WhereKeys(nextIDs...)
		}
		if err := clause.Exec(r.txn); err != nil {
			return err
		}
		atLast := len(r.query.Clauses) == i+1
		resources, err := r.retrieveEntities(clause)
		if err != nil || len(resources) == 0 || atLast {
			return err
		}
		if nextIDs, err = r.traverse(getTraverser(clause), resources); err != nil {
			return err
		}
	}
	return nil
}

const traverseOptKey = "traverse"

func setTraverser(q query.Query, f Traverser) {
	q.Set(traverseOptKey, f)
}

func getTraverser(q query.Query) Traverser {
	return q.GetRequired(traverseOptKey).(Traverser)
}

func (r Retrieve) retrieveEntities(
	clause gorp.Retrieve[ID, Resource],
) ([]Resource, error) {
	entries := gorp.GetEntries[ID, Resource](clause)
	for j, res := range entries.All() {
		data, err := r.registrar.retrieveEntity(r.txn, res.ID)
		if err != nil {
			return nil, err
		}
		res.Entity = data
		entries.Set(j, res)
	}
	return entries.All(), nil
}

func (r Retrieve) traverse(
	traverse Traverser,
	resources []Resource,
) ([]ID, error) {
	var nextIDs []ID
	return nextIDs, gorp.NewRetrieve[string, Relationship]().
		Where(func(rel *Relationship) bool {
			for _, resource := range resources {
				if traverse.Filter(&resource, rel) {
					nextIDs = append(nextIDs, traverse.Direction.GetID(rel))
				}
			}
			return false
		}).Exec(r.txn)
}
