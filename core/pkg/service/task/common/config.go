// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package common

import (
	"context"
	"encoding/json"
	"iter"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
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

// ConfigStore persists the configuration records of one task type. Implementations
// are ontology services: each stored record is a resource of the store's type.
type ConfigStore interface {
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

// ConfigPtr constrains a pointer to a config record type.
type ConfigPtr[E any] interface {
	*E
	SetKey(uuid.UUID)
}

// ConfigServiceConfig is the configuration for opening a ConfigService.
type ConfigServiceConfig struct {
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
	alamos.Instrumentation
}

var _ config.Config[ConfigServiceConfig] = ConfigServiceConfig{}

// Override implements config.Config.
func (c ConfigServiceConfig) Override(other ConfigServiceConfig) ConfigServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Type = override.String(c.Type, other.Type)
	c.Migrations = override.Slice(c.Migrations, other.Migrations)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements config.Config.
func (c ConfigServiceConfig) Validate() error {
	v := validate.New("task.common.config")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotEmptyString(v, "type", c.Type)
	return v.Error()
}

// ConfigService stores the configuration records of the task type it is opened
// with. E is the record type and PE its pointer type, e.g.
// ConfigService[ReadConfig, *ReadConfig].
type ConfigService[E gorp.Entry[uuid.UUID], PE ConfigPtr[E]] struct {
	cfg   ConfigServiceConfig
	table *gorp.Table[uuid.UUID, E]
}

var configResourceSchema = zyn.Object(map[string]zyn.Schema{"key": zyn.UUID()})

// OpenConfigService opens a ConfigService for one task type. The service registers
// itself with the configured ontology and must be closed by calling Close.
func OpenConfigService[E gorp.Entry[uuid.UUID], PE ConfigPtr[E]](
	ctx context.Context,
	cfgs ...ConfigServiceConfig,
) (*ConfigService[E, PE], error) {
	cfg, err := config.New(ConfigServiceConfig{}, cfgs...)
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
	s := &ConfigService[E, PE]{cfg: cfg, table: table}
	cfg.Ontology.RegisterService(s)
	return s, nil
}

// Close closes the service and releases the resources it holds. Close is not safe
// to call concurrently with any other service methods.
func (s *ConfigService[E, PE]) Close() error { return s.table.Close() }

// Write implements ConfigStore.
func (s *ConfigService[E, PE]) Write(
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
	if ad, ok := any(&e).(interface{ ApplyDefaults() }); ok {
		ad.ApplyDefaults()
	}
	PE(&e).SetKey(key)
	if v, ok := any(&e).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	return s.create(ctx, tx, &e)
}

func (s *ConfigService[E, PE]) create(ctx context.Context, tx gorp.Tx, e *E) error {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	if err := s.table.NewCreate().Entry(e).Exec(ctx, tx); err != nil {
		return err
	}
	return s.cfg.Ontology.NewWriter(tx).DefineResources(
		ctx, ontology.ID{Type: s.cfg.Type, Key: (*e).GorpKey().String()},
	)
}

// Read implements ConfigStore.
func (s *ConfigService[E, PE]) Read(
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

// Delete implements ConfigStore.
func (s *ConfigService[E, PE]) Delete(
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

// Copy implements ConfigStore.
func (s *ConfigService[E, PE]) Copy(
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

func (s *ConfigService[E, PE]) retrieve(
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
func (s *ConfigService[E, PE]) Type() ontology.ResourceType { return s.cfg.Type }

// RetrieveResource implements ontology.Service.
func (s *ConfigService[E, PE]) RetrieveResource(
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
func (s *ConfigService[E, PE]) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, E]) {
		f(ctx, xiter.Map(reader, s.translateChange))
	}
	return s.table.Observe().OnChange(handleChange)
}

func (s *ConfigService[E, PE]) resource(e E) ontology.Resource {
	id := ontology.ID{Type: s.cfg.Type, Key: e.GorpKey().String()}
	return ontology.NewResource(configResourceSchema, id, string(s.cfg.Type), e)
}

func (s *ConfigService[E, PE]) translateChange(
	c xchange.Change[uuid.UUID, E],
) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     ontology.ID{Type: s.cfg.Type, Key: c.Key.String()}.String(),
		Value:   s.resource(c.Value),
	}
}

// ConfigRegistry routes task types to the ConfigStore that owns their records.
type ConfigRegistry struct {
	stores map[ontology.ResourceType]ConfigStore
}

// NewConfigRegistry builds a registry from the given stores. It returns an error
// when two stores declare the same type.
func NewConfigRegistry(stores ...ConfigStore) (ConfigRegistry, error) {
	m := make(map[ontology.ResourceType]ConfigStore, len(stores))
	for _, s := range stores {
		if _, ok := m[s.Type()]; ok {
			return ConfigRegistry{}, errors.Newf(
				"config store for %q registered twice", s.Type(),
			)
		}
		m[s.Type()] = s
	}
	return ConfigRegistry{stores: m}, nil
}

// IsZero reports whether the registry was never constructed.
func (r ConfigRegistry) IsZero() bool { return r.stores == nil }

// Store returns the store that owns records of the given task type, and false when
// no store claims it.
func (r ConfigRegistry) Store(t ontology.ResourceType) (ConfigStore, bool) {
	s, ok := r.stores[t]
	return s, ok
}

// Types returns the task types the registry routes, in no particular order.
func (r ConfigRegistry) Types() []ontology.ResourceType {
	types := make([]ontology.ResourceType, 0, len(r.stores))
	for t := range r.stores {
		types = append(types, t)
	}
	return types
}
