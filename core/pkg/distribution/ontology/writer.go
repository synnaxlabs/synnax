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
	resourceTable     *gorp.Table[string, Resource]
	relationshipTable *gorp.Table[string, Relationship]
}

// DefineResource defines one or more new resources with the given IDs. If any of the
// resources already exist, DefineResource does nothing for those. Returns nil if no IDs
// are provided.
func (w Writer) DefineResource(ctx context.Context, ids ...ID) error {
	resources, err := lo.MapErr(ids, func(id ID, _ int) (Resource, error) {
		if id.Key == "" {
			return Resource{}, errors.Wrapf(validate.ErrValidation, "key is required")
		}
		if err := id.Validate(); err != nil {
			return Resource{}, err
		}
		return Resource{ID: id}, nil
	})
	if err != nil {
		return err
	}
	return w.resourceTable.NewCreate().Entries(&resources).Exec(ctx, w.tx)
}

// DeleteResource deletes one or more resources with the given IDs along with all of
// their incoming and outgoing relationships. If any of the resources do not exist,
// DeleteResource does nothing for those. Returns nil if no IDs are provided.
func (w Writer) DeleteResource(ctx context.Context, ids ...ID) error {
	for _, id := range ids {
		if err := w.deleteIncomingRelationships(ctx, id); err != nil {
			return err
		}
		if err := w.deleteOutgoingRelationships(ctx, id); err != nil {
			return err
		}
	}
	return w.resourceTable.NewDelete().
		Where(gorp.MatchKeys[string, Resource](IDsToKeys(ids)...)).Exec(ctx, w.tx)
}

func (w Writer) HasRelationship(ctx context.Context, from ID, t RelationshipType, to ID) (bool, error) {
	return w.checkRelationshipExists(ctx, Relationship{
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
func (w Writer) DefineRelationship(
	ctx context.Context,
	from ID,
	t RelationshipType,
	to ...ID,
) error {
	if len(to) == 0 {
		return nil
	}
	to = lo.Uniq(to)
	if err := w.validateResourcesExist(ctx, from); err != nil {
		return err
	}
	if err := w.validateResourcesExist(ctx, to...); err != nil {
		return err
	}
	rels := make([]Relationship, 0, len(to))
	for _, id := range to {
		rel := Relationship{From: from, To: id, Type: t}
		exists, err := w.checkRelationshipExists(ctx, rel)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		descendants, err := w.retrieveDescendants(ctx, id)
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
	return w.relationshipTable.NewCreate().Entries(&rels).Exec(ctx, w.tx)
}

// DeleteRelationship deletes the relationship with the given keys and type. If the
// relationship does not exist, DeleteRelationship does nothing.
func (w Writer) DeleteRelationship(
	ctx context.Context,
	from ID,
	t RelationshipType,
	to ID,
) error {
	return w.relationshipTable.NewDelete().Where(gorp.MatchKeys[string, Relationship](Relationship{From: from, To: to, Type: t}.GorpKey())).
		Exec(ctx, w.tx)
}

func (w Writer) retrieveOutgoingRelationships(ctx context.Context, key ID) ([]Resource, error) {
	var relationships []Relationship
	if err := w.relationshipTable.NewRetrieve().
		WherePrefix([]byte(key.String())).
		Entries(&relationships).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	var keys []ID
	for _, rel := range relationships {
		keys = append(keys, rel.To)
	}
	return w.retrieveResources(ctx, keys)
}

func (w Writer) retrieveResources(ctx context.Context, ids []ID) ([]Resource, error) {
	var resources []Resource
	if err := w.resourceTable.NewRetrieve().Where(gorp.MatchKeys[string, Resource](IDsToKeys(ids)...)).
		Entries(&resources).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	return resources, nil
}

func (w Writer) retrieveDescendants(ctx context.Context, id ID) (map[ID]Resource, error) {
	descendants := make(map[ID]Resource)
	children, err := w.retrieveOutgoingRelationships(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(children) == 0 {
		return nil, nil
	}
	for _, child := range children {
		childDescendants, err := w.retrieveDescendants(ctx, child.ID)
		if err != nil {
			return nil, err
		}
		maps.Copy(descendants, childDescendants)
		descendants[child.ID] = child
	}
	return descendants, nil
}

func (w Writer) deleteIncomingRelationships(ctx context.Context, id ID) error {
	suffix := []byte(relationshipKeySep + id.String())
	return w.relationshipTable.NewDelete().
		WhereRaw(func(key, _ []byte) (bool, error) {
			return bytes.HasSuffix(key, suffix), nil
		}).
		Exec(ctx, w.tx)
}

func (w Writer) deleteOutgoingRelationships(ctx context.Context, from ID) error {
	return w.relationshipTable.NewDelete().
		WherePrefix([]byte(from.String()+relationshipKeySep)).
		Exec(ctx, w.tx)
}

// DeleteOutgoingRelationshipsOfType deletes all outgoing relationships of the given
// type from the resource with the given ID. If the resource does not exist, or if it
// has no outgoing relationships of the given type, DeleteOutgoingRelationshipsOfType
// does nothing.
func (w Writer) DeleteOutgoingRelationshipsOfType(ctx context.Context, from ID, relationshipType RelationshipType) error {
	prefix := from.String() + relationshipKeySep + string(relationshipType) + relationshipKeySep
	return w.relationshipTable.NewDelete().
		WherePrefix([]byte(prefix)).
		Exec(ctx, w.tx)
}

// DeleteIncomingRelationshipsOfType deletes all incoming relationships of the given
// type to the resource with the given ID. If the resource does not exist, or if it has
// no incoming relationships of the given type, DeleteIncomingRelationshipsOfType does
// nothing.
func (w Writer) DeleteIncomingRelationshipsOfType(ctx context.Context, to ID, relationshipType RelationshipType) error {
	suffix := []byte(relationshipKeySep + string(relationshipType) + relationshipKeySep + to.String())
	return w.relationshipTable.NewDelete().
		WhereRaw(func(key, _ []byte) (bool, error) {
			return bytes.HasSuffix(key, suffix), nil
		}).
		Exec(ctx, w.tx)
}

func (w Writer) checkRelationshipExists(ctx context.Context, rel Relationship) (bool, error) {
	exists, err := w.relationshipTable.NewRetrieve().Where(gorp.MatchKeys[string, Relationship](rel.GorpKey())).
		Exists(ctx, w.tx)
	if err != nil {
		return false, err
	}
	reverseRel := Relationship{From: rel.To, To: rel.From, Type: rel.Type}
	reverseExists, err := w.relationshipTable.NewRetrieve().Where(gorp.MatchKeys[string, Relationship](reverseRel.GorpKey())).
		Exists(ctx, w.tx)
	if err != nil {
		return false, err
	}
	if reverseExists {
		return true, graph.ErrCyclicDependency
	}
	return exists, nil
}

func (w Writer) validateResourcesExist(ctx context.Context, ids ...ID) error {
	return w.resourceTable.NewRetrieve().Where(gorp.MatchKeys[string, Resource](IDsToKeys(ids)...)).Exec(ctx, w.tx)
}
