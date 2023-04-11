// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package ontology provides a service for mapping relationships between different resources
// within a Synnax cluster. It implements a directed acyclic graph (DAG) that allows
// for the dynamic definition of complex relationship hierarchies. The primary objective
// is to separate the core algorithms operating on the data structures of a particular service
// (frame iteration, channel command streaming, user management, etc.) from the relationships
// those data structures may have with other resources in the cluster.
//
// This enables powerful patterns that allows us to define relationships without needing
// to modify the core algorithms or data structures of a particular service, enabling
// a more flexible and extensible architecture.
//
// It also serves our users by allowing them to query the topology of the cluster by
// traversing relationships (such as a browsable tree).
//
// For more information, see the [ontology RFC]: https://docs.synnaxlabs.com/rfc/5-220716-ontology.
package ontology

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

type (
	// Schema is a set of definitions that describe the structure of a resource.
	Schema = schema.Schema
	// Entity is the underlying data structure of a resource.
	Entity = schema.Entity
	// Type is a unique identifier for a particular class of resources (channel, user, etc.)
	Type = schema.Type
)

// Ontology exposes an ontology stored in a key-value database for reading and writing.
type Ontology struct {
	DB        *gorp.DB
	registrar serviceRegistrar
}

// Open opens the ontology stored in the given database. If the Root resource does not
// exist, it will be created.
func Open(ctx context.Context, db *gorp.DB) (*Ontology, error) {
	o := &Ontology{
		DB:        db,
		registrar: serviceRegistrar{BuiltIn: &builtinService{}},
	}
	err := o.NewRetrieve().WhereIDs(Root).Exec(ctx, db)
	if errors.Is(err, query.NotFound) {
		err = o.OpenWriter(db).DefineResource(ctx, Root)
	}
	return o, err
}

// Writer defines and deletes resources within the ontology.
type Writer interface {
	// DefineResource defines a new resource with the given ID. If the resource already
	// exists, DefineResource does nothing.
	DefineResource(ctx context.Context, id ID) error
	// DeleteResource deletes the resource with the given ID along with all of its
	// incoming and outgoing relationships.  If the resource does not exist,
	// DeleteResource does nothing.
	DeleteResource(ctx context.Context, id ID) error
	// DefineRelationship defines a directional relationship of type t between the
	// resources with the given IDs. If the relationship already exists, DefineRelationship
	// does nothing.
	DefineRelationship(ctx context.Context, from ID, t RelationshipType, to ID) error
	// DeleteRelationship deletes the relationship with the given IDs and type. If the
	// relationship does not exist, DeleteRelationship does nothing.
	DeleteRelationship(ctx context.Context, from ID, t RelationshipType, to ID) error
	// NewRetrieve opens a new Retrieve query that provides a view of pending
	// operations merged with the underlying database. If the Writer is executing directly
	// against the underlying database, the Retrieve query behaves exactly as if calling
	// Ontology.NewRetrieve.
	NewRetrieve() Retrieve
}

// OpenWriter opens a new Writer using the provided transaction.
// Panics if the transaction does not root from the same database as the Ontology.
func (o *Ontology) OpenWriter(tx gorp.Tx) Writer {
	return dagWriter{tx: o.DB.OverrideTx(tx)}
}

// RegisterService registers a Service for a particular [Type] with the [Ontology].
// Ontology will execute queries for Entity information for the given Type using the
// provided Service. RegisterService panics if a Service is already registered for
// the given Type.
func (o *Ontology) RegisterService(s Service) { o.registrar.register(s) }
