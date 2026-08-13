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

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// Store persists the configuration records of one task type.
type Store interface {
	// Type returns the task type whose records the store owns.
	Type() string
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

// ServiceConfig is the configuration for opening a Service. E is the record type
// the service stores.
type ServiceConfig[E any] struct {
	// DB is the database config records are stored in.
	// [REQUIRED]
	DB *gorp.DB
	// Type is the task type whose records the service owns.
	// [REQUIRED]
	Type string
	// SetEntryKey stamps key onto an entry before it is stored.
	// [REQUIRED]
	SetEntryKey func(e *E, key uuid.UUID)
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
	c.Type = override.String(c.Type, other.Type)
	c.SetEntryKey = override.Nil(c.SetEntryKey, other.SetEntryKey)
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
	validate.NotEmptyString(v, "type", c.Type)
	validate.NotNil(v, "set_entry_key", c.SetEntryKey)
	return v.Error()
}

// Service stores the configuration records of the task type it is opened with. E is
// the record type, e.g. Service[ReadConfig].
type Service[E gorp.Entry[uuid.UUID]] struct {
	cfg   ServiceConfig[E]
	table *gorp.Table[uuid.UUID, E]
}

// OpenService opens a Service for one task type. The service must be closed by
// calling Close.
func OpenService[E gorp.Entry[uuid.UUID]](
	ctx context.Context,
	cfgs ...ServiceConfig[E],
) (*Service[E], error) {
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
	return &Service[E]{cfg: cfg, table: table}, nil
}

// Type implements Store.
func (s *Service[E]) Type() string { return s.cfg.Type }

// Close closes the service and releases the resources it holds. Close is not safe
// to call concurrently with any other service methods.
func (s *Service[E]) Close() error { return s.table.Close() }

// Write implements Store.
func (s *Service[E]) Write(
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
	s.cfg.SetEntryKey(&e, key)
	if s.cfg.ValidateEntry != nil {
		if err := s.cfg.ValidateEntry(&e); err != nil {
			return err
		}
	}
	return s.create(ctx, tx, &e)
}

func (s *Service[E]) create(ctx context.Context, tx gorp.Tx, e *E) error {
	return s.table.NewCreate().Entry(e).Exec(ctx, gorp.OverrideTx(s.cfg.DB, tx))
}

// Read implements Store.
func (s *Service[E]) Read(
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
func (s *Service[E]) Delete(
	ctx context.Context,
	tx gorp.Tx,
	keys ...uuid.UUID,
) error {
	if err := s.table.NewDelete().
		Where(gorp.MatchKeys[uuid.UUID, E](keys...)).
		Exec(ctx, gorp.OverrideTx(s.cfg.DB, tx)); err != nil &&
		!errors.Is(err, query.ErrNotFound) {
		return err
	}
	return nil
}

// Copy implements Store.
func (s *Service[E]) Copy(
	ctx context.Context,
	tx gorp.Tx,
	from, to uuid.UUID,
) error {
	e, err := s.retrieve(ctx, tx, from)
	if err != nil {
		return err
	}
	s.cfg.SetEntryKey(&e, to)
	return s.create(ctx, tx, &e)
}

func (s *Service[E]) retrieve(
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

// Registry routes task types to the Store that owns their records.
type Registry struct {
	stores map[string]Store
}

// NewRegistry builds a registry from the given stores. It returns an error
// when two stores declare the same type.
func NewRegistry(stores ...Store) (Registry, error) {
	m := make(map[string]Store, len(stores))
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
func (r Registry) Store(t string) (Store, bool) {
	s, ok := r.stores[t]
	return s, ok
}
