// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"
	"encoding/json/v2"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v0"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
)

// MigrateReadConfig lifts a v0 read config to v1. Its topic becomes a topic with no
// records, and its records move into Inline, for ResolveReadConfig to add back to that
// topic with an ICD.
func MigrateReadConfig(ctx context.Context, old v0.ReadConfig) (ReadConfig, error) {
	cfg, err := autoMigrateReadConfig(ctx, old)
	if err != nil {
		return cfg, err
	}
	cfg.Topics = []ReadTopic{}
	if old.Topic != "" {
		cfg.Topics = append(cfg.Topics, ReadTopic{
			Key:     uuid.New().String(),
			Name:    old.Topic,
			Records: []ReadRecord{},
		})
	}
	if len(old.Records) == 0 {
		return cfg, nil
	}
	staged := make([]InlineRecord, len(old.Records))
	for i, r := range old.Records {
		staged[i] = InlineRecord{
			Key:       r.Key,
			RecordKey: r.RecordKey,
			Index:     r.Index,
			Fields:    r.Fields,
		}
	}
	cfg.Inline = &InlineRead{Topic: old.Topic, Records: staged}
	return cfg, nil
}

// CreateICD creates ic through tx.
type CreateICD func(ctx context.Context, tx gorp.Tx, ic icd.ICD) error

// NoteLift returns the note for ResolveReadConfig and ResolveWriteConfig that logs each
// change to a field of the config with key through ins.
func NoteLift(ins alamos.Instrumentation, key uuid.UUID) func(path, change string) {
	return func(path, change string) {
		ins.L.Warn(
			"changed a Kafka task field to v1",
			zap.Stringer("config", key),
			zap.String("field", path),
			zap.String("change", change),
		)
	}
}

// StagedRecordsKVPrefix is the KV key prefix under which ReadMigration stages the
// records of each config. The rest of the key is the config key as a string, and the
// value is the InlineRead that holds the records, as JSON.
const StagedRecordsKVPrefix = "sy_kafka_read_staged_records/"

// StagedRecordsKVKey returns the staging KV key of the records of the config with the
// given key.
func StagedRecordsKVKey(key uuid.UUID) []byte {
	return []byte(StagedRecordsKVPrefix + key.String())
}

// ReadMigration lifts each stored v0 read config to v1. The v1 shape stores no inline
// records, so ReadMigration stages the records of each config under
// StagedRecordsKVPrefix.
var ReadMigration = migrate.New(
	"v63_kafka_read_stage",
	func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
		var olds []v0.ReadConfig
		if err := gorp.NewRetrieve[uuid.UUID, v0.ReadConfig]().
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
				if err = tx.Set(ctx, StagedRecordsKVKey(cfg.Key), b); err != nil {
					return err
				}
				cfg.Inline = nil
			}
			news[i] = cfg
		}
		return gorp.WrapWriter[uuid.UUID, ReadConfig](tx).Set(ctx, news...)
	},
)

// StagedChannelsKVPrefix is the KV key prefix under which WriteMigration stages the
// channels of each config. The rest of the key is the config key as a string, and the
// value is the InlineWrite that holds the channels, as JSON.
const StagedChannelsKVPrefix = "sy_kafka_write_staged_channels/"

// StagedChannelsKVKey returns the staging KV key of the channels of the write config
// with the given key.
func StagedChannelsKVKey(key uuid.UUID) []byte {
	return []byte(StagedChannelsKVPrefix + key.String())
}

// WriteMigration lifts each stored v0 write config to v1. It stages the channels of
// each config under StagedChannelsKVPrefix, with its topic, record key, and record.
var WriteMigration = migrate.New(
	"v63_kafka_write_targets",
	func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
		var olds []v0.WriteConfig
		if err := gorp.NewRetrieve[uuid.UUID, v0.WriteConfig]().
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
				if err = tx.Set(ctx, StagedChannelsKVKey(cfg.Key), b); err != nil {
					return err
				}
				cfg.Inline = nil
			}
			news[i] = cfg
		}
		return gorp.WrapWriter[uuid.UUID, WriteConfig](tx).Set(ctx, news...)
	},
)
