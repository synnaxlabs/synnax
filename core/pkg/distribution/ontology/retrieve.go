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
	"bytes"
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/encoding/orc"
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
// relationships in the ontology.
type Retrieve struct {
	clauses           []clause
	registrar         serviceRegistrar
	tx                gorp.Tx
	resourceTable     *gorp.Table[string, Resource]
	relationshipTable *gorp.Table[[]byte, Relationship]
}

func (r Retrieve) nextClause() Retrieve {
	c := clause{Retrieve: r.resourceTable.NewRetrieve()}
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
func (o *Ontology) NewRetrieve() Retrieve {
	return newRetrieve(o.registrar, o.DB, o.resourceTable, o.relationshipTable)
}

func newRetrieve(
	registrar serviceRegistrar,
	tx gorp.Tx,
	resourceTable *gorp.Table[string, Resource],
	relationshipTable *gorp.Table[[]byte, Relationship],
) Retrieve {
	r := Retrieve{
		registrar:         registrar,
		tx:                tx,
		resourceTable:     resourceTable,
		relationshipTable: relationshipTable,
	}
	return r.nextClause()
}

// WhereIDs filters resources by the provided keys.
func (r Retrieve) WhereIDs(ids ...ID) Retrieve {
	c := r.currentClause()
	c.Retrieve = c.WhereKeys(IDsToKeys(ids)...)
	return r.setCurrentClause(c)
}

// Where filters resources by the provided predicate.
func (r Retrieve) Where(filter gorp.FilterFunc[string, Resource]) Retrieve {
	c := r.currentClause()
	c.Retrieve = c.Where(filter)
	return r.setCurrentClause(c)
}

func (r Retrieve) WhereTypes(types ...ResourceType) Retrieve {
	c := r.currentClause()
	if len(types) == 1 {
		c.Retrieve = c.WherePrefix([]byte(types[0].String()))
	} else {
		c.Retrieve = c.Where(func(_ gorp.Context, r *Resource) (bool, error) {
			return lo.Contains(types, r.ID.Type), nil
		})
	}
	return r.setCurrentClause(c)
}

// Limit limits the number of results returned.
func (r Retrieve) Limit(limit int) Retrieve {
	c := r.currentClause()
	c.Retrieve = c.Limit(limit)
	return r.setCurrentClause(c)
}

// Offset offsets the results returned.
func (r Retrieve) Offset(offset int) Retrieve {
	c := r.currentClause()
	c.Retrieve = c.Offset(offset)
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
	// DirectionForward represents a forward traversal i.e. Start -> To.
	DirectionForward Direction = iota + 1
	// DirectionBackward represents a backward traversal i.e. To -> Start.
	DirectionBackward Direction = 2
)

// GetID returns the directional ID of the relationship.
func (d Direction) GetID(rel *Relationship) ID {
	return lo.Ternary(d == DirectionForward, rel.To, rel.From)
}

// RawTraversal is a callback that operates on raw orc-encoded relationship bytes.
// It checks whether the row matches any of the target IDs and, if so, appends the
// resulting ID to nextIDs. This avoids decoding the relationship entirely.
type RawTraversal func(data []byte, nextIDs *[]ID) error

// RelationshipPrefix returns a FilterPrefix function that scopes traversal queries
// to relationships of the given type originating from a specific resource.
func RelationshipPrefix(relType RelationshipType) func(id ID) []byte {
	suffix := []byte("->" + string(relType) + "->")
	return func(id ID) []byte {
		idStr := id.String()
		prefix := make([]byte, 0, len(idStr)+len(suffix))
		prefix = append(prefix, idStr...)
		prefix = append(prefix, suffix...)
		return prefix
	}
}

// ReadRawID reads a Type+Key ID pair from the current position in raw data.
func ReadRawID(r orc.Raw) ID {
	t, r := r.ReadString()
	k, _ := r.ReadString()
	return ID{Type: ResourceType(t), Key: string(k)}
}

// Traverser is a struct that defines the traversal of a relationship between entities
// in the ontology.
type Traverser struct {
	// FilterPrefix is an optional function that returns a prefix for efficient lookup.
	// If nil, a full table scan will be used.
	FilterPrefix func(id ID) []byte
	// Traverse builds a raw byte callback for the given IDs that extracts
	// matching relationship targets directly from encoded bytes without decoding.
	// When set, traverseByScan uses this instead of Filter.
	Traverse func(ids []ID) RawTraversal
	// Direction is the direction of the traversal. See (Direction) for more.
	Direction Direction
}

var (
	relationshipTypeParentOfBytes = []byte(RelationshipTypeParentOf)
	// ParentsTraverser traverses to the parents of a resource.
	ParentsTraverser = Traverser{
		Traverse: func(ids []ID) RawTraversal {
			w := orc.NewWriter(64)
			encoded := make([][]byte, len(ids))
			for i, id := range ids {
				w.Reset()
				w.String(string(id.Type))
				w.String(id.Key)
				encoded[i] = w.Copy()
			}
			return func(data []byte, nextIDs *[]ID) error {
				raw, err := orc.NewRaw(data)
				if err != nil {
					return err
				}
				fromType, r := raw.ReadString()
				fromKey, r := r.ReadString()
				relType, r := r.ReadString()
				if bytes.Equal(relType, relationshipTypeParentOfBytes) {
					for _, enc := range encoded {
						if bytes.HasPrefix(r, enc) {
							*nextIDs = append(*nextIDs, ID{
								Type: ResourceType(fromType),
								Key:  string(fromKey),
							})
						}
					}
				}
				return nil
			}
		},
		Direction: DirectionBackward,
	}
	// ChildrenTraverser traverse to the children of a resource.
	ChildrenTraverser = Traverser{
		Traverse: func(_ []ID) RawTraversal {
			return func(data []byte, nextIDs *[]ID) error {
				reader, err := orc.NewRaw(data)
				if err != nil {
					return err
				}
				*nextIDs = append(*nextIDs, ReadRawID(reader.SkipStrings(3)))
				return nil
			}
		},
		Direction:    DirectionForward,
		FilterPrefix: RelationshipPrefix(RelationshipTypeParentOf),
	}
)

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
	c.Retrieve = c.Entry(res)
	return r.setCurrentClause(c)
}

// Entries binds a slice that the query will fill results into. Calls to Entry will
// override all previous calls to Entries or Entry.
func (r Retrieve) Entries(res *[]Resource) Retrieve {
	c := r.currentClause()
	c.Retrieve = c.Entries(res)
	return r.setCurrentClause(c)
}

// Exec executes the query.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	var nextIDs []ID
	tx = gorp.OverrideTx(r.tx, tx)
	for i, cls := range r.clauses {
		if i != 0 {
			cls.Retrieve = cls.WhereKeys(IDsToKeys(nextIDs)...)
		}
		atLast := len(r.clauses) == i+1
		entriesBound := cls.GetEntries().Bound()
		// If we only have keys and no filters, and don't need entries, skip execution
		// entirely and use the keys directly.
		if canSkipExec(cls, entriesBound, atLast) {
			nextIDs = lo.Must(ParseIDs(cls.GetWhereKeys()))
		} else {
			// For intermediate clauses that don't have user-bound entries, we need to
			// bind a temporary slice so gorp can store the query results. Without this,
			// gorp.Retrieve.Replace/Add silently drops results when entries aren't
			// bound.
			if !atLast && !entriesBound {
				cls.Retrieve = cls.Entries(&[]Resource{})
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
	return !entriesBound && !atLast && q.HasWhereKeys() &&
		!q.HasFilters() && !q.HasLimit() && !q.HasOffset()
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
	if !entries.Bound() {
		return entries.All(), nil
	}
	// Iterate over the entries in place, retrieving the resource if the query requires it.
	err := entries.MapInPlace(func(res Resource) (Resource, bool, error) {
		if res.ID.IsZero() {
			if !entries.IsMultiple() {
				return res, false, query.ErrNotFound
			}
			return res, false, nil
		}
		if !retrieveResource {
			return res, true, nil
		}
		res, err := r.registrar.retrieveResource(ctx, res.ID, tx)
		if errors.Is(err, query.ErrNotFound) && entries.IsMultiple() {
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
	if traverse.FilterPrefix != nil {
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
	rt := traverse.Traverse(ids)
	for _, id := range ids {
		q := r.relationshipTable.NewRetrieve().
			WherePrefix(traverse.FilterPrefix(id)).
			WhereRaw(func(data []byte) (bool, error) {
				return false, rt(data, &nextIDs)
			})
		if err := q.Exec(ctx, tx); err != nil {
			return nil, err
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
		nextIDs = make([]ID, 0, len(ids)*4)
		rt      = traverse.Traverse(ids)
		q       = r.relationshipTable.NewRetrieve().
			WhereRaw(func(data []byte) (bool, error) {
				return false, rt(data, &nextIDs)
			})
	)
	if err := q.Exec(ctx, tx); err != nil {
		return nil, err
	}
	return nextIDs, nil
}
