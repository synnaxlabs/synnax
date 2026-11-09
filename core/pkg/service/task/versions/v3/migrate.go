// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"context"
	"encoding/json/v2"
	"fmt"

	"github.com/cespare/xxhash/v2"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	v2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"go.uber.org/zap"
)

// MigrationConfig is the configuration for NewMigration.
type MigrationConfig struct {
	// Status deletes the statuses of the tasks the migration removes.
	Status *status.Service
	// Configs routes each task type to the store that owns its config records.
	Configs config.Registry
}

// QuarantineKVPrefix is the KV key prefix under which the migration stages task rows
// it cannot convert. The remainder of the key is the task's UUID string; the value is
// the JSON encoding of the legacy row, raw config included.
const QuarantineKVPrefix = "sy_task_quarantine/"

// QuarantineKVKey returns the staging KV key for the quarantined task with the given
// key.
func QuarantineKVKey(key v2.Key) []byte {
	return []byte(QuarantineKVPrefix + key.String())
}

// typeRenames maps legacy task type strings to their replacements.
var typeRenames = map[string]string{"Rack Status": "rack_status"}

// retiredTypes are task types released builds created that have no successor: the Lua
// sequence task and the pre-cutover heartbeat and OPC scanner tasks. Their rows are
// staged and removed rather than treated as unconvertible. "Rack State" (drivers 0.41
// through 0.48) is retired, not renamed: 0.49+ drivers created a "Rack Status" twin
// without deleting the old row, so renaming both would double up; the config was
// always empty and the Driver recreates the task on connect.
var retiredTypes = set.New("sequence", "heartbeat", "opcScanner", "Rack State")

// hashConfig is a frozen copy of the config hash as of this version: xxhash64 of the
// JSON encoding as 16 lowercase hex characters.
func hashConfig(config msgpack.EncodedJSON) (string, error) {
	if config == nil {
		config = msgpack.EncodedJSON{}
	}
	b, err := json.Marshal(map[string]any(config), json.Deterministic(true))
	if err != nil {
		return "", errors.Wrap(err, "failed to hash task config")
	}
	return fmt.Sprintf("%016x", xxhash.Sum64(b)), nil
}

// configContent is a frozen copy of the hash input rule as of this version: the
// canonical config without the record key.
func configContent(config msgpack.EncodedJSON) msgpack.EncodedJSON {
	content := make(msgpack.EncodedJSON, len(config))
	for k, v := range config {
		if k != "key" {
			content[k] = v
		}
	}
	return content
}

// NewMigration returns the v3 migration, which moves every task's inline config into
// its type's config record store under the task's key and recomputes the config hash
// from the canonical stored form. Rows it cannot convert are quarantined under
// QuarantineKVPrefix and removed from service.
func NewMigration(cfg MigrationConfig) gorp.Migration {
	return migrate.New(
		"v57_task_config_records",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			return migrateConfigsToRecords(ctx, tx, ins, cfg)
		},
	)
}

func migrateConfigsToRecords(
	ctx context.Context,
	tx gorp.Tx,
	ins alamos.Instrumentation,
	cfg MigrationConfig,
) error {
	var tasks []v2.Task
	err := gorp.NewRetrieve[v2.Key, v2.Task]().
		Entries(&tasks).
		Exec(ctx, tx)
	if err != nil || len(tasks) == 0 {
		return err
	}
	var (
		rows    = gorp.WrapWriter[v2.Key, v2.Task](tx)
		removed []resource.ID
	)
	for _, t := range tasks {
		id := resource.ID{Type: resource.TypeTask, Key: t.Key.String()}
		if retiredTypes.Contains(t.Type) {
			if err := retire(ctx, tx, ins, cfg.Status, t); err != nil {
				return err
			}
			removed = append(removed, id)
			continue
		}
		newType := t.Type
		if renamed, ok := typeRenames[t.Type]; ok {
			newType = renamed
		}
		store, ok := cfg.Configs.Store(newType)
		if !ok {
			if err := quarantine(
				ctx, tx, ins, cfg.Status, t, "unknown task type "+t.Type,
			); err != nil {
				return err
			}
			removed = append(removed, id)
			continue
		}
		// Stored blobs are all console-era, which every store's legacy rewrite
		// handles as version 0.
		converted, err := store.Normalize(ctx, tx, 0, t.Config)
		if err != nil {
			if err := quarantine(
				ctx, tx, ins, cfg.Status, t, unconvertibleReason(err),
			); err != nil {
				return err
			}
			removed = append(removed, id)
			continue
		}
		rec, err := store.Prepare(ctx, tx, t.Key, t.Name, converted)
		if err == nil {
			err = rec.Write(ctx, tx)
		}
		if err != nil {
			// A write that the resolve hook ran stays, so the user keeps what it made,
			// such as a secret.
			if err := store.Delete(ctx, tx, t.Key); err != nil {
				return err
			}
			if err := quarantine(
				ctx, tx, ins, cfg.Status, t, unconvertibleReason(err),
			); err != nil {
				return err
			}
			removed = append(removed, id)
			continue
		}
		stored, err := rec.Data()
		if err != nil {
			return err
		}
		hash, err := hashConfig(configContent(stored))
		if err != nil {
			return err
		}
		t.Type, t.Config, t.ConfigHash = newType, nil, hash
		if err := rows.Set(ctx, t); err != nil {
			return err
		}
	}
	return ontology.DeleteResources(ctx, tx, removed...)
}

// stageAndRemove stages the JSON encoding of the legacy row in KV and removes the
// row and its status. A config JSON cannot encode (a NaN float) is staged without the
// config rather than aborting boot.
func stageAndRemove(
	ctx context.Context,
	tx gorp.Tx,
	ins alamos.Instrumentation,
	statuses *status.Service,
	t v2.Task,
) error {
	b, err := json.Marshal(t, json.Deterministic(true))
	if err != nil {
		marshalErr := err
		stripped := t
		stripped.Config = nil
		if b, err = json.Marshal(stripped, json.Deterministic(true)); err != nil {
			return err
		}
		ins.L.Warn(
			"task config could not be encoded; staged without config",
			zap.String("key", t.Key.String()),
			zap.String("name", t.Name),
			zap.String("type", t.Type),
			zap.Error(marshalErr),
		)
	}
	if err := tx.Set(ctx, QuarantineKVKey(t.Key), b); err != nil {
		return err
	}
	if err := gorp.WrapWriter[v2.Key, v2.Task](tx).Delete(ctx, t.Key); err != nil {
		return err
	}
	return statuses.NewWriter(tx).DeleteOwned(
		ctx,
		resource.ID{Type: resource.TypeTask, Key: t.Key.String()},
	)
}

// unconvertibleReason frames a raw conversion or encode error as a quarantine reason
// an operator can act on.
func unconvertibleReason(err error) string {
	return "config could not be converted to the current format: " + err.Error()
}

// quarantine stages and removes a row whose config cannot be converted, logging a
// warning once.
func quarantine(
	ctx context.Context,
	tx gorp.Tx,
	ins alamos.Instrumentation,
	statuses *status.Service,
	t v2.Task,
	reason string,
) error {
	if err := stageAndRemove(ctx, tx, ins, statuses, t); err != nil {
		return err
	}
	ins.L.Warn(
		"quarantined task with unconvertible config",
		zap.String("key", t.Key.String()),
		zap.String("name", t.Name),
		zap.String("type", t.Type),
		zap.String("reason", reason),
	)
	return nil
}

// retire stages and removes a row of a retired task type, logging the removal once.
func retire(
	ctx context.Context,
	tx gorp.Tx,
	ins alamos.Instrumentation,
	statuses *status.Service,
	t v2.Task,
) error {
	if err := stageAndRemove(ctx, tx, ins, statuses, t); err != nil {
		return err
	}
	ins.L.Info(
		"removed task of retired type",
		zap.String("key", t.Key.String()),
		zap.String("name", t.Name),
		zap.String("type", t.Type),
	)
	return nil
}
