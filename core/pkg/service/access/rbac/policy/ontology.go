// Copyright 2025 Synnax Labs, Inc.
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

const OntologyType ontology.Type = "policy"

// OntologyID constructs a unique ontology.ID for the Policy with the given key.
func OntologyID(k uuid.UUID) ontology.ID {
	return ontology.ID{Type: OntologyType, Key: k.String()}
}

// OntologyIDs constructs a slice of unique ontology.ResourceIDs for the Policys with the given
// keys.
func OntologyIDs(keys []uuid.UUID) []ontology.ID {
	return lo.Map(keys, func(k uuid.UUID, _ int) ontology.ID { return OntologyID(k) })
}

// OntologyIDsFromPolicies constructs a slice of unique ontology.ResourceIDs for the given Policys.
func OntologyIDsFromPolicies(policies []Policy) []ontology.ID {
	return lo.Map(policies, func(l Policy, _ int) ontology.ID { return OntologyID(l.Key) })
}

// KeysFromOntologyIds extracts the Policy keys from the given ontology.ResourceIDs.
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

func newResource(l Policy) ontology.Resource {
	return core.NewResource(schema, OntologyID(l.Key), l.Name, l)
}

type change = changex.Change[uuid.UUID, Policy]

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
	var l Policy
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
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, Policy]) {
		f(ctx, iter.NexterTranslator[change, ontology.Change]{
			Wrap:      reader,
			Translate: translateChange,
		})
	}
	return gorp.Observe[uuid.UUID, Policy](s.cfg.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter() (iter.NexterCloser[ontology.Resource], error) {
	n, err := gorp.WrapReader[uuid.UUID, Policy](s.cfg.DB).OpenNexter()
	return iter.NexterCloserTranslator[Policy, ontology.Resource]{
		Wrap:      n,
		Translate: newResource,
	}, err
}
