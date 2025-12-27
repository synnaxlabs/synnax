// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot

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

const OntologyType ontology.Type = "lineplot"

// OntologyID returns unique identifier for the lineplot within the ontology.
func OntologyID(k uuid.UUID) ontology.ID {
	return ontology.ID{Type: OntologyType, Key: k.String()}
}

// OntologyIDs returns unique identifiers for the schematics within the ontology.
func OntologyIDs(keys []uuid.UUID) []ontology.ID {
	return lo.Map(keys, func(k uuid.UUID, _ int) ontology.ID { return OntologyID(k) })
}

// OntologyIDsFromLinePlots returns the ontology IDs of the schematics.
func OntologyIDsFromLinePlots(linePlots []LinePlot) []ontology.ID {
	return lo.Map(linePlots, func(linePlot LinePlot, _ int) ontology.ID {
		return OntologyID(linePlot.Key)
	})
}

var schema = zyn.Object(map[string]zyn.Schema{"key": zyn.UUID(), "name": zyn.String()})

func newResource(linePlot LinePlot) ontology.Resource {
	return ontology.NewResource(
		schema,
		OntologyID(linePlot.Key),
		linePlot.Name,
		linePlot,
	)
}

var _ ontology.Service = (*Service)(nil)

type change = xchange.Change[uuid.UUID, LinePlot]

func (s *Service) Type() ontology.Type { return OntologyType }

// Schema implements ontology.Service.
func (s *Service) Schema() zyn.Schema { return schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (ontology.Resource, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var linePlot LinePlot
	if err = s.NewRetrieve().WhereKeys(k).Entry(&linePlot).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(linePlot), nil
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
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, LinePlot]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return gorp.Observe[uuid.UUID, LinePlot](s.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter(ctx context.Context) (iter.Seq[ontology.Resource], io.Closer, error) {
	n, closer, err := gorp.WrapReader[uuid.UUID, LinePlot](s.DB).OpenNexter(ctx)
	if err != nil {
		return nil, nil, err
	}
	return xiter.Map(n, newResource), closer, nil
}
