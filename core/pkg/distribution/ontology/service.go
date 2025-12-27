// Copyright 2025 Synnax Labs, Inc.
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
	"io"
	"iter"

	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
	"go.uber.org/zap"
)

// Service represents a service that exposes a set of entities to the ontology (such as
// a channel, node, user, etc.). Because the ontology only stores the relationships
// between entities, it is a service's responsibility to provide the entities themselves
// when the ontology requests them.
type Service interface {
	Type() Type
	// Schema returns the schema of the entities returned by this service.
	Schema() zyn.Schema
	// RetrieveResource returns the resource with the give key (Name.Name). If the resource
	// does not exist, returns a query.NotFound error.
	RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (Resource, error)
	// Observable is used by the ontology to subscribe to changes in the entities.
	// This functionality is primarily used for search indexing. If the service's entities
	// are static, use observe.Noop.
	observe.Observable[iter.Seq[Change]]
	// OpenNexter opens a Nexter type iterator that allows the caller to iterate over
	// all resources held by the Service.
	OpenNexter(context.Context) (iter.Seq[Resource], io.Closer, error)
}

type serviceRegistrar map[Type]Service

func (s serviceRegistrar) register(svc Service) {
	t := svc.Type()
	if _, ok := s[t]; ok {
		zap.S().DPanic("service already registered", zap.Stringer("type", t))
		return
	}
	s[t] = svc
}

func (s serviceRegistrar) retrieveResource(ctx context.Context, id ID, tx gorp.Tx) (Resource, error) {
	svc, ok := s[id.Type]
	if !ok {
		panic(fmt.Sprintf("[ontology] - service %s not found", id.Type))
	}
	return svc.RetrieveResource(ctx, id.Key, tx)
}
