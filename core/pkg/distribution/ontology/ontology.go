// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package ontology provides a service for mapping relationships between different
// resources within a Synnax cluster. It implements a directed acyclic graph (DAG) that
// allows for the dynamic definition of complex relationship hierarchies. The primary
// objective is to separate the core algorithms operating on the data structures of a
// particular service (frame iteration, channel command streaming, user management,
// etc.) from the relationships those data structures may have with other resources in
// the cluster.
//
// This enables powerful patterns that allows us to define relationships without needing
// to modify the core algorithms or data structures of a particular service, enabling a
// more flexible and extensible architecture.
//
// It also serves our users by allowing them to query the topology of the cluster by
// traversing relationships (such as a browsable tree).
//
// For more information, see the [ontology RFC]:
// https://docs.synnaxlabs.com/rfc/5-220716-ontology.
package ontology

import (
	"context"
	"iter"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Ontology exposes an ontology stored in a key-value database for reading and writing.
type Ontology struct {
	Config
	ResourceObserver     observe.Observer[iter.Seq[Change]]
	RelationshipObserver observe.Observable[gorp.TxReader[[]byte, Relationship]]
	registrar            serviceRegistrar
	disconnectObservers  []observe.Disconnect
	resourceTable        *gorp.Table[string, Resource]
	relationshipTable    *gorp.Table[[]byte, Relationship]
}

type Config struct {
	DB *gorp.DB
	alamos.Instrumentation
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("ontology")
	validate.NotNil(v, "db", c.DB)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Open opens the ontology using the given configuration. If the RootID resource does
// not exist, it will be created.
func Open(ctx context.Context, configs ...Config) (*Ontology, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	resourceTable, err := gorp.OpenTable[string, Resource](ctx, cfg.DB)
	if err != nil {
		return nil, err
	}
	relationshipTable, err := gorp.OpenTable[[]byte, Relationship](ctx, cfg.DB)
	if err != nil {
		return nil, err
	}
	o := &Ontology{
		Config:               cfg,
		ResourceObserver:     observe.New[iter.Seq[Change]](),
		RelationshipObserver: gorp.Observe[[]byte, Relationship](cfg.DB),
		registrar:            serviceRegistrar{ResourceTypeBuiltin: &builtinService{}},
		resourceTable:        resourceTable,
		relationshipTable:    relationshipTable,
	}

	if err = o.NewRetrieve().WhereIDs(RootID).Exec(ctx, cfg.DB); errors.Is(err, query.ErrNotFound) {
		err = o.NewWriter(cfg.DB).DefineResource(ctx, RootID)
	}
	if err != nil {
		return nil, err
	}

	return o, nil
}

// Writer defines and deletes resources within the ontology.
type Writer interface {
	// DefineResource defines a new resource with the given ID. If the resource already
	// exists, DefineResource does nothing.
	DefineResource(context.Context, ID) error
	// HasResource returns true if the resource with the given ID exists.
	HasResource(context.Context, ID) (bool, error)
	// DefineManyResources defines multiple resources with the given IDs. If any of the
	// resources already exist, DefineManyResources does nothing.
	DefineManyResources(context.Context, []ID) error
	// DeleteResource deletes the resource with the given ID along with all of its
	// incoming and outgoing relationships.  If the resource does not exist,
	// DeleteResource does nothing.
	DeleteResource(context.Context, ID) error
	// DeleteManyResources deletes multiple resources with the given IDs along with all
	// of their incoming and outgoing relationships. If any of the resources do not
	// exist, DeleteManyResources does nothing.
	DeleteManyResources(context.Context, []ID) error
	HasRelationship(ctx context.Context, from ID, t RelationshipType, to ID) (bool, error)
	// DefineRelationship defines a directional relationship of type t between the
	// resources with the given keys. If the relationship already exists,
	// DefineRelationship does nothing.
	DefineRelationship(ctx context.Context, from ID, t RelationshipType, to ID) error
	// DefineFromOneToManyRelationships defines a directional relationship of type t
	// from the resource with the given ID to multiple resources. If any of the
	// relationships already exist, DefineFromOneToManyRelationships does nothing.
	DefineFromOneToManyRelationships(ctx context.Context, from ID, t RelationshipType, to []ID) error
	// DeleteRelationship deletes the relationship with the given keys and type. If the
	// relationship does not exist, DeleteRelationship does nothing.
	DeleteRelationship(ctx context.Context, from ID, t RelationshipType, to ID) error
	// DeleteOutgoingRelationshipsOfType deletes all outgoing relationships of the given
	// types from the resource with the given ID. If the resource does not exist, or if
	// it has no outgoing relationships of the given types,
	// DeleteOutgoingRelationshipsOfTypes does nothing.
	DeleteOutgoingRelationshipsOfType(ctx context.Context, from ID, relationshipType RelationshipType) error
	// DeleteIncomingRelationshipsOfType deletes all incoming relationships of the given
	// types to the resource with the given ID. If the resource does not exist, or if it
	// has no incoming relationships of the given types,
	// DeleteIncomingRelationshipsOfTypes does nothing.
	DeleteIncomingRelationshipsOfType(ctx context.Context, to ID, relationshipType RelationshipType) error
	// NewRetrieve opens a new Retrieve query that provides a view of pending operations
	// merged with the underlying database. If the Writer is executing directly against
	// the underlying database, the Retrieve query behaves exactly as if calling
	// Ontology.NewRetrieve.
	NewRetrieve() Retrieve
}

// NewWriter opens a new Writer using the provided transaction. Panics if the
// transaction does not root from the same database as the Ontology.
func (o *Ontology) NewWriter(tx gorp.Tx) Writer {
	return dagWriter{tx: o.DB.OverrideTx(tx), registrar: o.registrar}
}

// RegisterService registers a Service for a particular [Type] with the [Ontology].
// Ontology will execute queries for Entity information for the given Type using the
// provided Service. RegisterService panics if a Service is already registered for the
// given Type.
func (o *Ontology) RegisterService(svc Service) {
	o.L.Debug("registering service", zap.Stringer("type", svc.Type()))
	o.registrar.register(svc)
	o.disconnectObservers = append(o.disconnectObservers, svc.OnChange(o.ResourceObserver.Notify))
}

func (o *Ontology) Close() error {
	for _, d := range o.disconnectObservers {
		d()
	}
	return errors.Join(o.resourceTable.Close(), o.relationshipTable.Close())
}
