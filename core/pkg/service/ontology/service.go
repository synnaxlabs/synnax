// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology

import (
	"context"
	"fmt"
	"iter"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/search"
	"go.uber.org/zap"
)

// Service represents a service that exposes a set of entities to the ontology (such as
// a channel, node, user, etc.). Because the ontology only stores the relationships
// between entities, it is a service's responsibility to provide the entities themselves
// when the ontology requests them.
type Service interface {
	// Type returns the [ResourceType] this service is responsible for.
	Type() ResourceType
	// RetrieveResource returns the resource with the given key. If the resource does
	// not exist, returns [query.ErrNotFound].
	RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (Resource, error)
	// Search returns the IDs of at most limit resources whose text matches term,
	// closest match first. Search reads the changes staged against tx.
	Search(
		ctx context.Context,
		tx gorp.Tx,
		term string,
		limit int,
	) (search.Result[ID], error)
	// Observable is used by the [Ontology] to subscribe to changes in the entities.
	// This is used to propagate resource changes for signals (see
	// [Ontology.ObserveResources]). If the service's entities are static, use
	// [observe.Noop].
	observe.Observable[iter.Seq[Change]]
}

// SearchIndex returns the IDs of at most limit entries of idx whose text matches term,
// closest match first. id returns the ID of the entry with a key. Services use it to
// implement Service.Search.
func SearchIndex[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	tx gorp.Tx,
	idx *gorp.SearchIndex[K, E],
	term string,
	limit int,
	id func(K) ID,
) (search.Result[ID], error) {
	res, err := idx.Search(ctx, tx, term, limit)
	if err != nil {
		return search.Result[ID]{}, err
	}
	return search.MapKeys(res, id), nil
}

type serviceRegistrar map[ResourceType]Service

// register returns false, and keeps the registered Service, when a Service is already
// registered for the type of svc.
func (s serviceRegistrar) register(svc Service) bool {
	t := svc.Type()
	if _, ok := s[t]; ok {
		zap.S().DPanic("service already registered", zap.Stringer("type", t))
		return false
	}
	s[t] = svc
	return true
}

func (s serviceRegistrar) retrieveResource(
	ctx context.Context,
	id ID,
	tx gorp.Tx,
) (Resource, error) {
	svc, ok := s[id.Type]
	if !ok {
		panic(fmt.Sprintf("[ontology] - service %s not found", id.Type))
	}
	return svc.RetrieveResource(ctx, id.Key, tx)
}
