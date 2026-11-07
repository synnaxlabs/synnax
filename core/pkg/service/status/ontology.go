// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"context"
	"iter"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	xchange "github.com/synnaxlabs/x/change"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/zyn"
)

// ResourceID returns the unique ID to identify the status within the Synnax ontology.
func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypeStatus, Key: k}
}

// ResourceIDs converts a slice of keys to a slice of resource IDs.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(k Key, _ int) resource.ID { return ResourceID(k) })
}

// KeysFromResourceIDs converts a slice of resource IDs to a slice of keys.
func KeysFromResourceIDs(ids []resource.ID) []Key {
	return lo.Map(ids, func(id resource.ID, _ int) Key { return id.Key })
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":         zyn.String(),
	"name":        zyn.String(),
	"variant":     zyn.String(),
	"message":     zyn.String(),
	"description": zyn.String(),
	"time":        zyn.Int64().Coerce(),
})

func newResource(s Status[any]) ontology.Resource {
	return ontology.NewResource(schema, s.ResourceID(), s.Name, s)
}

var _ ontology.Service = (*Service)(nil)

type change = xchange.Change[Key, Status[any]]

func (s *Service) Type() resource.Type { return resource.TypeStatus }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(
	ctx context.Context,
	key Key,
	tx gorp.Tx,
) (ontology.Resource, error) {
	var st Status[any]
	if err := s.NewRetrieve[any]().
		Where(MatchKeys[any](key)).
		Entry(&st).
		Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(st), nil
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
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Status[any]]) {
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
