// Copyright 2026 Synnax Labs, Inc.
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

// OntologyID returns a unique ID for the view with the given key within the Synnax
// ontology.
func OntologyID(keys Key) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeView, Key: keys.String()}
}

// OntologyIDs returns the ontology IDs for the given keys.
func OntologyIDs(keys []Key) []ontology.ID {
	return lo.Map(keys, func(k Key, _ int) ontology.ID { return OntologyID(k) })
}

// KeysFromOntologyIDs returns the keys of the views for the given ontology IDs.
func KeysFromOntologyIDs(ids []ontology.ID) ([]Key, error) {
	return lo.MapErr(ids, func(id ontology.ID, _ int) (Key, error) {
		return uuid.Parse(id.Key)
	})
}

// OntologyIDsFromViews converts a slice of views to a slice of ontology IDs.
func OntologyIDsFromViews(views []View) []ontology.ID {
	return lo.Map(views, func(v View, _ int) ontology.ID { return v.OntologyID() })
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":  zyn.UUID(),
	"name": zyn.String(),
	"type": zyn.String(),
})

func newResource(v View) ontology.Resource {
	return ontology.NewResource(schema, v.OntologyID(), v.Name, v)
}

var (
	_ ontology.Service = (*Service)(nil)
	_ search.Service   = (*Service)(nil)
)

type change = xchange.Change[Key, View]

func (*Service) Type() ontology.ResourceType { return ontology.ResourceTypeView }

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
	if err = s.NewRetrieve().Where(MatchKeys(k)).Entry(&v).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(v), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key).String(),
		Value:   newResource(c.Value),
	}
}

func (s *Service) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, View]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return s.table.Observe().OnChange(handleChange)
}

func (s *Service) OpenNexter(
	ctx context.Context,
) (iter.Seq[ontology.Resource], io.Closer, error) {
	n, closer, err := s.table.OpenNexter(ctx)
	if err != nil {
		return nil, nil, err
	}
	return xiter.Map(n, newResource), closer, nil
}
