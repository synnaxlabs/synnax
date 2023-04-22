// Copyright 2023 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
)

// Service represents a service that exposes a set of entities to the ontology (such
// as a channel, node, user, etc.). Because the ontology only stores the relationships
// between entities, it is a service's responsibility to provide the entities themselves
// when the ontology requests them.
type Service interface {
	// Schema returns the schema of the entities returned by this service.
	Schema() *Schema
	// RetrieveResource returns the resource with the give key (ID.Key). If the resource
	// does not exist, returns a query.NotFound error.
	RetrieveResource(ctx context.Context, key string) (Resource, error)
	// Observable is used by the ontology to subscribe to changes in the entities.
	// This functionality is primarily used for search indexing. If the service's entities
	// are static, use observe.Noop.
	observe.Observable[iter.Next[schema.Change]]
	OpenNext() iter.NextCloser[Resource]
}

type serviceRegistrar map[Type]Service

func (s serviceRegistrar) register(svc Service) {
	t := svc.Schema().Type
	if _, ok := s[t]; ok {
		panic("[ontology] - service already registered")
	}
	s[t] = svc
}

func (s serviceRegistrar) retrieveResource(ctx context.Context, id ID) (Resource, error) {
	svc, ok := s[id.Type]
	if !ok {
		panic("[ontology] - service not found")
	}
	return svc.RetrieveResource(ctx, id.Key)
}
