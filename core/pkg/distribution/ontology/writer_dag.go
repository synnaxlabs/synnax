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
	"maps"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

// dagWriter is a key-value backed directed acyclic graph that implements the Writer
// interface.
type dagWriter struct {
	tx        gorp.Tx
	registrar serviceRegistrar
}

var _ Writer = dagWriter{}

// ErrCycle is returned when a cycle is created in the graph.
var ErrCycle = errors.New("[ontology] - cyclic dependency")

// DefineResource implements the Writer interface.
func (d dagWriter) DefineResource(ctx context.Context, tk ID) error {
	if err := tk.Validate(); err != nil {
		return err
	}
	return gorp.NewCreate[ID, Resource]().
		Entry(&Resource{ID: tk}).
		Exec(ctx, d.tx)
}

func (d dagWriter) DefineManyResources(ctx context.Context, ids []ID) error {
	for _, id := range ids {
		if err := id.Validate(); err != nil {
			return err
		}
	}
	resources := lo.Map(ids, func(id ID, _ int) Resource { return Resource{ID: id} })
	return gorp.NewCreate[ID, Resource]().Entries(&resources).Exec(ctx, d.tx)

}

// DeleteResource implements the Writer interface.
func (d dagWriter) DeleteResource(ctx context.Context, key ID) error {
	if err := d.deleteIncomingRelationships(ctx, key); err != nil {
		return err
	}
	if err := d.deleteOutgoingRelationships(ctx, key); err != nil {
		return err
	}
	return gorp.NewDelete[ID, Resource]().WhereKeys(key).Exec(ctx, d.tx)
}

func (d dagWriter) HasResource(ctx context.Context, key ID) (bool, error) {
	return gorp.NewRetrieve[ID, Resource]().WhereKeys(key).Exists(ctx, d.tx)
}

func (d dagWriter) HasRelationship(ctx context.Context, from ID, t RelationshipType, to ID) (bool, error) {
	return d.checkRelationshipExists(ctx, Relationship{
		From: from,
		Type: t,
		To:   to,
	})
}

func (d dagWriter) DeleteManyResources(ctx context.Context, ids []ID) error {
	for _, id := range ids {
		if err := d.deleteIncomingRelationships(ctx, id); err != nil {
			return err
		}
		if err := d.deleteOutgoingRelationships(ctx, id); err != nil {
			return err
		}
	}
	return gorp.NewDelete[ID, Resource]().WhereKeys(ids...).Exec(ctx, d.tx)
}

// DefineRelationship implements the Writer interface.
func (d dagWriter) DefineRelationship(ctx context.Context, from ID, t RelationshipType, to ID) error {
	rel := Relationship{From: from, To: to, Type: t}
	exists, err := d.checkRelationshipExists(ctx, rel)
	if err != nil || exists {
		return err
	}
	if err := d.validateResourcesExist(ctx, from, to); err != nil {
		return err
	}
	descendants, err := d.retrieveDescendants(ctx, to)
	if err != nil {
		return err
	}
	if _, exists := descendants[from]; exists {
		return ErrCycle
	}
	return gorp.NewCreate[[]byte, Relationship]().Entry(&rel).Exec(ctx, d.tx)
}

// DefineFromOneToManyRelationships implements the Writer interface.
func (d dagWriter) DefineFromOneToManyRelationships(ctx context.Context, from ID, t RelationshipType, to []ID) error {
	rels := lo.Map(to, func(id ID, _ int) Relationship { return Relationship{From: from, To: id, Type: t} })
	if err := d.validateResourcesExist(ctx, from); err != nil {
		return err
	}
	if err := d.validateResourcesExist(ctx, to...); err != nil {
		return err
	}
	for _, rel := range rels {
		descendants, err := d.retrieveDescendants(ctx, rel.To)
		if err != nil {
			return err
		}
		if _, exists := descendants[from]; exists {
			return ErrCycle
		}
	}
	return gorp.NewCreate[[]byte, Relationship]().Entries(&rels).Exec(ctx, d.tx)
}

// DeleteRelationship implements the Writer interface.
func (d dagWriter) DeleteRelationship(
	ctx context.Context,
	from ID,
	t RelationshipType,
	to ID,
) error {
	return gorp.NewDelete[[]byte, Relationship]().
		WhereKeys(Relationship{From: from, To: to, Type: t}.GorpKey()).
		Exec(ctx, d.tx)
}

// NewRetrieve implements the Writer interface.
func (d dagWriter) NewRetrieve() Retrieve { return newRetrieve(d.registrar, d.tx) }

func (d dagWriter) retrieveOutgoingRelationships(ctx context.Context, key ID) ([]Resource, error) {
	var relationships []Relationship
	if err := gorp.NewRetrieve[[]byte, Relationship]().
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

func (d dagWriter) retrieveResources(ctx context.Context, ids []ID) ([]Resource, error) {
	var resources []Resource
	return resources, gorp.NewRetrieve[ID, Resource]().
		WhereKeys(ids...).
		Entries(&resources).
		Exec(ctx, d.tx)
}

func (d dagWriter) retrieveDescendants(ctx context.Context, id ID) (map[ID]Resource, error) {
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

func (d dagWriter) deleteIncomingRelationships(ctx context.Context, id ID) error {
	return gorp.NewDelete[[]byte, Relationship]().Where(func(ctx gorp.Context, rel *Relationship) (bool, error) {
		return rel.To == id, nil
	}).Exec(ctx, d.tx)
}

func (d dagWriter) deleteOutgoingRelationships(ctx context.Context, from ID) error {
	return gorp.NewDelete[[]byte, Relationship]().Where(func(ctx gorp.Context, rel *Relationship) (bool, error) {
		return rel.From == from, nil
	}).Exec(ctx, d.tx)
}

func (d dagWriter) DeleteOutgoingRelationshipsOfType(ctx context.Context, from ID, type_ RelationshipType) error {
	return gorp.NewDelete[[]byte, Relationship]().Where(func(ctx gorp.Context, rel *Relationship) (bool, error) {
		return rel.From == from && rel.Type == type_, nil
	}).Exec(ctx, d.tx)
}

func (d dagWriter) DeleteIncomingRelationshipsOfType(ctx context.Context, to ID, type_ RelationshipType) error {
	return gorp.NewDelete[[]byte, Relationship]().Where(func(ctx gorp.Context, rel *Relationship) (bool, error) {
		return rel.To == to && rel.Type == type_, nil
	}).Exec(ctx, d.tx)
}

func (d dagWriter) checkRelationshipExists(ctx context.Context, rel Relationship) (bool, error) {
	exists, err := gorp.NewRetrieve[[]byte, Relationship]().
		WhereKeys(rel.GorpKey()).
		Exists(ctx, d.tx)
	if err != nil {
		return false, err
	}
	reverseRel := Relationship{From: rel.To, To: rel.From, Type: rel.Type}
	reverseExists, err := gorp.NewRetrieve[[]byte, Relationship]().
		WhereKeys(reverseRel.GorpKey()).
		Exists(ctx, d.tx)
	if err != nil {
		return false, err
	}
	if reverseExists {
		return true, ErrCycle
	}
	return exists, nil
}

func (d dagWriter) validateResourcesExist(ctx context.Context, ids ...ID) error {
	return gorp.NewRetrieve[ID, Resource]().WhereKeys(ids...).Exec(ctx, d.tx)
}
