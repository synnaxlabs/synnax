// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package effect

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

const ontologyType ontology.Type = "effect"

// OntologyID returns unique identifier for the effect within the ontology.
func OntologyID(k uuid.UUID) ontology.ID {
	return ontology.ID{Type: ontologyType, Key: k.String()}
}

// OntologyIDs returns unique identifiers for the effects within the ontology.
func OntologyIDs(keys []uuid.UUID) []ontology.ID {
	return lo.Map(keys, func(key uuid.UUID, _ int) ontology.ID {
		return OntologyID(key)
	})
}

// KeysFromOntologyIDs extracts the keys of the effects from the ontology IDs.
func KeysFromOntologyIDs(ids []ontology.ID) (keys []uuid.UUID, err error) {
	keys = make([]uuid.UUID, len(ids))
	for i, id := range ids {
		keys[i], err = uuid.Parse(id.Key)
		if err != nil {
			return nil, err
		}
	}
	return keys, nil
}

// OntologyIDsFromEffects returns the ontology IDs of the effects.
func OntologyIDsFromEffects(effects []Effect) []ontology.ID {
	return lo.Map(effects, func(c Effect, _ int) ontology.ID {
		return OntologyID(c.Key)
	})
}

var _schema = ontology.NewSchema(

	ontologyType,
	map[string]zyn.Z{
		"key":  zyn.String(),
		"type": zyn.String(),
	},
)

func newResource(c Effect) core.Resource {
	return core.NewResource(_schema, OntologyID(c.Key), c.Name, c)
}

var _ ontology.Service = (*Service)(nil)

type change = changex.Change[uuid.UUID, Effect]

// Schema implements ontology.Service.
func (s *Service) Schema() *core.Schema { return _schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (core.Resource, error) {
	k := uuid.MustParse(key)
	var effect Effect
	err := s.NewRetrieve().WhereKeys(k).Entry(&effect).Exec(ctx, tx)
	return newResource(effect), err
}

func translateChange(c change) core.Change {
	return core.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(ctx context.Context, nexter iter.Nexter[core.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, Effect]) {
		f(ctx, iter.NexterTranslator[change, core.Change]{Wrap: reader, Translate: translateChange})
	}
	return gorp.Observe[uuid.UUID, Effect](s.cfg.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter() (iter.NexterCloser[core.Resource], error) {
	n, err := gorp.WrapReader[uuid.UUID, Effect](s.cfg.DB).OpenNexter()
	return iter.NexterCloserTranslator[Effect, core.Resource]{
		Wrap:      n,
		Translate: newResource,
	}, err
}
