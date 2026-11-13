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
	"slices"
	"strings"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/task/config/legacy"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/content"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// Store persists the configuration records of one task type. A record's key is the key
// of its task, and deleting the task deletes the record.
type Store interface {
	// Type returns the task type whose records the store owns.
	Type() string
	// Prepare decodes data into the store's record type under key and validates it
	// against snap, writing nothing. name is the name of the task that owns the
	// record. It returns the record to write, or an error wrapping
	// validate.ErrValidation when data does not decode or fails validation.
	Prepare(
		ctx context.Context,
		snap gorp.Snapshot,
		key uuid.UUID,
		name string,
		data msgpack.EncodedJSON,
	) (Record, error)
	// Edit applies the path actions at cursors to the record stored under key, then
	// prepares the result as Prepare does. Each cursor is past the segment that names
	// the config. Edit returns the record to write, and whether it differs from the
	// result of the actions alone, such as by a filled default. It returns an error
	// wrapping query.ErrNotFound when no record exists, or validate.ErrValidation when
	// an action or the result is not valid.
	Edit(
		ctx context.Context,
		snap gorp.Snapshot,
		key uuid.UUID,
		name string,
		cursors []patch.Cursor,
	) (Record, bool, error)
	// Read returns the record stored under key, or an error wrapping
	// query.ErrNotFound when no record exists.
	Read(
		ctx context.Context,
		snap gorp.Snapshot,
		key uuid.UUID,
	) (msgpack.EncodedJSON, error)
	// Hash returns the hash of the record stored under key, as Record.Hash does, or an
	// error wrapping query.ErrNotFound when no record exists.
	Hash(ctx context.Context, snap gorp.Snapshot, key uuid.UUID) (string, error)
	// Delete removes the records stored under keys. Delete is idempotent.
	Delete(ctx context.Context, tx gorp.Tx, keys ...uuid.UUID) error
	// Copy duplicates the record stored under from into a new record stored under to,
	// which must be the key of a task. It returns an error wrapping query.ErrNotFound
	// when from does not exist.
	Copy(ctx context.Context, tx gorp.Tx, from, to uuid.UUID) error
	// Decode converts a config blob at the given version to the current stored shape
	// with defaults applied. It reads no stored data. A version that predates the
	// type's ladder runs the type's legacy rewrite first. It returns a validation error
	// when data does not decode at version or version is newer than the store's.
	Decode(
		ctx context.Context,
		version imex.Version,
		data msgpack.EncodedJSON,
	) (Decoded, error)
	// Normalize decodes data as Decode does, then normalizes the result on tx as
	// Decoded.Normalize does.
	Normalize(
		ctx context.Context,
		tx gorp.Tx,
		version imex.Version,
		data msgpack.EncodedJSON,
	) (msgpack.EncodedJSON, error)
	// Version returns the current version of the store's config type, stamped on
	// exported envelopes.
	Version() imex.Version
	// OnDetach sets the hook that runs in the transaction of each delete that detaches
	// a target from a record, with the key of the record. An error from the hook fails
	// the delete. Set it before the store serves deletes.
	OnDetach(hook func(ctx context.Context, tx gorp.Tx, key uuid.UUID) error)
}

// Decoded is a config blob in the current stored shape that is not yet checked against
// stored resources. Only a Store builds one.
type Decoded struct {
	normalize func(ctx context.Context, tx gorp.Tx) (msgpack.EncodedJSON, error)
}

// Normalize drops each target of the config's references that does not exist in tx and
// that its reference can drop, then reconciles the config with the resources in tx.
func (d Decoded) Normalize(
	ctx context.Context,
	tx gorp.Tx,
) (msgpack.EncodedJSON, error) {
	return d.normalize(ctx, tx)
}

// Record is a validated configuration record that is not written yet. Only a Store
// builds one.
type Record struct {
	hash  string
	data  func() (msgpack.EncodedJSON, error)
	write func(ctx context.Context, tx gorp.Tx) error
}

// Data returns the record as Read returns it once written.
func (r Record) Data() (msgpack.EncodedJSON, error) { return r.data() }

// Hash returns 16 hexadecimal characters that hash every value of the record except its
// keys. A list entry added again under a new key hashes as the entry it replaces.
func (r Record) Hash() string { return r.hash }

// Write writes the record under its key, creating or overwriting it, after the write
// that the store's resolve hook returned. It returns a validation error when no task
// has the key.
func (r Record) Write(ctx context.Context, tx gorp.Tx) error { return r.write(ctx, tx) }

// ServiceConfig is the configuration for opening a Service. E is the record type
// the service stores.
type ServiceConfig[E gorp.Entry[uuid.UUID]] struct {
	// DB is the database config records are stored in.
	// [REQUIRED]
	DB *gorp.DB
	// Type is the task type whose records the service owns.
	// [REQUIRED]
	Type string
	// SetEntryKey stamps key onto an entry before it is stored.
	// [REQUIRED]
	SetEntryKey func(e *E, key uuid.UUID)
	// PatchEntry applies the path action at c, which is not done, to an entry.
	// [REQUIRED]
	PatchEntry func(e *E, c patch.Cursor) error
	// HashEntry writes the content of an entry to h.
	// [REQUIRED]
	HashEntry func(e *E, h *content.Hash)
	// Migrations is the stored-shape migration chain for the record type. Empty for a
	// type whose stored shape has never changed.
	Migrations []gorp.Migration
	// References are the references the records hold, besides the one to their task.
	// Opening the service drops each target that does not exist from stored entries.
	// [OPTIONAL]
	References []gorp.Reference[uuid.UUID, E]
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
	// ResolveEntry fills fields of a decoded entry from other resources, read through
	// snap, before the entry is validated. name is the name of the task that owns the
	// entry. It returns nil, or a write that Record.Write runs in its transaction
	// before it stores the entry.
	// [OPTIONAL] - nil when the entry depends on no other resource.
	ResolveEntry func(
		ctx context.Context,
		snap gorp.Snapshot,
		name string,
		e *E,
	) (func(context.Context, gorp.Tx) error, error)
	// ReconcileEntry changes a normalized entry to agree with other resources, read
	// through tx, so that a config from an older version or another Core passes
	// ResolveEntry. Normalize runs it at every version. Prepare and Edit do not.
	// [OPTIONAL] - nil when the entry depends on no other resource.
	ReconcileEntry func(ctx context.Context, tx gorp.Tx, e *E) error
	// ValidateEntry checks a decoded entry before it is stored.
	// [OPTIONAL] - nil when the entry type has no validation rules.
	ValidateEntry func(*E) error
	alamos.Instrumentation
}

// Override implements xconfig.Config.
func (c ServiceConfig[E]) Override(other ServiceConfig[E]) ServiceConfig[E] {
	c.DB = override.Nil(c.DB, other.DB)
	c.Type = override.String(c.Type, other.Type)
	c.SetEntryKey = override.Nil(c.SetEntryKey, other.SetEntryKey)
	c.PatchEntry = override.Nil(c.PatchEntry, other.PatchEntry)
	c.HashEntry = override.Nil(c.HashEntry, other.HashEntry)
	c.Migrations = override.Slice(c.Migrations, other.Migrations)
	c.References = override.Slice(c.References, other.References)
	c.Ladder = override.If(c.Ladder, other.Ladder, other.Ladder.Decode != nil)
	c.LegacyVersion = override.Numeric(c.LegacyVersion, other.LegacyVersion)
	c.Legacy = override.Nil(c.Legacy, other.Legacy)
	c.ResolveEntry = override.Nil(c.ResolveEntry, other.ResolveEntry)
	c.ReconcileEntry = override.Nil(c.ReconcileEntry, other.ReconcileEntry)
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
	v.NotNil("patch_entry", c.PatchEntry)
	v.NotNil("hash_entry", c.HashEntry)
	v.NotNil("ladder.decode", c.Ladder.Decode)
	return v.Error()
}

// Service stores the configuration records of the task type it is opened with. E is
// the record type, e.g. Service[ReadConfig].
type Service[E gorp.Entry[uuid.UUID]] struct {
	cfg   ServiceConfig[E]
	table *gorp.Table[uuid.UUID, E]
	// onDetach is nil until OnDetach sets it.
	onDetach func(ctx context.Context, tx gorp.Tx, key uuid.UUID) error
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
	s := &Service[E]{cfg: cfg}
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, E]{
		DB:   cfg.DB,
		Name: TableName(cfg.Type),
		Migrations: append(
			slices.Clone(cfg.Migrations),
			gorp.DanglingMigration("v61_dangling_references"),
		),
		References: append(
			[]gorp.Reference[uuid.UUID, E]{newTaskReference[E]()},
			cfg.References...,
		),
		OnDetach: func(ctx context.Context, tx gorp.Tx, e E) error {
			if s.onDetach == nil {
				return nil
			}
			return s.onDetach(ctx, tx, e.GorpKey())
		},
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, err
	}
	return s, nil
}

// tableSuffix ends the name of each table that stores config records.
const tableSuffix = "_config"

// TableName returns the name of the table that stores the config records of taskType.
func TableName(taskType string) string { return taskType + tableSuffix }

// TaskOf returns the ID of the task whose config record target names, and false when
// the table of target stores no config records.
func TaskOf(target gorp.Target) (resource.ID, bool) {
	if !strings.HasSuffix(target.Table, tableSuffix) {
		return resource.ID{}, false
	}
	return resource.ID{Type: resource.TypeTask, Key: target.Key}, true
}

// newTaskReference returns the reference from a record to the task that shares its key.
func newTaskReference[E gorp.Entry[uuid.UUID]]() *gorp.KeyReference[uuid.UUID, E, uuid.UUID] {
	return gorp.NewKeyReference(gorp.KeyReferenceConfig[uuid.UUID, E, uuid.UUID]{
		Name:     "key",
		Table:    string(resource.TypeTask),
		OnDelete: gorp.Cascade,
		Kind:     gorp.Place,
		Targets:  func(e *E) []uuid.UUID { return []uuid.UUID{(*e).GorpKey()} },
	})
}

// Type implements Store.
func (s *Service[E]) Type() string { return s.cfg.Type }

// Close closes the service and releases the resources it holds. Close is not safe
// to call concurrently with any other service methods.
func (s *Service[E]) Close() error { return s.table.Close() }

// Prepare implements Store.
func (s *Service[E]) Prepare(
	ctx context.Context,
	snap gorp.Snapshot,
	key uuid.UUID,
	name string,
	data msgpack.EncodedJSON,
) (Record, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return Record{}, errors.Wrapf(
			validate.ErrValidation, "encoding %s config: %s", s.cfg.Type, err,
		)
	}
	var e E
	if err := json.Unmarshal(b, &e); err != nil {
		return Record{}, errors.Wrapf(
			validate.ErrValidation, "decoding %s config: %s", s.cfg.Type, err,
		)
	}
	return s.prepare(ctx, snap, key, name, &e)
}

// Edit implements Store.
func (s *Service[E]) Edit(
	ctx context.Context,
	snap gorp.Snapshot,
	key uuid.UUID,
	name string,
	cursors []patch.Cursor,
) (Record, bool, error) {
	e, err := s.retrieve(ctx, snap, key)
	if err != nil {
		return Record{}, false, err
	}
	var filled bool
	for _, c := range cursors {
		if c = c.Partial(&filled); c.Done() {
			err = patch.Value(&e, c)
		} else {
			err = s.cfg.PatchEntry(&e, c)
		}
		if err != nil {
			return Record{}, false, err
		}
	}
	edited, keyed := s.hash(&e), e.GorpKey() == key
	if e, err = redecode(e); err != nil {
		return Record{}, false, err
	}
	rec, err := s.prepare(ctx, snap, key, name, &e)
	if err != nil {
		return Record{}, false, err
	}
	return rec, filled || !keyed || rec.hash != edited, nil
}

// redecode returns e as a decode of its JSON form gives it, which fills each default
// group whose members are all zero.
func redecode[E any](e E) (E, error) {
	var out E
	b, err := json.Marshal(e)
	if err != nil {
		return out, errors.Wrapf(validate.ErrValidation, "encoding config: %s", err)
	}
	if err = json.Unmarshal(b, &out); err != nil {
		var zero E
		return zero, errors.Wrapf(validate.ErrValidation, "decoding config: %s", err)
	}
	return out, nil
}

// prepare gives e key, resolves, and validates it.
func (s *Service[E]) prepare(
	ctx context.Context,
	snap gorp.Snapshot,
	key uuid.UUID,
	name string,
	e *E,
) (Record, error) {
	s.cfg.SetEntryKey(e, key)
	var resolved func(context.Context, gorp.Tx) error
	if s.cfg.ResolveEntry != nil {
		var err error
		if resolved, err = s.cfg.ResolveEntry(ctx, snap, name, e); err != nil {
			return Record{}, err
		}
	}
	if s.cfg.ValidateEntry != nil {
		if err := s.cfg.ValidateEntry(e); err != nil {
			return Record{}, err
		}
	}
	return Record{
		hash: s.hash(e),
		data: func() (msgpack.EncodedJSON, error) { return encode(*e) },
		write: func(ctx context.Context, tx gorp.Tx) error {
			if resolved != nil {
				if err := resolved(ctx, tx); err != nil {
					return err
				}
			}
			return s.create(ctx, tx, e)
		},
	}, nil
}

func (s *Service[E]) hash(e *E) string {
	h := content.NewHash()
	s.cfg.HashEntry(e, &h)
	return h.Sum()
}

func (s *Service[E]) create(ctx context.Context, tx gorp.Tx, e *E) error {
	return s.table.NewCreate().Entry(e).Exec(ctx, tx)
}

// Read implements Store.
func (s *Service[E]) Read(
	ctx context.Context,
	snap gorp.Snapshot,
	key uuid.UUID,
) (msgpack.EncodedJSON, error) {
	e, err := s.retrieve(ctx, snap, key)
	if err != nil {
		return nil, err
	}
	return encode(e)
}

// Hash implements Store.
func (s *Service[E]) Hash(
	ctx context.Context,
	snap gorp.Snapshot,
	key uuid.UUID,
) (string, error) {
	e, err := s.retrieve(ctx, snap, key)
	if err != nil {
		return "", err
	}
	return s.hash(&e), nil
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
		Exec(ctx, tx); err != nil &&
		!errors.Is(err, query.ErrNotFound) {
		return err
	}
	return nil
}

// Decode implements Store.
func (s *Service[E]) Decode(
	ctx context.Context,
	version imex.Version,
	data msgpack.EncodedJSON,
) (Decoded, error) {
	ladder := s.cfg.Ladder
	// A legacy rewrite or a migration gives a field that its source lacks the zero
	// value.
	older := version < ladder.Latest
	if version < ladder.First || version <= s.cfg.LegacyVersion {
		rewrite := legacy.Rewrite{}
		if s.cfg.Legacy != nil {
			rewrite = *s.cfg.Legacy
		}
		if rewrite.Validate != nil {
			if err := rewrite.Validate(data); err != nil {
				return Decoded{}, errors.Wrapf(
					validate.ErrValidation, "decoding %s config: %s", s.cfg.Type, err,
				)
			}
		}
		data, version = rewrite.Apply(data), ladder.First
	} else if version > ladder.Latest {
		return Decoded{}, imex.NewErrUnsupportedVersion(
			s.cfg.Type, version, ladder.Latest,
		)
	}
	e, err := ladder.Decode(ctx, version, data)
	if err != nil {
		return Decoded{}, errors.Wrapf(
			validate.ErrValidation, "decoding %s config: %s", s.cfg.Type, err,
		)
	}
	if older {
		e = defaults.Fill(e)
	}
	return Decoded{normalize: func(
		ctx context.Context,
		tx gorp.Tx,
	) (msgpack.EncodedJSON, error) {
		if _, err := s.table.DropDangling(ctx, tx, &e); err != nil {
			return nil, err
		}
		if s.cfg.ReconcileEntry != nil {
			if err := s.cfg.ReconcileEntry(ctx, tx, &e); err != nil {
				return nil, err
			}
		}
		return encode(e)
	}}, nil
}

// Normalize implements Store.
func (s *Service[E]) Normalize(
	ctx context.Context,
	tx gorp.Tx,
	version imex.Version,
	data msgpack.EncodedJSON,
) (msgpack.EncodedJSON, error) {
	d, err := s.Decode(ctx, version, data)
	if err != nil {
		return nil, err
	}
	return d.Normalize(ctx, tx)
}

// Version implements Store.
func (s *Service[E]) Version() imex.Version { return s.cfg.Ladder.Latest }

// OnDetach implements Store.
func (s *Service[E]) OnDetach(
	hook func(ctx context.Context, tx gorp.Tx, key uuid.UUID) error,
) {
	s.onDetach = hook
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
	snap gorp.Snapshot,
	key uuid.UUID,
) (e E, err error) {
	err = s.table.NewRetrieve().
		Where(gorp.MatchKeys[uuid.UUID, E](key)).
		Entry(&e).
		Exec(ctx, snap)
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
