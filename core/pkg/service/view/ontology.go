// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package view

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

const OntologyType ontology.Type = "view"

// OntologyID returns a unique ID for the view with the given key within the Synnax
// ontology.
func OntologyID(keys uuid.UUID) ontology.ID {
	return ontology.ID{Type: OntologyType, Key: keys.String()}
}

// OntologyIDs returns the ontology IDs for the given keys.
func OntologyIDs(keys []uuid.UUID) []ontology.ID {
	return lo.Map(keys, func(k uuid.UUID, _ int) ontology.ID { return OntologyID(k) })
}

// KeysFromOntologyIDs returns the keys of the views for the given ontology IDs.
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

// OntologyIDsFromViews converts a slice of views to a slice of ontology IDs.
func OntologyIDsFromViews(views []View) []ontology.ID {
	return lo.Map(views, func(v View, _ int) ontology.ID { return OntologyID(v.Key) })
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":  zyn.UUID(),
	"name": zyn.String(),
	"type": zyn.String(),
})

func newResource(v View) ontology.Resource {
	return ontology.NewResource(schema, OntologyID(v.Key), v.Name, v)
}

var _ ontology.Service = (*Service)(nil)

type change = xchange.Change[uuid.UUID, View]

func (s *Service) Type() ontology.Type { return OntologyType }

func (s *Service) Schema() zyn.Schema { return schema }

func (s *Service) RetrieveResource(
	ctx context.Context,
	key string,
	tx gorp.Tx,
) (ontology.Resource, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var v View
	if err = s.NewRetrieve().WhereKeys(k).Entry(&v).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(v), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key),
		Value:   newResource(c.Value),
	}
}

func (s *Service) OnChange(
	f func(context.Context, iter.Nexter[ontology.Change]),
) observe.Disconnect {
	handleChange := func(
		ctx context.Context,
		reader gorp.TxReader[uuid.UUID, View],
	) {
		f(ctx, iter.NexterTranslator[change, ontology.Change]{
			Wrap:      reader,
			Translate: translateChange,
		})
	}
	return gorp.Observe[uuid.UUID, View](s.cfg.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter() (iter.NexterCloser[ontology.Resource], error) {
	n, err := gorp.WrapReader[uuid.UUID, View](s.cfg.DB).OpenNexter()
	if err != nil {
		return nil, err
	}
	return iter.NexterCloserTranslator[View, ontology.Resource]{
		Wrap:      n,
		Translate: newResource,
	}, nil
}
