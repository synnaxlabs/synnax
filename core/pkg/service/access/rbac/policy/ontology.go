// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy

import (
	"context"
	"io"
	"iter"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

// OntologyType is the type of the policy ontology.
const OntologyType ontology.Type = "policy"

// OntologyID constructs a unique ontology ID for the policy with the given key.
func OntologyID(key uuid.UUID) ontology.ID {
	return ontology.ID{Type: OntologyType, Key: key.String()}
}

// OntologyIDs constructs a slice of unique ontology IDs for the policies with the given
// keys.
func OntologyIDs(keys []uuid.UUID) []ontology.ID {
	return lo.Map(keys, func(key uuid.UUID, _ int) ontology.ID {
		return OntologyID(key)
	})
}

// OntologyIDsFromPolicies constructs a slice of unique ontology IDs for the given
// policies.
func OntologyIDsFromPolicies(policies []Policy) []ontology.ID {
	return lo.Map(policies, func(p Policy, _ int) ontology.ID {
		return p.OntologyID()
	})
}

// KeysFromOntologyIDs extracts the keys from the given ontology IDs.
func KeysFromOntologyIDs(ids []ontology.ID) ([]uuid.UUID, error) {
	keys := make([]uuid.UUID, len(ids))
	var err error
	for i, id := range ids {
		if keys[i], err = uuid.Parse(id.Key); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

// TODO: fix this schema to include the constraints
var schema = zyn.Object(map[string]zyn.Schema{
	"key":    zyn.UUID(),
	"name":   zyn.String(),
	"effect": zyn.String(),
})

func newResource(p Policy) ontology.Resource {
	return ontology.NewResource(schema, OntologyID(p.Key), p.Name, p)
}

type change = xchange.Change[uuid.UUID, Policy]

var _ ontology.Service = (*Service)(nil)

// Type returns the ontology type for the policy service.
func (s *Service) Type() ontology.Type { return OntologyType }

// Schema returns the schema for the policy service.
func (s *Service) Schema() zyn.Schema { return schema }

// RetrieveResource retrieves a policy as an ontology.Resource.
func (s *Service) RetrieveResource(
	ctx context.Context,
	key string,
	tx gorp.Tx,
) (ontology.Resource, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var p Policy
	if err := s.NewRetrieve().WhereKeys(k).Entry(&p).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(p), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key),
		Value:   newResource(c.Value),
	}
}

// OnChange listens for changes to policies.
func (s *Service) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, Policy]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return gorp.Observe[uuid.UUID, Policy](s.cfg.DB).OnChange(handleChange)
}

// OpenNexter opens a nexter to iterate over policies.
func (s *Service) OpenNexter(
	ctx context.Context,
) (iter.Seq[ontology.Resource], io.Closer, error) {
	n, closer, err := gorp.WrapReader[uuid.UUID, Policy](s.cfg.DB).OpenNexter(ctx)
	if err != nil {
		return nil, nil, err
	}
	return xiter.Map(n, newResource), closer, nil
}
