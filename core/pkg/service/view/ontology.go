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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/core"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

const OntologyType ontology.Type = "view"

// OntologyID returns the unique ID to identify the view within the Synnax ontology.
func OntologyID(k uuid.UUID) ontology.ID {
	return ontology.ID{Type: OntologyType, Key: k.String()}
}

// OntologyIDs converts a slice of keys to a slice of ontology IDs.
func OntologyIDs(keys []uuid.UUID) (ids []ontology.ID) {
	return lo.Map(keys, func(k uuid.UUID, _ int) ontology.ID {
		return OntologyID(k)
	})
}

func KeyFromOntologyID(id ontology.ID) string {
	return id.Key
}

// KeysFromOntologyIDs extracts the keys of the logs from the ontology IDs.
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

// OntologyIDsFromViews converts a slice of views to a slice of ontology IDs.
func OntologyIDsFromViews(views []View) (ids []ontology.ID) {
	return lo.Map(views, func(s View, _ int) ontology.ID {
		return OntologyID(s.Key)
	})
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":  zyn.UUID(),
	"name": zyn.String(),
})

func newResource(s View) ontology.Resource {
	return core.NewResource(schema, OntologyID(s.Key), s.Name, s)
}

var _ ontology.Service = (*Service)(nil)

type change = changex.Change[uuid.UUID, View]

func (s *Service) Type() ontology.Type { return OntologyType }

// Schema implements ontology.Service.
func (s *Service) Schema() zyn.Schema { return schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (ontology.Resource, error) {
	k := uuid.MustParse(key)
	var st View
	err := s.NewRetrieve().WhereKeys(k).Entry(&st).Exec(ctx, tx)
	return newResource(st), err
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
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, View]) {
		f(ctx, iter.NexterTranslator[change, ontology.Change]{Wrap: reader, Translate: translateChange})
	}
	return gorp.Observe[uuid.UUID, View](s.cfg.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter() (iter.NexterCloser[ontology.Resource], error) {
	n, err := gorp.WrapReader[uuid.UUID, View](s.cfg.DB).OpenNexter()
	return iter.NexterCloserTranslator[View, ontology.Resource]{
		Wrap:      n,
		Translate: newResource,
	}, err
}
