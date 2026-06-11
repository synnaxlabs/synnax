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
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Ontology exposes an ontology stored in a key-value database for reading and writing.
type Ontology struct {
	Config
	ResourceObserver     observe.Observer[iter.Seq[Change]]
	RelationshipObserver observe.Observable[gorp.TxReader[string, Relationship]]
	registrar            serviceRegistrar
	disconnectObservers  []observe.Disconnect
	closer               io.MultiCloser
	resourceTable        *gorp.Table[string, Resource]
	relationshipTable    *gorp.Table[string, Relationship]
	relIndexes           relationshipIndexes
}

// relationshipIndexes bundles the secondary indexes registered on the relationship
// table. relByTo answers "given an ID, which relationships point at it" in O(1), which
// is the lookup that ParentsTraverser would otherwise serve via a full pebble scan.
// relByFrom is the symmetric from-keyed index used by ChildrenTraverser; it duplicates
// work that the from-prefix scan already does cheaply, but having both directions in
// the index lets the traverse dispatcher pick the index uniformly without
// special-casing direction.
type relationshipIndexes struct {
	byTo   *gorp.LookupIndex[string, Relationship, ID]
	byFrom *gorp.LookupIndex[string, Relationship, ID]
}

func newRelationshipIndexes() relationshipIndexes {
	return relationshipIndexes{
		byTo: gorp.NewLookupIndex(
			"relationship_by_to",
			func(r *Relationship) ID { return r.To },
		),
		byFrom: gorp.NewLookupIndex(
			"relationship_by_from",
			func(r *Relationship) ID { return r.From },
		),
	}
}

func (i relationshipIndexes) all() []gorp.Index[string, Relationship] {
	return []gorp.Index[string, Relationship]{i.byTo, i.byFrom}
}

type Config struct {
	DB *gorp.DB
	alamos.Instrumentation
}

var _ config.Config[Config] = Config{}

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
func Open(ctx context.Context, configs ...Config) (o *Ontology, err error) {
	cfg, err := config.New(Config{}, configs...)
	if err != nil {
		return nil, err
	}
	o = &Ontology{
		Config:           cfg,
		ResourceObserver: observe.New[iter.Seq[Change]](),
		registrar:        serviceRegistrar{ResourceTypeBuiltin: &builtinService{}},
		relIndexes:       newRelationshipIndexes(),
	}
	cleanup, ok := service.NewOpener(ctx, &o.closer)
	defer func() { err = cleanup(err) }()
	if o.resourceTable, err = gorp.OpenTable(ctx, gorp.TableConfig[string, Resource]{
		DB:              cfg.DB,
		Instrumentation: cfg.Instrumentation,
		Migrations: []migrate.Migration{
			gorp.CodecMigration[string, Resource]("msgpack_to_orc"),
		},
	}); !ok(err, o.resourceTable) {
		return nil, err
	}
	if o.relationshipTable, err = gorp.OpenTable(ctx, gorp.TableConfig[string, Relationship]{
		DB:              cfg.DB,
		Instrumentation: cfg.Instrumentation,
		Indexes:         o.relIndexes.all(),
		Migrations: []migrate.Migration{
			gorp.CodecMigration[string, Relationship]("msgpack_to_orc"),
		},
	}); !ok(err, o.relationshipTable) {
		return nil, err
	}
	o.RelationshipObserver = o.relationshipTable.Observe()

	if err = o.NewRetrieve().WhereIDs(RootID).Exec(ctx, cfg.DB); errors.Is(err, query.ErrNotFound) {
		err = o.NewWriter(cfg.DB).DefineResource(ctx, RootID)
	}
	if !ok(err, nil) {
		return nil, err
	}

	return o, nil
}

// Writer defines and deletes resources within the ontology.
type Writer interface {
	// DefineResource defines one or more new resources with the given IDs. If any of
	// the resources already exist, DefineResource does nothing for those. Returns nil
	// if no IDs are provided.
	DefineResource(context.Context, ...ID) error
	// HasResource returns true if the resource with the given ID exists.
	HasResource(context.Context, ID) (bool, error)
	// DeleteResource deletes one or more resources with the given IDs along with all of
	// their incoming and outgoing relationships. If any of the resources do not exist,
	// DeleteResource does nothing for those. Returns nil if no IDs are provided.
	DeleteResource(context.Context, ...ID) error
	HasRelationship(ctx context.Context, from ID, t RelationshipType, to ID) (bool, error)
	// DefineRelationship defines a directional relationship of type t from the resource
	// with the given from ID to one or more to IDs. Already-existing relationships are
	// silently skipped. Returns graph.ErrCyclicDependency if any of the new
	// relationships would create a cycle (including the case where the
	// reverse-direction relationship already exists). Returns nil if no to IDs are
	// provided.
	DefineRelationship(ctx context.Context, from ID, t RelationshipType, to ...ID) error
	// DeleteRelationship deletes the relationship with the given keys and type. If the
	// relationship does not exist, DeleteRelationship does nothing.
	DeleteRelationship(ctx context.Context, from ID, t RelationshipType, to ID) error
	// DeleteOutgoingRelationshipsOfType deletes all outgoing relationships of the given
	// types from the resource with the given ID. If the resource does not exist, or if
	// it has no outgoing relationships of the given types,
	// DeleteOutgoingRelationshipsOfTypes does nothing.
	DeleteOutgoingRelationshipsOfType(context.Context, ID, RelationshipType) error
	// DeleteIncomingRelationshipsOfType deletes all incoming relationships of the given
	// types to the resource with the given ID. If the resource does not exist, or if it
	// has no incoming relationships of the given types,
	// DeleteIncomingRelationshipsOfTypes does nothing.
	DeleteIncomingRelationshipsOfType(context.Context, ID, RelationshipType) error
	// NewRetrieve opens a new Retrieve query that provides a view of pending operations
	// merged with the underlying database. If the Writer is executing directly against
	// the underlying database, the Retrieve query behaves exactly as if calling
	// Ontology.NewRetrieve.
	NewRetrieve() Retrieve
}

// NewWriter opens a new Writer using the provided transaction. Panics if the
// transaction does not root from the same database as the Ontology.
func (o *Ontology) NewWriter(tx gorp.Tx) Writer {
	return dagWriter{
		tx:                o.DB.OverrideTx(tx),
		registrar:         o.registrar,
		resourceTable:     o.resourceTable,
		relationshipTable: o.relationshipTable,
		relIndexes:        o.relIndexes,
	}
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
	return o.closer.Close()
}
