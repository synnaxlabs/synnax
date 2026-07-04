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
	"maps"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/graph"
	"github.com/synnaxlabs/x/validate"
)

// Writer defines and deletes resources and relationships within the ontology. It is a
// key-value backed directed acyclic graph. Open one with Ontology.NewWriter.
type Writer struct {
	tx                gorp.Tx
	registrar         serviceRegistrar
	resourceTable     *gorp.Table[string, Resource]
	relationshipTable *gorp.Table[string, Relationship]
	relIndexes        relationshipIndexes
}

// DefineResource defines one or more new resources with the given IDs. If any of the
// resources already exist, DefineResource does nothing for those. Returns nil if no IDs
// are provided.
func (d Writer) DefineResource(ctx context.Context, ids ...ID) error {
	if len(ids) == 0 {
		return nil
	}
	for _, id := range ids {
		if id.Key == "" {
			return errors.Wrapf(validate.ErrValidation, "key is required")
		}
		if err := id.Validate(); err != nil {
			return err
		}
	}
	resources := lo.Map(ids, func(id ID, _ int) Resource { return Resource{ID: id} })
	return d.resourceTable.NewCreate().Entries(&resources).Exec(ctx, d.tx)
}

// DeleteResource deletes one or more resources with the given IDs along with all of
// their incoming and outgoing relationships. If any of the resources do not exist,
// DeleteResource does nothing for those. Returns nil if no IDs are provided.
func (d Writer) DeleteResource(ctx context.Context, ids ...ID) error {
	if len(ids) == 0 {
		return nil
	}
	for _, id := range ids {
		if err := d.deleteIncomingRelationships(ctx, id); err != nil {
			return err
		}
		if err := d.deleteOutgoingRelationships(ctx, id); err != nil {
			return err
		}
	}
	return d.resourceTable.NewDelete().
		Where(gorp.MatchKeys[string, Resource](IDsToKeys(ids)...)).Exec(ctx, d.tx)
}

// HasResource returns true if the resource with the given ID exists.
func (d Writer) HasResource(ctx context.Context, id ID) (bool, error) {
	return d.resourceTable.NewRetrieve().Where(gorp.MatchKeys[string, Resource](id.String())).Exists(ctx, d.tx)
}

func (d Writer) HasRelationship(ctx context.Context, from ID, t RelationshipType, to ID) (bool, error) {
	return d.checkRelationshipExists(ctx, Relationship{
		From: from,
		Type: t,
		To:   to,
	})
}

// DefineRelationship defines a directional relationship of type t from the resource
// with the given from ID to one or more to IDs. Already-existing relationships are
// silently skipped. Returns graph.ErrCyclicDependency if any of the new relationships
// would create a cycle (including the case where the reverse-direction relationship
// already exists). Returns nil if no to IDs are provided.
func (d Writer) DefineRelationship(
	ctx context.Context,
	from ID,
	t RelationshipType,
	to ...ID,
) error {
	if len(to) == 0 {
		return nil
	}
	to = lo.Uniq(to)
	if err := d.validateResourcesExist(ctx, from); err != nil {
		return err
	}
	if err := d.validateResourcesExist(ctx, to...); err != nil {
		return err
	}
	rels := make([]Relationship, 0, len(to))
	for _, id := range to {
		rel := Relationship{From: from, To: id, Type: t}
		exists, err := d.checkRelationshipExists(ctx, rel)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		descendants, err := d.retrieveDescendants(ctx, id)
		if err != nil {
			return err
		}
		if _, cyclic := descendants[from]; cyclic {
			return graph.ErrCyclicDependency
		}
		rels = append(rels, rel)
	}
	if len(rels) == 0 {
		return nil
	}
	return d.relationshipTable.NewCreate().Entries(&rels).Exec(ctx, d.tx)
}

// DeleteRelationship deletes the relationship with the given keys and type. If the
// relationship does not exist, DeleteRelationship does nothing.
func (d Writer) DeleteRelationship(
	ctx context.Context,
	from ID,
	t RelationshipType,
	to ID,
) error {
	return d.relationshipTable.NewDelete().Where(gorp.MatchKeys[string, Relationship](Relationship{From: from, To: to, Type: t}.GorpKey())).
		Exec(ctx, d.tx)
}

// NewRetrieve opens a new Retrieve query that provides a view of pending operations
// merged with the underlying database. If the Writer is executing directly against the
// underlying database, the Retrieve query behaves exactly as if calling
// Ontology.NewRetrieve.
func (d Writer) NewRetrieve() Retrieve {
	return newRetrieve(d.registrar, d.tx, d.resourceTable, d.relationshipTable, d.relIndexes)
}

func (d Writer) retrieveOutgoingRelationships(ctx context.Context, key ID) ([]Resource, error) {
	var relationships []Relationship
	if err := d.relationshipTable.NewRetrieve().
		WherePrefix([]byte(key.String())).
		Entries(&relationships).
		Exec(ctx, d.tx); err != nil {
		return nil, err
	}
	var keys []ID
	for _, rel := range relationships {
		keys = append(keys, rel.To)
	}
	return d.retrieveResources(ctx, keys)
}

func (d Writer) retrieveResources(ctx context.Context, ids []ID) ([]Resource, error) {
	var resources []Resource
	if err := d.resourceTable.NewRetrieve().Where(gorp.MatchKeys[string, Resource](IDsToKeys(ids)...)).
		Entries(&resources).
		Exec(ctx, d.tx); err != nil {
		return nil, err
	}
	return resources, nil
}

func (d Writer) retrieveDescendants(ctx context.Context, id ID) (map[ID]Resource, error) {
	descendants := make(map[ID]Resource)
	children, err := d.retrieveOutgoingRelationships(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(children) == 0 {
		return nil, nil
	}
	for _, child := range children {
		childDescendants, err := d.retrieveDescendants(ctx, child.ID)
		if err != nil {
			return nil, err
		}
		maps.Copy(descendants, childDescendants)
		descendants[child.ID] = child
	}
	return descendants, nil
}

func (d Writer) deleteIncomingRelationships(ctx context.Context, id ID) error {
	suffix := []byte(relationshipKeySep + id.String())
	return d.relationshipTable.NewDelete().
		WhereRaw(func(key, _ []byte) (bool, error) {
			return bytes.HasSuffix(key, suffix), nil
		}).
		Exec(ctx, d.tx)
}

func (d Writer) deleteOutgoingRelationships(ctx context.Context, from ID) error {
	return d.relationshipTable.NewDelete().
		WherePrefix([]byte(from.String()+relationshipKeySep)).
		Exec(ctx, d.tx)
}

// DeleteOutgoingRelationshipsOfType deletes all outgoing relationships of the given type
// from the resource with the given ID. If the resource does not exist, or if it has no
// outgoing relationships of the given type, DeleteOutgoingRelationshipsOfType does
// nothing.
func (d Writer) DeleteOutgoingRelationshipsOfType(ctx context.Context, from ID, relationshipType RelationshipType) error {
	prefix := from.String() + relationshipKeySep + string(relationshipType) + relationshipKeySep
	return d.relationshipTable.NewDelete().
		WherePrefix([]byte(prefix)).
		Exec(ctx, d.tx)
}

// DeleteIncomingRelationshipsOfType deletes all incoming relationships of the given type
// to the resource with the given ID. If the resource does not exist, or if it has no
// incoming relationships of the given type, DeleteIncomingRelationshipsOfType does
// nothing.
func (d Writer) DeleteIncomingRelationshipsOfType(ctx context.Context, to ID, relationshipType RelationshipType) error {
	suffix := []byte(relationshipKeySep + string(relationshipType) + relationshipKeySep + to.String())
	return d.relationshipTable.NewDelete().
		WhereRaw(func(key, _ []byte) (bool, error) {
			return bytes.HasSuffix(key, suffix), nil
		}).
		Exec(ctx, d.tx)
}

func (d Writer) checkRelationshipExists(ctx context.Context, rel Relationship) (bool, error) {
	exists, err := d.relationshipTable.NewRetrieve().Where(gorp.MatchKeys[string, Relationship](rel.GorpKey())).
		Exists(ctx, d.tx)
	if err != nil {
		return false, err
	}
	reverseRel := Relationship{From: rel.To, To: rel.From, Type: rel.Type}
	reverseExists, err := d.relationshipTable.NewRetrieve().Where(gorp.MatchKeys[string, Relationship](reverseRel.GorpKey())).
		Exists(ctx, d.tx)
	if err != nil {
		return false, err
	}
	if reverseExists {
		return true, graph.ErrCyclicDependency
	}
	return exists, nil
}

func (d Writer) validateResourcesExist(ctx context.Context, ids ...ID) error {
	return d.resourceTable.NewRetrieve().Where(gorp.MatchKeys[string, Resource](IDsToKeys(ids)...)).Exec(ctx, d.tx)
}
