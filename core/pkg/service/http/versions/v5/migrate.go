// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

import (
	"context"
	"encoding/json/v2"
	"slices"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
)

// MigrateReadConfig lifts a v4 read config to v5, with the fields of each endpoint
// staged in Inline. NaN fill turns on when an optional field has no fill value, because
// a v4 read wrote NaN for its missing value.
func MigrateReadConfig(ctx context.Context, old v4.ReadConfig) (ReadConfig, error) {
	cfg, err := autoMigrateReadConfig(ctx, old)
	if err != nil || len(old.Endpoints) == 0 {
		return cfg, err
	}
	staged := make([]InlineEndpoint, len(old.Endpoints))
	for i, ep := range old.Endpoints {
		staged[i] = InlineEndpoint{Key: ep.Key, Index: ep.Index, Fields: ep.Fields}
		cfg.NanFillEnabled = cfg.NanFillEnabled || slices.ContainsFunc(
			ep.Fields,
			func(f ReadField) bool { return f.Optional && f.FillValue == nil },
		)
	}
	cfg.Inline = &InlineRead{Endpoints: staged}
	return cfg, nil
}

// MigrateReadEndpoint lifts a v4 read endpoint to v5, with no ICD and no messages.
func MigrateReadEndpoint(
	ctx context.Context,
	old v4.ReadEndpoint,
) (ReadEndpoint, error) {
	return autoMigrateReadEndpoint(ctx, old)
}

// MigrateWriteConfig lifts a v4 write config to v5, with the key, channel, and fields
// of each endpoint staged in Inline.
func MigrateWriteConfig(ctx context.Context, old v4.WriteConfig) (WriteConfig, error) {
	cfg, err := autoMigrateWriteConfig(ctx, old)
	if err != nil || len(old.Endpoints) == 0 {
		return cfg, err
	}
	staged := make([]InlineWriteEndpoint, len(old.Endpoints))
	for i, ep := range old.Endpoints {
		staged[i] = InlineWriteEndpoint{
			Key:     ep.Key,
			Channel: ep.Channel,
			Fields:  ep.Fields,
		}
	}
	cfg.Inline = &InlineWrite{Endpoints: staged}
	return cfg, nil
}

// MigrateWriteEndpoint lifts a v4 write endpoint to v5, with no ICD and no message.
func MigrateWriteEndpoint(
	ctx context.Context,
	old v4.WriteEndpoint,
) (WriteEndpoint, error) {
	return autoMigrateWriteEndpoint(ctx, old)
}

// NoteLift returns the note for ResolveReadConfig that logs each change to a field of
// the config with key through ins.
func NoteLift(ins alamos.Instrumentation, key uuid.UUID) func(path, change string) {
	return noteLift(ins, "changed an HTTP read field to v5", key)
}

// NoteWriteLift returns the note for ResolveWriteConfig that logs each change to a
// field of the config with key through ins.
func NoteWriteLift(
	ins alamos.Instrumentation,
	key uuid.UUID,
) func(path, change string) {
	return noteLift(ins, "changed an HTTP write field to v5", key)
}

func noteLift(
	ins alamos.Instrumentation,
	message string,
	key uuid.UUID,
) func(path, change string) {
	return func(path, change string) {
		ins.L.Warn(
			message,
			zap.Stringer("config", key),
			zap.String("field", path),
			zap.String("change", change),
		)
	}
}

// StagedFieldsKVPrefix is the KV key prefix under which ReadMigration stages the fields
// of each config. The rest of the key is the config key as a string, and the value is
// the InlineRead that holds the fields, as JSON.
const StagedFieldsKVPrefix = "sy_http_read_staged_fields/"

// StagedFieldsKVKey returns the staging KV key of the fields of the config with the
// given key.
func StagedFieldsKVKey(key uuid.UUID) []byte {
	return []byte(StagedFieldsKVPrefix + key.String())
}

// ReadMigration lifts each stored v4 read config to v5. The v5 shape does not store
// Inline, so ReadMigration stages it under StagedFieldsKVPrefix.
var ReadMigration = migrate.New(
	"v63_http_read_stage",
	func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
		var olds []v4.ReadConfig
		if err := gorp.NewRetrieve[uuid.UUID, v4.ReadConfig]().
			Entries(&olds).
			Exec(ctx, tx); err != nil || len(olds) == 0 {
			return err
		}
		news := make([]ReadConfig, len(olds))
		for i, old := range olds {
			cfg, err := MigrateReadConfig(ctx, old)
			if err != nil {
				return errors.Wrapf(err, "config %s", old.Key)
			}
			if cfg.Inline != nil {
				b, err := json.Marshal(cfg.Inline)
				if err != nil {
					return errors.Wrapf(err, "config %s", old.Key)
				}
				if err = tx.Set(ctx, StagedFieldsKVKey(cfg.Key), b); err != nil {
					return err
				}
				cfg.Inline = nil
			}
			news[i] = cfg
		}
		return gorp.WrapWriter[uuid.UUID, ReadConfig](tx).Set(ctx, news...)
	},
)

// StagedWriteFieldsKVPrefix is the KV key prefix under which WriteMigration stages the
// channel and fields of each config. The rest of the key is the config key as a string,
// and the value is the InlineWrite that holds them, as JSON.
const StagedWriteFieldsKVPrefix = "sy_http_write_staged_fields/"

// StagedWriteFieldsKVKey returns the staging KV key of the channel and fields of the
// config with the given key.
func StagedWriteFieldsKVKey(key uuid.UUID) []byte {
	return []byte(StagedWriteFieldsKVPrefix + key.String())
}

// WriteMigration lifts each stored v4 write config to v5. The v5 shape does not store
// Inline, so WriteMigration stages it under StagedWriteFieldsKVPrefix.
var WriteMigration = migrate.New(
	"v63_http_write_stage",
	func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
		var olds []v4.WriteConfig
		if err := gorp.NewRetrieve[uuid.UUID, v4.WriteConfig]().
			Entries(&olds).
			Exec(ctx, tx); err != nil || len(olds) == 0 {
			return err
		}
		news := make([]WriteConfig, len(olds))
		for i, old := range olds {
			cfg, err := MigrateWriteConfig(ctx, old)
			if err != nil {
				return errors.Wrapf(err, "config %s", old.Key)
			}
			if cfg.Inline != nil {
				b, err := json.Marshal(cfg.Inline)
				if err != nil {
					return errors.Wrapf(err, "config %s", old.Key)
				}
				if err = tx.Set(ctx, StagedWriteFieldsKVKey(cfg.Key), b); err != nil {
					return err
				}
				cfg.Inline = nil
			}
			news[i] = cfg
		}
		return gorp.WrapWriter[uuid.UUID, WriteConfig](tx).Set(ctx, news...)
	},
)
