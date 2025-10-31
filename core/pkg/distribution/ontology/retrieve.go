// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

// Retrieve implements a set of methods for retrieving resources and traversing their
// relationships in teh ontology.
type Retrieve struct {
	query     *gorp.CompoundRetrieve[ID, Resource]
	registrar serviceRegistrar
	tx        gorp.Tx
}

// NewRetrieve opens a new Retrieve query, which can be used to traverse and read resources
// from the underlying ontology.
func (o *Ontology) NewRetrieve() Retrieve { return newRetrieve(o.registrar, o.DB) }

func newRetrieve(registrar serviceRegistrar, tx gorp.Tx) Retrieve {
	r := Retrieve{
		query:     &gorp.CompoundRetrieve[ID, Resource]{},
		registrar: registrar,
		tx:        tx,
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
func (r Retrieve) Where(filter gorp.FilterFunc[ID, Resource]) Retrieve {
	r.query.Current().Where(filter)
	return r
}

func (r Retrieve) WhereTypes(types ...Type) Retrieve {
	r.query.Current().Where(func(ctx gorp.Context, r *Resource) (bool, error) {
		return lo.Contains(types, r.ID.Type), nil
	})
	return r
}

// Limit limits the number of results returned.
func (r Retrieve) Limit(limit int) Retrieve {
	r.query.Current().Limit(limit)
	return r
}

// Offset offsets the results returned.
func (r Retrieve) Offset(offset int) Retrieve {
	r.query.Current().Offset(offset)
	return r
}

// ExcludeFieldData includes the field data of the resource in the results based on the
// provided predicate.
func (r Retrieve) ExcludeFieldData(excludeFieldData bool) Retrieve {
	setExcludeFieldData(r.query.Current().Params, excludeFieldData)
	return r
}

// Direction is the direction of a relationship traversal.
type Direction uint8

const (
	// Forward represents a forward traversal i.e. Start -> To.
	Forward Direction = iota + 1
	// Backward represents a backward traversal i.e. To -> Start.
	Backward Direction = 2
)

// GetID returns the directional ID of the relationship.
func (d Direction) GetID(rel *Relationship) ID {
	return lo.Ternary(d == Forward, rel.To, rel.From)
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
	// Children traverse to the children of a resource.
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
	setTraverser(r.query.Current().Params, t)
	r.query.Next()
	return r
}

// Entry binds the entry that the query will fill results into. Calls to Entry will
// override all previous calls to Entries or Entry.
func (r Retrieve) Entry(res *Resource) Retrieve {
	r.query.Current().Entry(res)
	return r
}

// Entries binds a slice that the query will fill results into. Calls to Entry will
// override all previous calls to Entries or Entry.
func (r Retrieve) Entries(res *[]Resource) Retrieve {
	r.query.Current().Entries(res)
	return r
}

// Exec executes the query.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	var nextIDs []ID
	tx = gorp.OverrideTx(r.tx, tx)
	for i, clause := range r.query.Clauses {
		if i != 0 {
			clause.WhereKeys(nextIDs...)
		}
		cErr := clause.Exec(ctx, tx)
		atLast := len(r.query.Clauses) == i+1
		resources, err := r.retrieveEntities(ctx, clause, tx)
		if cErr != nil || err != nil || len(resources) == 0 || atLast {
			return errors.Combine(cErr, err)
		}
		if nextIDs, err = r.traverse(
			ctx,
			tx,
			getTraverser(clause.Params),
			resources,
		); err != nil {
			return err
		}
	}
	return nil
}

const traverseOptKey = "traverse"

func setTraverser(q query.Parameters, f Traverser) {
	q.Set(traverseOptKey, f)
}

func getTraverser(q query.Parameters) Traverser {
	return q.GetRequired(traverseOptKey).(Traverser)
}

const includeFieldDataOptKey = "includeFieldData"

func setExcludeFieldData(q query.Parameters, b bool) {
	q.Set(includeFieldDataOptKey, b)
}

func getExcludeFieldData(q query.Parameters) bool {
	v, ok := q.Get(includeFieldDataOptKey)
	if !ok {
		return false
	}
	return v.(bool)
}

func (r Retrieve) retrieveEntities(
	ctx context.Context,
	clause gorp.Retrieve[ID, Resource],
	tx gorp.Tx,
) ([]Resource, error) {
	var (
		entries          = gorp.GetEntries[ID, Resource](clause.Params)
		excludeFieldData = getExcludeFieldData(clause.Params)
		retrieveResource = !excludeFieldData
	)
	// Iterate over the entries in place, retrieving the resource if the query requires it.
	err := entries.MapInPlace(func(res Resource) (Resource, bool, error) {
		if res.ID.IsZero() {
			if !entries.IsMultiple() {
				return res, false, query.NotFound
			}
			return res, false, nil
		}
		if !retrieveResource {
			return res, true, nil
		}
		res, err := r.registrar.retrieveResource(ctx, res.ID, tx)
		if errors.Is(err, query.NotFound) && entries.IsMultiple() {
			return res, false, nil
		}
		if excludeFieldData {
			res.Data = nil
		}
		return res, true, err
	})
	return entries.All(), err
}

func (r Retrieve) traverse(
	ctx context.Context,
	tx gorp.Tx,
	traverse Traverser,
	resources []Resource,
) ([]ID, error) {
	var nextIDs []ID
	return nextIDs, gorp.NewRetrieve[[]byte, Relationship]().
		Where(func(ctx gorp.Context, rel *Relationship) (bool, error) {
			for _, resource := range resources {
				if traverse.Filter(&resource, rel) {
					nextIDs = append(nextIDs, traverse.Direction.GetID(rel))
				}
			}
			return false, nil
		}).Exec(ctx, tx)
}
