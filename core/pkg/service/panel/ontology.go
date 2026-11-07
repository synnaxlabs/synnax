// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel

import (
	"context"
	"iter"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/color"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/zyn"
)

func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypePanel, Key: k.String()}
}

func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(k Key, _ int) resource.ID { return ResourceID(k) })
}

func ResourceIDsFromPanels(panels []Panel) []resource.ID {
	return lo.Map(panels, func(p Panel, _ int) resource.ID { return p.ResourceID() })
}

func KeysFromResourceIDs(ids []resource.ID) ([]Key, error) {
	keys := make([]Key, len(ids))
	var err error
	for i, id := range ids {
		if keys[i], err = uuid.Parse(id.Key); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":   zyn.UUID(),
	"name":  zyn.String(),
	"icon":  zyn.String().Optional(),
	"color": color.Schema.Optional(),
})

func newResource(p Panel) ontology.Resource {
	return ontology.NewResource(schema, p.ResourceID(), p.Name, p)
}

type change = xchange.Change[Key, Panel]

var _ ontology.Service = (*Service)(nil)

func (s *Service) Type() resource.Type { return resource.TypePanel }

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
	var p Panel
	if err = s.NewRetrieve().Where(MatchKeys(k)).Entry(&p).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(p), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     ResourceID(c.Key).String(),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Panel]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return s.table.Observe().OnChange(handleChange)
}

// Search implements ontology.Service.
func (s *Service) Search(
	ctx context.Context,
	tx gorp.Tx,
	q search.Query[resource.ID],
) (search.Result[resource.ID], error) {
	return ontology.SearchIndex(ctx, tx, s.indexes.search, ResourceID, q)
}
