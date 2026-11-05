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
	"encoding/json/v2"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/task/config/legacy"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// Store persists the configuration records of one task type. A record's key is the key
// of its task, and deleting the task deletes the record.
type Store interface {
	// Type returns the task type whose records the store owns.
	Type() string
	// Validate decodes data into the store's record type under key and validates it.
	// It returns the record unstored, or an error wrapping validate.ErrValidation when
	// data does not decode or fails validation.
	Validate(
		ctx context.Context,
		tx gorp.Tx,
		key uuid.UUID,
		data msgpack.EncodedJSON,
	) (Record, error)
	// ValidateStrict is Validate, except that it also rejects data holding a member
	// that the record type does not declare.
	ValidateStrict(
		ctx context.Context,
		tx gorp.Tx,
		key uuid.UUID,
		data msgpack.EncodedJSON,
	) (Record, error)
	// Read returns the record stored under key, or an error wrapping
	// query.ErrNotFound when no record exists.
	Read(ctx context.Context, tx gorp.Tx, key uuid.UUID) (msgpack.EncodedJSON, error)
	// Delete removes the records stored under keys. Delete is idempotent.
	Delete(ctx context.Context, tx gorp.Tx, keys ...uuid.UUID) error
	// Copy duplicates the record stored under from into a new record stored under
	// to, which must be the key of a task. It returns an error wrapping
	// query.ErrNotFound when from does not exist.
	Copy(ctx context.Context, tx gorp.Tx, from, to uuid.UUID) error
	// Normalize converts a config blob at the given version to the current stored
	// shape with defaults applied. A version that predates the type's ladder runs the
	// type's legacy rewrite first. It returns a validation error when data does not
	// decode at version or version is newer than the store's.
	Normalize(
		ctx context.Context,
		version imex.Version,
		data msgpack.EncodedJSON,
	) (msgpack.EncodedJSON, error)
	// Version returns the current version of the store's config type, stamped on
	// exported envelopes.
	Version() imex.Version
}

// Record is a validated configuration record that is not stored yet.
type Record struct {
	// Data is the record as Read returns it once stored.
	Data  msgpack.EncodedJSON
	store func(ctx context.Context, tx gorp.Tx) error
}

// Store stores the record under its key, creating or overwriting it. It returns a
// validation error when no task has the key.
func (r Record) Store(ctx context.Context, tx gorp.Tx) error { return r.store(ctx, tx) }

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
	Migrations []gorp.Migration
	// Ladder decodes typed config blobs at every version the type has exported and
	// migrates them to the current shape. Its Latest is stamped on exports.
	// [REQUIRED]
	Ladder imex.Ladder[E]
	// LegacyVersion is the newest version released Cores stamped on a shape that needs
	// Legacy. Blobs at or below it, or below Ladder's First version, decode through
	// Legacy and enter Ladder at First.
	LegacyVersion imex.Version
	// Legacy is the rewrite that converts the type's legacy config shapes. Nil
	// applies era normalization alone.
	Legacy *legacy.Rewrite
	// ApplyEntryDefaults fills absent fields of a decoded entry.
	// [OPTIONAL] - nil when the entry type has no defaults.
	ApplyEntryDefaults func(*E)
	// ResolveEntry fills fields of a decoded entry from other resources, read through
	// tx, before the entry is validated and stored.
	// [OPTIONAL] - nil when the entry depends on no other resource.
	ResolveEntry func(ctx context.Context, tx gorp.Tx, e *E) error
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
	c.Ladder = override.If(c.Ladder, other.Ladder, other.Ladder.Decode != nil)
	c.LegacyVersion = override.Numeric(c.LegacyVersion, other.LegacyVersion)
	c.Legacy = override.Nil(c.Legacy, other.Legacy)
	c.ApplyEntryDefaults = override.Nil(c.ApplyEntryDefaults, other.ApplyEntryDefaults)
	c.ResolveEntry = override.Nil(c.ResolveEntry, other.ResolveEntry)
	c.ValidateEntry = override.Nil(c.ValidateEntry, other.ValidateEntry)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements xconfig.Config.
func (c ServiceConfig[E]) Validate() error {
	v := validate.New("task.config")
	v.NotNil("db", c.DB)
	v.NotEmptyString("type", c.Type)
	v.NotNil("set_entry_key", c.SetEntryKey)
	v.NotNil("ladder.decode", c.Ladder.Decode)
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
		Name:            cfg.Type + "_config",
		Migrations:      cfg.Migrations,
		References:      []*gorp.Reference[uuid.UUID, E]{newTaskReference[E]()},
		Instrumentation: cfg.Instrumentation,
	})
	if err != nil {
		return nil, err
	}
	return &Service[E]{cfg: cfg, table: table}, nil
}

// newTaskReference returns the reference from a record to the task that shares its key.
func newTaskReference[E gorp.Entry[uuid.UUID]]() *gorp.Reference[uuid.UUID, E] {
	return gorp.NewReference(gorp.ReferenceConfig[uuid.UUID, E]{
		Name:     "key",
		Tables:   []string{string(resource.TypeTask)},
		OnDelete: gorp.Cascade,
		Targets: func(e *E) []gorp.Target {
			return []gorp.Target{gorp.To(string(resource.TypeTask), (*e).GorpKey())}
		},
	})
}

// Type implements Store.
func (s *Service[E]) Type() string { return s.cfg.Type }

// Close closes the service and releases the resources it holds. Close is not safe
// to call concurrently with any other service methods.
func (s *Service[E]) Close() error { return s.table.Close() }

// Validate implements Store.
func (s *Service[E]) Validate(
	ctx context.Context,
	tx gorp.Tx,
	key uuid.UUID,
	data msgpack.EncodedJSON,
) (Record, error) {
	return s.validate(ctx, tx, key, data, json.RejectUnknownMembers(false))
}

// ValidateStrict implements Store.
func (s *Service[E]) ValidateStrict(
	ctx context.Context,
	tx gorp.Tx,
	key uuid.UUID,
	data msgpack.EncodedJSON,
) (Record, error) {
	return s.validate(ctx, tx, key, data, json.RejectUnknownMembers(true))
}

func (s *Service[E]) validate(
	ctx context.Context,
	tx gorp.Tx,
	key uuid.UUID,
	data msgpack.EncodedJSON,
	opts json.Options,
) (Record, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return Record{}, errors.Wrapf(
			validate.ErrValidation, "encoding %s config: %s", s.cfg.Type, err,
		)
	}
	var e E
	if err := json.Unmarshal(b, &e, opts); err != nil {
		return Record{}, errors.Wrapf(
			validate.ErrValidation, "decoding %s config: %s", s.cfg.Type, err,
		)
	}
	if s.cfg.ApplyEntryDefaults != nil {
		s.cfg.ApplyEntryDefaults(&e)
	}
	s.cfg.SetEntryKey(&e, key)
	if s.cfg.ResolveEntry != nil {
		if err := s.cfg.ResolveEntry(
			ctx,
			gorp.OverrideTx(s.cfg.DB, tx),
			&e,
		); err != nil {
			return Record{}, err
		}
	}
	if s.cfg.ValidateEntry != nil {
		if err := s.cfg.ValidateEntry(&e); err != nil {
			return Record{}, err
		}
	}
	encoded, err := encode(e)
	if err != nil {
		return Record{}, err
	}
	return Record{
		Data:  encoded,
		store: func(ctx context.Context, tx gorp.Tx) error { return s.create(ctx, tx, &e) },
	}, nil
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
	return encode(e)
}

func encode[E any](e E) (msgpack.EncodedJSON, error) {
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

// Normalize implements Store.
func (s *Service[E]) Normalize(
	ctx context.Context,
	version imex.Version,
	data msgpack.EncodedJSON,
) (msgpack.EncodedJSON, error) {
	ladder := s.cfg.Ladder
	if version < ladder.First || version <= s.cfg.LegacyVersion {
		rewrite := legacy.Rewrite{}
		if s.cfg.Legacy != nil {
			rewrite = *s.cfg.Legacy
		}
		data, version = rewrite.Apply(data), ladder.First
	} else if version > ladder.Latest {
		return nil, imex.NewErrUnsupportedVersion(s.cfg.Type, version, ladder.Latest)
	}
	e, err := ladder.Decode(ctx, version, data)
	if err != nil {
		return nil, errors.Wrapf(
			validate.ErrValidation, "decoding %s config: %s", s.cfg.Type, err,
		)
	}
	if s.cfg.ApplyEntryDefaults != nil {
		s.cfg.ApplyEntryDefaults(&e)
	}
	return encode(e)
}

// Version implements Store.
func (s *Service[E]) Version() imex.Version { return s.cfg.Ladder.Latest }

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

// Types returns the task types the registry routes, in no particular order.
func (r Registry) Types() []string {
	types := make([]string, 0, len(r.stores))
	for t := range r.stores {
		types = append(types, t)
	}
	return types
}
