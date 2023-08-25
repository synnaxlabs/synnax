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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/search"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type (
	// Schema is a set of definitions that describe the structure of a resource.
	Schema = schema.Schema
	// Resource is the underlying data structure of a resource.
	Resource = schema.Resource
	ID       = schema.ID
	// Type is a unique identifier for a particular class of resources (channel, user, etc.)
	Type = schema.Type
)

// Ontology exposes an ontology stored in a key-value database for reading and writing.
type Ontology struct {
	Config
	search struct {
		signal.Go
		*search.Index
	}
	registrar serviceRegistrar
}

type Config struct {
	alamos.Instrumentation
	DB           *gorp.DB
	EnableSearch *bool
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{
		EnableSearch: config.True(),
	}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("ontology")
	validate.NotNil(v, "cesium", c.DB)
	validate.NotNil(v, "EnableSearch", c.EnableSearch)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.EnableSearch = override.Nil(c.EnableSearch, other.EnableSearch)
	return c
}

// Open opens the ontology using the given configuration. If the RootID resource does not
// exist, it will be created.
func Open(ctx context.Context, configs ...Config) (*Ontology, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	o := &Ontology{
		Config:    cfg,
		registrar: serviceRegistrar{BuiltIn: &builtinService{}},
	}

	err = o.NewRetrieve().WhereIDs(RootID).Exec(ctx, cfg.DB)
	if errors.Is(err, query.NotFound) {
		err = o.NewWriter(cfg.DB).DefineResource(ctx, RootID)
	} else if err != nil {
		return nil, err
	}

	if *o.Config.EnableSearch {
		o.search.Index, err = search.New(search.Config{Instrumentation: cfg.Instrumentation})
		o.search.Go = signal.Wrap(ctx, signal.WithInstrumentation(cfg.Instrumentation))
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

func (o *Ontology) Search(ctx context.Context, req search.Request) ([]Resource, error) {
	ids, err := o.SearchIDs(ctx, req)
	if err != nil {
		return nil, err
	}
	var resources []Resource
	return resources, o.NewRetrieve().WhereIDs(ids...).Entries(&resources).Exec(ctx, o.DB)
}

func (o *Ontology) SearchIDs(ctx context.Context, req search.Request) ([]ID, error) {
	if !*o.Config.EnableSearch {
		return nil, errors.New("[ontology] - search is not enabled")
	}
	return o.search.Index.Search(ctx, req)
}

// NewWriter opens a new Writer using the provided transaction.
// Panics if the transaction does not root from the same database as the Ontology.
func (o *Ontology) NewWriter(tx gorp.Tx) Writer {
	return dagWriter{tx: o.DB.OverrideTx(tx), registrar: o.registrar}
}

// RegisterService registers a Service for a particular [Type] with the [Ontology].
// Ontology will execute queries for Entity information for the given Type using the
// provided Service. RegisterService panics if a Service is already registered for
// the given Type.
func (o *Ontology) RegisterService(s Service) {
	o.L.Debug("registering service", zap.Stringer("type", s.Schema().Type))
	o.registrar.register(s)

	if !*o.Config.EnableSearch {
		return
	}

	o.search.Go.Go(func(ctx context.Context) error {
		n := s.OpenNexter()
		err := o.search.Index.WithTx(func(tx search.Tx) error {
			r, ok, err := n.Next(ctx)
			for ; ok && err == nil; r, ok, err = n.Next(ctx) {
				if err = tx.Index(r); err != nil {
					return err
				}
			}
			return nil
		})
		return errors.CombineErrors(err, n.Close())
	}, signal.WithKeyf("startup-indexing-%s", s.Schema().Type))

	// Set up a change handler to index new resources.
	s.OnChange(func(ctx context.Context, i iter.Nexter[schema.Change]) {
		err := o.search.Index.WithTx(func(tx search.Tx) error {
			ch, ok, err := i.Next(ctx)
			for ; ok && err == nil; ch, ok, err = i.Next(ctx) {
				o.L.Info("indexing resource", zap.String("type", string(s.Schema().Type)))
				if err = tx.Apply(ch); err != nil {
					break
				}
			}
			return err
		})
		if err != nil {
			o.L.Error("failed to index resource",
				zap.String("type", string(s.Schema().Type)),
				zap.Error(err),
			)
		}
	})
}
