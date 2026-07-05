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
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Config is the configuration for opening an [Ontology].
type Config struct {
	// DB is the key-value database the ontology persists its resources and
	// relationships to.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Instrumentation is used for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
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

// Ontology exposes an ontology stored in a key-value database for reading and writing.
type Ontology struct {
	cfg                 Config
	resourceObserver    observe.Observer[iter.Seq[Change]]
	registrar           serviceRegistrar
	disconnectObservers []observe.Disconnect
	closer              io.MultiCloser
	resourceTable       *gorp.Table[string, Resource]
	relationshipTable   *gorp.Table[string, Relationship]
	relIndexes          relationshipIndexes
}

// Open opens the ontology using the given configuration. If the RootID resource does
// not exist, it will be created.
func Open(ctx context.Context, configs ...Config) (_ *Ontology, err error) {
	cfg, err := config.New(Config{}, configs...)
	if err != nil {
		return nil, err
	}
	o := &Ontology{
		cfg:              cfg,
		resourceObserver: observe.New[iter.Seq[Change]](),
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
	if o.relationshipTable, err = gorp.OpenTable(
		ctx,
		gorp.TableConfig[string, Relationship]{
			DB:              cfg.DB,
			Instrumentation: cfg.Instrumentation,
			Indexes:         o.relIndexes.all(),
			Migrations: []migrate.Migration{
				gorp.CodecMigration[string, Relationship]("msgpack_to_orc"),
			},
		}); !ok(err, o.relationshipTable) {
		return nil, err
	}

	if err = o.NewWriter(cfg.DB).DefineResources(ctx, RootID); !ok(err, nil) {
		return nil, err
	}

	return o, nil
}

// NewWriter opens a new Writer using the provided transaction.
func (o *Ontology) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:                o.cfg.DB.OverrideTx(tx),
		resourceTable:     o.resourceTable,
		relationshipTable: o.relationshipTable,
	}
}

// NewRetrieve opens a new Retrieve query, which can be used to traverse and read
// resources from the underlying ontology.
func (o *Ontology) NewRetrieve() Retrieve {
	return newRetrieve(
		o.registrar, o.cfg.DB, o.resourceTable, o.relationshipTable, o.relIndexes,
	)
}

// ObserveResources returns an observable that notifies callers of changes to the
// resources in the ontology.
func (o *Ontology) ObserveResources() observe.Observable[iter.Seq[Change]] {
	return o.resourceObserver
}

// ObserveRelationships returns an observable that notifies callers of changes to the
// relationships in the ontology.
func (o *Ontology) ObserveRelationships() observe.Observable[gorp.TxReader[string,
	Relationship]] {
	return o.relationshipTable.Observe()
}

// RelationshipExists reports whether the given relationship exists in the ontology.
// Reads are executed against tx, falling back to the underlying database when tx is
// nil.
func (o *Ontology) RelationshipExists(
	ctx context.Context,
	tx gorp.Tx,
	rel Relationship,
) (bool, error) {
	return o.relationshipTable.NewRetrieve().
		Where(gorp.MatchKeys[string, Relationship](rel.GorpKey())).
		Exists(ctx, o.cfg.DB.OverrideTx(tx))
}

// RegisterService registers a Service for a particular [Type] with the [Ontology].
// Ontology will execute queries for Entity information for the given Type using the
// provided Service. RegisterService panics if a Service is already registered for the
// given Type.
func (o *Ontology) RegisterService(svc Service) {
	o.cfg.L.Debug("registering service", zap.Stringer("type", svc.Type()))
	o.registrar.register(svc)
	o.disconnectObservers = append(
		o.disconnectObservers,
		svc.OnChange(o.resourceObserver.Notify),
	)
}

// Close closes the ontology and all of its observers and releases all resources.
func (o *Ontology) Close() error {
	for _, d := range o.disconnectObservers {
		d()
	}
	return o.closer.Close()
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
