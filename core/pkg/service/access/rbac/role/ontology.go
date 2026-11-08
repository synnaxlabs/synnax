// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role

import (
	"context"
	"iter"
	"uuid"

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

// ResourceID constructs a unique resource.ID for the Role with the given key.
func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypeRole, Key: k.String()}
}

// ResourceIDsFromRoles constructs a slice of unique resource.IDs for the given Roles.
func ResourceIDsFromRoles(roles []Role) []resource.ID {
	return lo.Map(roles, func(r Role, _ int) resource.ID { return r.ResourceID() })
}

// KeyFromResourceID parses the role key from the given resource.ID.
func KeyFromResourceID(id resource.ID) (Key, error) { return uuid.Parse(id.Key) }

// KeysFromResourceIDs converts a slice of resource IDs to a slice of keys, returning an
// error if any of the IDs are invalid.
func KeysFromResourceIDs(ids []resource.ID) ([]Key, error) {
	return lo.MapErr(ids, func(id resource.ID, _ int) (Key, error) {
		return KeyFromResourceID(id)
	})
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":      zyn.UUID(),
	"name":     zyn.String(),
	"internal": zyn.Bool(),
})

func newResource(r Role) ontology.Resource {
	return ontology.NewResource(schema, r.ResourceID(), r.Name, r)
}

type change = xchange.Change[Key, Role]

func (s *Service) Type() resource.Type { return resource.TypeRole }

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
	var r Role
	if err := s.NewRetrieve().Where(MatchKeys(k)).Entry(&r).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(r), nil
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
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Role]) {
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
