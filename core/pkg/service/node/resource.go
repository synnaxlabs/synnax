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
	"maps"
	"slices"
	"strconv"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/validate"
)

// Resources returns the resource.Provider of the nodes that the cluster holds when each
// of its methods is called. The free node always exists.
func (s *Service) Resources() resource.Provider { return provider{svc: s} }

type provider struct{ svc *Service }

// Search implements resource.Provider.
func (p provider) Search(
	_ context.Context,
	_ gorp.Tx,
	q search.Query[resource.ID],
) (search.Result[resource.ID], error) {
	nodes := p.svc.cfg.Cluster.CopyState().Nodes
	idx := search.New[Key]()
	for _, key := range slices.Sorted(maps.Keys(nodes)) {
		idx.Set(key, name(key))
	}
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
	nodes := p.svc.cfg.Cluster.CopyState().Nodes
	resources := make([]resource.Resource, 0, len(ids))
	for _, id := range ids {
		k, err := strconv.ParseUint(id.Key, 10, 16)
		if err != nil {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"invalid node key %q",
				id.Key,
			)
		}
		key := Key(k)
		if _, ok := nodes[key]; ok || key.IsFree() {
			resources = append(
				resources,
				resource.Resource{ID: ResourceID(key), Name: name(key)},
			)
		}
	}
	return resources, nil
}

// List implements resource.Provider. It lists the nodes in order of key.
func (p provider) List(
	_ context.Context,
	_ gorp.Tx,
	limit int,
	except func(resource.ID) bool,
) ([]resource.Resource, error) {
	keys := slices.Sorted(maps.Keys(p.svc.cfg.Cluster.CopyState().Nodes))
	resources := make([]resource.Resource, 0, min(limit, len(keys)))
	for _, key := range keys {
		if len(resources) == limit {
			break
		}
		if id := ResourceID(key); except == nil || !except(id) {
			resources = append(resources, resource.Resource{ID: id, Name: name(key)})
		}
	}
	return resources, nil
}

// ResourceID returns the resource ID of the node with key.
func ResourceID(key Key) resource.ID {
	return resource.ID{Type: resource.TypeNode, Key: strconv.Itoa(int(key))}
}

func name(key Key) string { return fmt.Sprintf("Node %v", key) }
