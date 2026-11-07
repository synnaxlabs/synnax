// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package node

import (
	"context"
	"fmt"
	"strconv"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/validate"
)

// Resources returns the resource.Provider of the host node. The free node always exists.
func (s *Service) Resources() resource.Provider { return provider{svc: s} }

type provider struct{ svc *Service }

// Search implements resource.Provider.
func (p provider) Search(
	_ context.Context,
	_ gorp.Tx,
	q search.Query[resource.ID],
) (search.Result[resource.ID], error) {
	idx := search.New[Key]()
	idx.Set(p.svc.cfg.Host.Key, name(p.svc.cfg.Host.Key))
	res := idx.Search(search.Query[Key]{
		Term:   q.Term,
		Except: func(key Key) bool { return q.Except != nil && q.Except(ResourceID(key)) },
		Limit:  q.Limit,
	})
	return search.MapKeys(res, ResourceID), nil
}

// Retrieve implements resource.Provider.
func (p provider) Retrieve(
	_ context.Context,
	_ gorp.Tx,
	ids []resource.ID,
) ([]resource.Resource, error) {
	resources := make([]resource.Resource, 0, len(ids))
	for _, id := range ids {
		k, err := strconv.ParseUint(id.Key, 10, 16)
		if err != nil || ResourceID(Key(k)) != id {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"invalid node key %q",
				id.Key,
			)
		}
		key := Key(k)
		if key == p.svc.cfg.Host.Key || key.Free() {
			resources = append(
				resources,
				resource.Resource{ID: ResourceID(key), Name: name(key)},
			)
		}
	}
	return resources, nil
}

// List implements resource.Provider. It lists the host node.
func (p provider) List(
	_ context.Context,
	_ gorp.Tx,
	limit int,
	except func(resource.ID) bool,
) ([]resource.Resource, error) {
	key := p.svc.cfg.Host.Key
	id := ResourceID(key)
	if limit == 0 || (except != nil && except(id)) {
		return []resource.Resource{}, nil
	}
	return []resource.Resource{{ID: id, Name: name(key)}}, nil
}

// ResourceID returns the resource ID of the node with key.
func ResourceID(key Key) resource.ID {
	return resource.ID{Type: resource.TypeNode, Key: strconv.Itoa(int(key))}
}

func name(key Key) string { return fmt.Sprintf("Node %v", key) }
