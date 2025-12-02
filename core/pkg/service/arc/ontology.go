// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"io"
	"iter"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/core"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

const ontologyType ontology.Type = "arc"

// OntologyID returns unique identifier for the Arc within the ontology.
func OntologyID(k uuid.UUID) ontology.ID {
	return ontology.ID{Type: ontologyType, Key: k.String()}
}

// OntologyIDs returns unique identifiers for the arcs within the ontology.
func OntologyIDs(keys []uuid.UUID) []ontology.ID {
	return lo.Map(keys, func(key uuid.UUID, _ int) ontology.ID {
		return OntologyID(key)
	})
}

// KeysFromOntologyIDs extracts the keys of the arcs from the ontology IDs.
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

// OntologyIDsFromArcs returns the ontology IDs of the arcs.
func OntologyIDsFromArcs(arcs []Arc) []ontology.ID {
	return lo.Map(arcs, func(c Arc, _ int) ontology.ID {
		return OntologyID(c.Key)
	})
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key": zyn.UUID(),
})

func newResource(c Arc) core.Resource {
	return core.NewResource(schema, OntologyID(c.Key), c.Name, c)
}

var _ ontology.Service = (*Service)(nil)

type change = changex.Change[uuid.UUID, Arc]

func (s *Service) Type() ontology.Type { return ontologyType }

// Schema implements ontology.Service.
func (s *Service) Schema() zyn.Schema { return schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (core.Resource, error) {
	k := uuid.MustParse(key)
	var arc Arc
	err := s.NewRetrieve().WhereKeys(k).Entry(&arc).Exec(ctx, tx)
	return newResource(arc), err
}

func translateChange(c change) core.Change {
	return core.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(context.Context, iter.Seq[core.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, Arc]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return gorp.Observe[uuid.UUID, Arc](s.cfg.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter(ctx context.Context) (iter.Seq[core.Resource], io.Closer, error) {
	n, closer, err := gorp.WrapReader[uuid.UUID, Arc](s.cfg.DB).OpenNexter(ctx)
	return xiter.Map(n, newResource), closer, err
}
