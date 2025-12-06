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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/core"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

type clause struct {
	gorp.Retrieve[string, Resource]
	traverser        Traverser
	excludeFieldData bool
}

// Retrieve implements a set of methods for retrieving resources and traversing their
// relationships in teh ontology.
type Retrieve struct {
	clauses   []clause
	registrar serviceRegistrar
	tx        gorp.Tx
}

func (r Retrieve) nextClause() Retrieve {
	c := clause{Retrieve: gorp.NewRetrieve[string, Resource]()}
	r.clauses = append(r.clauses, c)
	return r
}

func (r Retrieve) currentClause() clause {
	return r.clauses[len(r.clauses)-1]
}

func (r Retrieve) setCurrentClause(c clause) Retrieve {
	r.clauses[len(r.clauses)-1] = c
	return r
}

// NewRetrieve opens a new Retrieve query, which can be used to traverse and read resources
// from the underlying ontology.
func (o *Ontology) NewRetrieve() Retrieve { return newRetrieve(o.registrar, o.DB) }

func newRetrieve(registrar serviceRegistrar, tx gorp.Tx) Retrieve {
	r := Retrieve{
		registrar: registrar,
		tx:        tx,
	}
	return r.nextClause()
}

// WhereIDs filters resources by the provided keys.
func (r Retrieve) WhereIDs(ids ...ID) Retrieve {
	c := r.currentClause()
	c.Retrieve = c.Retrieve.WhereKeys(IDsToString(ids)...)
	return r.setCurrentClause(c)
}

// Where filters resources by the provided predicate.
func (r Retrieve) Where(filter gorp.FilterFunc[string, Resource]) Retrieve {
	c := r.currentClause()
	c.Retrieve = c.Retrieve.Where(filter)
	return r.setCurrentClause(c)
}

func (r Retrieve) WhereTypes(types ...Type) Retrieve {
	c := r.currentClause()
	c.Retrieve = c.Retrieve.Where(func(ctx gorp.Context, r *Resource) (bool, error) {
		return lo.Contains(types, r.ID.Type), nil
	})
	return r.setCurrentClause(c)
}

// Limit limits the number of results returned.
func (r Retrieve) Limit(limit int) Retrieve {
	c := r.currentClause()
	c.Retrieve = c.Retrieve.Limit(limit)
	return r.setCurrentClause(c)
}

// Offset offsets the results returned.
func (r Retrieve) Offset(offset int) Retrieve {
	c := r.currentClause()
	c.Retrieve = c.Retrieve.Offset(offset)
	return r.setCurrentClause(c)
}

// ExcludeFieldData includes the field data of the resource in the results based on the
// provided predicate.
func (r Retrieve) ExcludeFieldData(excludeFieldData bool) Retrieve {
	c := r.currentClause()
	c.excludeFieldData = excludeFieldData
	return r.setCurrentClause(c)
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
	// Prefix is an optional function that returns a prefix for efficient lookup.
	// If nil, a full table scan will be used.
	Prefix func(id ID) []byte
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
		Prefix:    childrenPrefix,
	}
	childrenPrefixSuffix = []byte("->" + string(ParentOf) + "->")
)

func childrenPrefix(id ID) []byte {
	idStr := id.String()
	prefix := make([]byte, 0, len(idStr)+len(childrenPrefixSuffix))
	prefix = append(prefix, idStr...)
	prefix = append(prefix, childrenPrefixSuffix...)
	return prefix
}

// TraverseTo traverses to the provided relationship type. All filtering methods will
// now be applied to elements of the traversed relationship.
func (r Retrieve) TraverseTo(t Traverser) Retrieve {
	c := r.currentClause()
	c.traverser = t
	r = r.setCurrentClause(c)
	return r.nextClause()
}

// Entry binds the entry that the query will fill results into. Calls to Entry will
// override all previous calls to Entries or Entry.
func (r Retrieve) Entry(res *Resource) Retrieve {
	c := r.currentClause()
	c.Retrieve = c.Retrieve.Entry(res)
	return r.setCurrentClause(c)
}

// Entries binds a slice that the query will fill results into. Calls to Entry will
// override all previous calls to Entries or Entry.
func (r Retrieve) Entries(res *[]Resource) Retrieve {
	c := r.currentClause()
	c.Retrieve = c.Retrieve.Entries(res)
	return r.setCurrentClause(c)
}

// Exec executes the query.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	var nextIDs []ID
	tx = gorp.OverrideTx(r.tx, tx)
	for i, cls := range r.clauses {
		if i != 0 {
			cls.Retrieve = cls.Retrieve.WhereKeys(IDsToString(nextIDs)...)
		}
		atLast := len(r.clauses) == i+1
		entriesBound := cls.GetEntries().Bound()
		// If we only have keys and no filters, and don't need entries, skip execution
		// entirely and use the keys directly.
		if canSkipExec(cls, entriesBound, atLast) {
			nextIDs = lo.Must(core.ParseIDs(cls.GetWhereKeys()))
		} else {
			// For intermediate clauses that don't have user-bound entries, we need to
			// bind a temporary slice so gorp can store the query results. Without this,
			// gorp.Retrieve.Replace/Add silently drops results when entries aren't bound.
			if !atLast && !entriesBound {
				cls.Retrieve = cls.Retrieve.Entries(&[]Resource{})
			}
			cErr := cls.Exec(ctx, tx)
			if atLast || entriesBound {
				resources, err := r.retrieveEntities(ctx, cls, tx)
				if cErr != nil || err != nil || len(resources) == 0 || atLast {
					return errors.Combine(cErr, err)
				}
				nextIDs = ResourceIDs(resources)
			} else {
				ids := r.extractIDs(cls.Retrieve)
				if cErr != nil || len(ids) == 0 {
					return cErr
				}
				nextIDs = ids
			}
		}
		var err error
		if nextIDs, err = r.traverse(
			ctx,
			tx,
			cls.traverser,
			nextIDs,
		); err != nil {
			return err
		}
	}
	return nil
}

func canSkipExec(q clause, entriesBound, atLast bool) bool {
	return !entriesBound && !atLast && q.HasWhereKeys() && !q.HasFilters() && !q.HasLimit() && !q.HasOffset()
}

func (r Retrieve) retrieveEntities(
	ctx context.Context,
	clause clause,
	tx gorp.Tx,
) ([]Resource, error) {
	var (
		entries          = clause.GetEntries()
		retrieveResource = !clause.excludeFieldData
	)
	if !entries.Any() {
		return entries.All(), nil
	}
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
		if clause.excludeFieldData {
			res.Data = nil
		}
		return res, true, err
	})
	return entries.All(), err
}

func (r Retrieve) extractIDs(clause gorp.Retrieve[string, Resource]) []ID {
	entries := clause.GetEntries()
	resources := entries.All()
	ids := make([]ID, 0, len(resources))
	for _, res := range resources {
		if !res.ID.IsZero() {
			ids = append(ids, res.ID)
		}
	}
	return ids
}

func (r Retrieve) traverse(
	ctx context.Context,
	tx gorp.Tx,
	traverse Traverser,
	ids []ID,
) ([]ID, error) {
	if traverse.Prefix != nil {
		return r.traverseByPrefix(ctx, tx, traverse, ids)
	}
	return r.traverseByScan(ctx, tx, traverse, ids)
}

func (r Retrieve) traverseByPrefix(
	ctx context.Context,
	tx gorp.Tx,
	traverse Traverser,
	ids []ID,
) ([]ID, error) {
	nextIDs := make([]ID, 0, len(ids)*4)
	relationships := make([]Relationship, 0, 16)
	for _, id := range ids {
		relationships = relationships[:0]
		if err := gorp.NewRetrieve[[]byte, Relationship]().
			WherePrefix(traverse.Prefix(id)).
			Entries(&relationships).
			Exec(ctx, tx); err != nil {
			return nil, err
		}
		for i := range relationships {
			nextIDs = append(nextIDs, traverse.Direction.GetID(&relationships[i]))
		}
	}
	return nextIDs, nil
}

func (r Retrieve) traverseByScan(
	ctx context.Context,
	tx gorp.Tx,
	traverse Traverser,
	ids []ID,
) ([]ID, error) {
	var (
		nextIDs       = make([]ID, 0, len(ids)*4)
		relationships []Relationship
	)
	err := gorp.NewRetrieve[[]byte, Relationship]().
		Entries(&relationships).
		Where(func(ctx gorp.Context, rel *Relationship) (bool, error) {
			for _, id := range ids {
				res := Resource{ID: id}
				if traverse.Filter(&res, rel) {
					nextIDs = append(nextIDs, traverse.Direction.GetID(rel))
				}
			}
			return false, nil
		}).Exec(ctx, tx)
	return nextIDs, err
}
