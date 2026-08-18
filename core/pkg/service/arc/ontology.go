// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

// OntologyID returns unique identifier for the Arc within the ontology.
func OntologyID(k Key) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeArc, Key: k.String()}
}

// OntologyIDs returns unique identifiers for the Arcs within the ontology.
func OntologyIDs(keys []Key) []ontology.ID {
	return lo.Map(keys, func(key Key, _ int) ontology.ID { return OntologyID(key) })
}

// KeysFromOntologyIDs extracts the keys of the arcs from the ontology IDs.
func KeysFromOntologyIDs(ids []ontology.ID) ([]Key, error) {
	return lo.MapErr(ids, func(id ontology.ID, _ int) (Key, error) {
		return uuid.Parse(id.Key)
	})
}

// OntologyIDsFromArcs returns the ontology IDs of the arcs.
func OntologyIDsFromArcs(arcs []Arc) []ontology.ID {
	return lo.Map(arcs, func(a Arc, _ int) ontology.ID { return a.OntologyID() })
}

var schema = zyn.Object(map[string]zyn.Schema{"key": zyn.UUID()})

func newResource(a Arc) ontology.Resource {
	return ontology.NewResource(schema, a.OntologyID(), a.Name, a)
}

var (
	_ ontology.Service = (*Service)(nil)
	_ search.Service   = (*Service)(nil)
)

type change = xchange.Change[Key, Arc]

func (s *Service) Type() ontology.ResourceType { return ontology.ResourceTypeArc }

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
	var arc Arc
	if err = s.NewRetrieve().Where(MatchKeys(k)).Entry(&arc).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(arc), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key).String(),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Arc]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return s.table.Observe().OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter(
	ctx context.Context,
) (iter.Seq[ontology.Resource], io.Closer, error) {
	n, closer, err := s.table.OpenNexter(ctx)
	if err != nil {
		return nil, nil, err
	}
	return xiter.Map(n, newResource), closer, nil
}
