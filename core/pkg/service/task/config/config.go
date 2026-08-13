// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package config

import (
	"context"
	"encoding/json"
	"iter"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	xchange "github.com/synnaxlabs/x/change"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

// Store persists the configuration records of one task type. Implementations
// are ontology services: each stored record is a resource of the store's type.
type Store interface {
	ontology.Service
	// Write decodes data into the store's record type, validates it, and stores it
	// under key, creating or overwriting the record. It returns an error wrapping
	// validate.ErrValidation when data does not decode or fails validation.
	Write(
		ctx context.Context,
		tx gorp.Tx,
		key uuid.UUID,
		data msgpack.EncodedJSON,
	) error
	// Read returns the record stored under key, or an error wrapping
	// query.ErrNotFound when no record exists.
	Read(ctx context.Context, tx gorp.Tx, key uuid.UUID) (msgpack.EncodedJSON, error)
	// Delete removes the records stored under keys. Delete is idempotent.
	Delete(ctx context.Context, tx gorp.Tx, keys ...uuid.UUID) error
	// Copy duplicates the record stored under from into a new record stored under
	// to. It returns an error wrapping query.ErrNotFound when from does not exist.
	Copy(ctx context.Context, tx gorp.Tx, from, to uuid.UUID) error
}

// Ptr constrains a pointer to a config record type.
type Ptr[E any] interface {
	*E
	SetKey(uuid.UUID)
}

// ServiceConfig is the configuration for opening a Service. E is the record type
// the service stores.
type ServiceConfig[E any] struct {
	// DB is the database config records are stored in.
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is used to register the record resource type.
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Type is the task type whose records the service owns.
	// [REQUIRED]
	Type ontology.ResourceType
	// Migrations is the stored-shape migration chain for the record type. Empty for
	// a type whose stored shape has never changed.
	Migrations []migrate.Migration
	// ApplyEntryDefaults fills absent fields of a decoded entry before it is stored.
	// [OPTIONAL] - nil when the entry type has no defaults.
	ApplyEntryDefaults func(*E)
	// ValidateEntry checks a decoded entry before it is stored.
	// [OPTIONAL] - nil when the entry type has no validation rules.
	ValidateEntry func(*E) error
	alamos.Instrumentation
}

var _ xconfig.Config[ServiceConfig[struct{}]] = ServiceConfig[struct{}]{}

// Override implements xconfig.Config.
func (c ServiceConfig[E]) Override(other ServiceConfig[E]) ServiceConfig[E] {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Type = override.String(c.Type, other.Type)
	c.Migrations = override.Slice(c.Migrations, other.Migrations)
	c.ApplyEntryDefaults = override.Nil(c.ApplyEntryDefaults, other.ApplyEntryDefaults)
	c.ValidateEntry = override.Nil(c.ValidateEntry, other.ValidateEntry)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements xconfig.Config.
func (c ServiceConfig[E]) Validate() error {
	v := validate.New("task.config")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotEmptyString(v, "type", c.Type)
	return v.Error()
}

// Service stores the configuration records of the task type it is opened with. E is
// the record type and PE its pointer type, e.g. Service[ReadConfig, *ReadConfig].
type Service[E gorp.Entry[uuid.UUID], PE Ptr[E]] struct {
	cfg   ServiceConfig[E]
	table *gorp.Table[uuid.UUID, E]
}

var resourceSchema = zyn.Object(map[string]zyn.Schema{"key": zyn.UUID()})

// OpenService opens a Service for one task type. The service registers itself with
// the configured ontology and must be closed by calling Close.
func OpenService[E gorp.Entry[uuid.UUID], PE Ptr[E]](
	ctx context.Context,
	cfgs ...ServiceConfig[E],
) (*Service[E, PE], error) {
	cfg, err := xconfig.New(ServiceConfig[E]{}, cfgs...)
	if err != nil {
		return nil, err
	}
	table, err := gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, E]{
		DB:              cfg.DB,
		Migrations:      cfg.Migrations,
		Instrumentation: cfg.Instrumentation,
	})
	if err != nil {
		return nil, err
	}
	s := &Service[E, PE]{cfg: cfg, table: table}
	cfg.Ontology.RegisterService(s)
	return s, nil
}

// Close closes the service and releases the resources it holds. Close is not safe
// to call concurrently with any other service methods.
func (s *Service[E, PE]) Close() error { return s.table.Close() }

// Write implements Store.
func (s *Service[E, PE]) Write(
	ctx context.Context,
	tx gorp.Tx,
	key uuid.UUID,
	data msgpack.EncodedJSON,
) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	var e E
	if err := json.Unmarshal(b, &e); err != nil {
		return errors.Wrapf(
			validate.ErrValidation, "decoding %s config: %s", s.cfg.Type, err,
		)
	}
	if s.cfg.ApplyEntryDefaults != nil {
		s.cfg.ApplyEntryDefaults(&e)
	}
	PE(&e).SetKey(key)
	if s.cfg.ValidateEntry != nil {
		if err := s.cfg.ValidateEntry(&e); err != nil {
			return err
		}
	}
	return s.create(ctx, tx, &e)
}

func (s *Service[E, PE]) create(ctx context.Context, tx gorp.Tx, e *E) error {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	if err := s.table.NewCreate().Entry(e).Exec(ctx, tx); err != nil {
		return err
	}
	return s.cfg.Ontology.NewWriter(tx).DefineResources(
		ctx, ontology.ID{Type: s.cfg.Type, Key: (*e).GorpKey().String()},
	)
}

// Read implements Store.
func (s *Service[E, PE]) Read(
	ctx context.Context,
	tx gorp.Tx,
	key uuid.UUID,
) (msgpack.EncodedJSON, error) {
	e, err := s.retrieve(ctx, tx, key)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	var data msgpack.EncodedJSON
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// Delete implements Store.
func (s *Service[E, PE]) Delete(
	ctx context.Context,
	tx gorp.Tx,
	keys ...uuid.UUID,
) error {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	if err := s.table.NewDelete().
		Where(gorp.MatchKeys[uuid.UUID, E](keys...)).
		Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return err
	}
	ids := make([]ontology.ID, len(keys))
	for i, k := range keys {
		ids[i] = ontology.ID{Type: s.cfg.Type, Key: k.String()}
	}
	return s.cfg.Ontology.NewWriter(tx).DeleteResources(ctx, ids...)
}

// Copy implements Store.
func (s *Service[E, PE]) Copy(
	ctx context.Context,
	tx gorp.Tx,
	from, to uuid.UUID,
) error {
	e, err := s.retrieve(ctx, tx, from)
	if err != nil {
		return err
	}
	PE(&e).SetKey(to)
	return s.create(ctx, tx, &e)
}

func (s *Service[E, PE]) retrieve(
	ctx context.Context,
	tx gorp.Tx,
	key uuid.UUID,
) (e E, err error) {
	err = s.table.NewRetrieve().
		Where(gorp.MatchKeys[uuid.UUID, E](key)).
		Entry(&e).
		Exec(ctx, gorp.OverrideTx(s.cfg.DB, tx))
	if err != nil {
		return e, errors.Wrapf(err, "%s config %s", s.cfg.Type, key)
	}
	return e, nil
}

// Type implements ontology.Service.
func (s *Service[E, PE]) Type() ontology.ResourceType { return s.cfg.Type }

// RetrieveResource implements ontology.Service.
func (s *Service[E, PE]) RetrieveResource(
	ctx context.Context,
	key string,
	tx gorp.Tx,
) (ontology.Resource, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	e, err := s.retrieve(ctx, tx, k)
	if err != nil {
		return ontology.Resource{}, err
	}
	return s.resource(e), nil
}

// OnChange implements ontology.Service.
func (s *Service[E, PE]) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, E]) {
		f(ctx, xiter.Map(reader, s.translateChange))
	}
	return s.table.Observe().OnChange(handleChange)
}

func (s *Service[E, PE]) resource(e E) ontology.Resource {
	id := ontology.ID{Type: s.cfg.Type, Key: e.GorpKey().String()}
	return ontology.NewResource(resourceSchema, id, string(s.cfg.Type), e)
}

func (s *Service[E, PE]) translateChange(
	c xchange.Change[uuid.UUID, E],
) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     ontology.ID{Type: s.cfg.Type, Key: c.Key.String()}.String(),
		Value:   s.resource(c.Value),
	}
}

// Registry routes task types to the Store that owns their records.
type Registry struct {
	stores map[ontology.ResourceType]Store
}

// NewRegistry builds a registry from the given stores. It returns an error
// when two stores declare the same type.
func NewRegistry(stores ...Store) (Registry, error) {
	m := make(map[ontology.ResourceType]Store, len(stores))
	for _, s := range stores {
		if _, ok := m[s.Type()]; ok {
			return Registry{}, errors.Newf(
				"config store for %q registered twice", s.Type(),
			)
		}
		m[s.Type()] = s
	}
	return Registry{stores: m}, nil
}

// IsZero reports whether the registry was never constructed.
func (r Registry) IsZero() bool { return r.stores == nil }

// Store returns the store that owns records of the given task type, and false when
// no store claims it.
func (r Registry) Store(t ontology.ResourceType) (Store, bool) {
	s, ok := r.stores[t]
	return s, ok
}

// Types returns the task types the registry routes, in no particular order.
func (r Registry) Types() []ontology.ResourceType {
	types := make([]ontology.ResourceType, 0, len(r.stores))
	for t := range r.stores {
		types = append(types, t)
	}
	return types
}
