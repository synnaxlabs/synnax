// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/core"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

const ontologyType ontology.Type = "role"

const (
	// HasRole indicates that a resource is labeled by another resource. When
	// examining a Relationship of type HasRole, the Start field will be the
	// resource that is labeled and the To field will be the resource that is
	// doing the labeling (i.e. Start HasRole To).
	HasRole ontology.RelationshipType = "has_role"
)

// Roles is an ontology.Traverser that allows the caller to traverse an ontology.Retrieve
// query to find all the roles for a particular subject. Pass this traverser to
// ontology.Retrieve.TraverseTo.
var (
	Roles = ontology.Traverser{
		Filter: func(res *ontology.Resource, rel *ontology.Relationship) bool {
			return rel.Type == HasRole && rel.From == res.ID
		},
		Direction: ontology.Forward,
	}
)

// OntologyID constructs a unique ontology.ID for the Role with the given key.
func OntologyID(k uuid.UUID) ontology.ID {
	return ontology.ID{Type: ontologyType, Key: k.String()}
}

// OntologyIDs constructs a slice of unique ontology.ResourceIDs for the Roles with the given
// keys.
func OntologyIDs(keys []uuid.UUID) []ontology.ID {
	return lo.Map(keys, func(k uuid.UUID, _ int) ontology.ID { return OntologyID(k) })
}

// OntologyIDsFromPolicies constructs a slice of unique ontology.ResourceIDs for the given Roles.
func OntologyIDsFromPolicies(policies []Role) []ontology.ID {
	return lo.Map(policies, func(l Role, _ int) ontology.ID { return OntologyID(l.Key) })
}

// KeysFromOntologyIds extracts the Role keys from the given ontology.ResourceIDs.
func KeysFromOntologyIds(ids []ontology.ID) (keys []uuid.UUID, err error) {
	keys = make([]uuid.UUID, len(ids))
	for i, id := range ids {
		keys[i], err = uuid.Parse(id.Key)
		if err != nil {
			return nil, err
		}
	}
	return keys, nil
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":  zyn.UUID(),
	"name": zyn.String(),
})

func newResource(l Role) ontology.Resource {
	return core.NewResource(schema, OntologyID(l.Key), l.Name, l)
}

type change = changex.Change[uuid.UUID, Role]

func (s *Service) Type() ontology.Type { return ontologyType }

// Schema implements ontology.Service.
func (s *Service) Schema() zyn.Schema { return schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(
	ctx context.Context,
	key string,
	tx gorp.Tx,
) (ontology.Resource, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var l Role
	if err := s.NewRetrieve().WhereKeys(k).Entry(&l).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(l), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(ctx context.Context, nexter iter.Nexter[ontology.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, Role]) {
		f(ctx, iter.NexterTranslator[change, ontology.Change]{
			Wrap:      reader,
			Translate: translateChange,
		})
	}
	return gorp.Observe[uuid.UUID, Role](s.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter() (iter.NexterCloser[ontology.Resource], error) {
	n, err := gorp.WrapReader[uuid.UUID, Role](s.DB).OpenNexter()
	return iter.NexterCloserTranslator[Role, ontology.Resource]{Wrap: n, Translate: newResource}, err
}
