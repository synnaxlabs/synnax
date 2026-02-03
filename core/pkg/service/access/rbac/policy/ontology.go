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

const OntologyType ontology.Type = "policy"

// OntologyID constructs a unique ontology.ID for the Policy with the given key.
func OntologyID(k uuid.UUID) ontology.ID {
	return ontology.ID{Type: OntologyType, Key: k.String()}
}

// OntologyIDs constructs a slice of unique ontology.IDs for the Policys with the given
// keys.
func OntologyIDs(keys []uuid.UUID) []ontology.ID {
	return lo.Map(keys, func(k uuid.UUID, _ int) ontology.ID { return OntologyID(k) })
}

// OntologyIDsFromPolicies constructs a slice of unique ontology.IDs for the given Policys.
func OntologyIDsFromPolicies(policies []Policy) []ontology.ID {
	return lo.Map(policies, func(l Policy, _ int) ontology.ID { return OntologyID(l.Key) })
}

// KeysFromOntologyIDs extracts the Policy keys from the given ontology.IDs.
func KeysFromOntologyIDs(ids []ontology.ID) ([]uuid.UUID, error) {
	keys := make([]uuid.UUID, len(ids))
	var err error
	for i, id := range ids {
		keys[i], err = uuid.Parse(id.Key)
		if err != nil {
			return nil, err
		}
	}
	return keys, nil
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":      zyn.UUID(),
	"name":     zyn.String(),
	"internal": zyn.Bool(),
})

func newResource(p Policy) ontology.Resource {
	return ontology.NewResource(schema, OntologyID(p.Key), p.Name, p)
}

type change = xchange.Change[uuid.UUID, Policy]

func (s *Service) Type() ontology.Type { return OntologyType }

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

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(context.Context, iter.Seq[ontology.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, Policy]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return gorp.Observe[uuid.UUID, Policy](s.cfg.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter(ctx context.Context) (iter.Seq[ontology.Resource], io.Closer, error) {
	n, closer, err := gorp.WrapReader[uuid.UUID, Policy](s.cfg.DB).OpenNexter(ctx)
	if err != nil {
		return nil, nil, err
	}
	return xiter.Map(n, newResource), closer, nil
}
