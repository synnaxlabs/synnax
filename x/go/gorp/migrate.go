// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import (
	"context"

	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
)

// Migration is a versioned schema migration that transforms entries stored in gorp.
type Migration interface {
	// Name returns a human-readable identifier for this migration.
	Name() string
	// Run executes the migration within the provided kv.Tx.
	Run(ctx context.Context, tx kv.Tx, cfg MigrationConfig) error
}

// MigrationConfig provides the configuration needed by Migration implementations
// to locate and encode/decode entries.
type MigrationConfig struct {
	Prefix []byte
	Codec  binary.Codec
}

// AutoMigrateFunc transforms an old entry into a new entry.
type AutoMigrateFunc[I, O any] func(ctx context.Context, old I) (O, error)

// PostMigrateFunc is called after the auto-migration to allow additional
// modifications to the new entry using data from the old entry.
type PostMigrateFunc[I, O any] func(ctx context.Context, new *O, old I) error

type typedMigration[I, O any] struct {
	name       string
	inputCodec Codec[I]
	outputCodec Codec[O]
	auto       AutoMigrateFunc[I, O]
	post       PostMigrateFunc[I, O]
}

// NewTypedMigration creates a Migration that iterates over all entries with the
// configured prefix, decodes each as type I, transforms it to type O via auto
// (and optionally post), and writes it back. Either auto or post may be nil but
// not both. When inputCodec is non-nil, it is used to decode entries instead of
// the DB's default codec. When outputCodec is non-nil, it is used to encode
// entries instead of the DB's default codec.
func NewTypedMigration[I, O any](
	name string,
	inputCodec Codec[I],
	outputCodec Codec[O],
	auto AutoMigrateFunc[I, O],
	post PostMigrateFunc[I, O],
) Migration {
	return &typedMigration[I, O]{
		name:        name,
		inputCodec:  inputCodec,
		outputCodec: outputCodec,
		auto:        auto,
		post:        post,
	}
}

func (m *typedMigration[I, O]) Name() string { return m.name }

func (m *typedMigration[I, O]) Run(
	ctx context.Context,
	kvTx kv.Tx,
	cfg MigrationConfig,
) error {
	iter, err := kvTx.OpenIterator(kv.IterPrefix(cfg.Prefix))
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	for iter.First(); iter.Valid(); iter.Next() {
		var old I
		if m.inputCodec != nil {
			old, err = m.inputCodec.Unmarshal(ctx, iter.Value())
		} else {
			err = cfg.Codec.Decode(ctx, iter.Value(), &old)
		}
		if err != nil {
			return err
		}
		var newEntry O
		if m.auto != nil {
			newEntry, err = m.auto(ctx, old)
			if err != nil {
				return err
			}
		}
		if m.post != nil {
			if err = m.post(ctx, &newEntry, old); err != nil {
				return err
			}
		}
		var data []byte
		if m.outputCodec != nil {
			data, err = m.outputCodec.Marshal(ctx, newEntry)
		} else {
			data, err = cfg.Codec.Encode(ctx, newEntry)
		}
		if err != nil {
			return err
		}
		if err = kvTx.Set(ctx, iter.Key(), data); err != nil {
			return err
		}
	}
	return err
}

type codecTransitionMigration[K Key, E Entry[K]] struct {
	name  string
	codec Codec[E]
}

// NewCodecTransition creates a Migration that re-encodes all entries from the DB's
// default codec (e.g. msgpack) to the provided target codec (e.g. protobuf).
func NewCodecTransition[K Key, E Entry[K]](name string, codec Codec[E]) Migration {
	return &codecTransitionMigration[K, E]{name: name, codec: codec}
}

func (m *codecTransitionMigration[K, E]) Name() string { return m.name }

func (m *codecTransitionMigration[K, E]) Run(
	ctx context.Context,
	kvTx kv.Tx,
	cfg MigrationConfig,
) error {
	iter, err := kvTx.OpenIterator(kv.IterPrefix(cfg.Prefix))
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	for iter.First(); iter.Valid(); iter.Next() {
		var entry E
		if err = cfg.Codec.Decode(ctx, iter.Value(), &entry); err != nil {
			return err
		}
		var data []byte
		if data, err = m.codec.Marshal(ctx, entry); err != nil {
			return err
		}
		if err = kvTx.Set(ctx, iter.Key(), data); err != nil {
			return err
		}
	}
	return err
}

type rawMigration struct {
	name string
	fn   func(ctx context.Context, tx Tx) error
}

// NewRawMigration creates a Migration that receives a fully wrapped gorp.Tx,
// allowing arbitrary read/write operations on the store.
func NewRawMigration(
	name string,
	fn func(ctx context.Context, tx Tx) error,
) Migration {
	return &rawMigration{name: name, fn: fn}
}

func (m *rawMigration) Name() string { return m.name }

func (m *rawMigration) Run(
	ctx context.Context,
	kvTx kv.Tx,
	cfg MigrationConfig,
) error {
	return m.fn(ctx, WrapTx(kvTx, cfg.Codec))
}
