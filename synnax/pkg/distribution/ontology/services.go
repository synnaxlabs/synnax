// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology

// Service represents a service that exposes a set of entities to the ontology (such
// as a channel, node, user, etc.). Because the ontology only stores the relationships
// between entities, it is a service's responsibility to provide the entities themselves
// when the ontology requests them.
type Service interface {
	// Schema returns the schema of the entities returned by this service.
	Schema() *Schema
	// RetrieveEntity returns the entity with the give key (ID.Key). If the entity
	// does not exist, a query.NotFound error should be returned.
	RetrieveEntity(key string) (Entity, error)
}

type serviceRegistrar map[Type]Service

func (s serviceRegistrar) register(svc Service) {
	t := svc.Schema().Type
	if _, ok := s[t]; ok {
		panic("[ontology] - service already registered")
	}
	s[t] = svc
}

func (s serviceRegistrar) retrieveEntity(id ID) (Entity, error) {
	svc, ok := s[id.Type]
	if !ok {
		panic("[ontology] - service not found")
	}
	return svc.RetrieveEntity(id.Key)
}
