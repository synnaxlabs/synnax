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
	"encoding/json"
	"fmt"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task/common"
	v2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/set"
	"go.uber.org/zap"
)

// MigrationConfig is the configuration for NewMigration.
type MigrationConfig struct {
	// Ontology defines the config record relationships the migration creates.
	Ontology *ontology.Ontology
	// Configs routes each task type to the store that owns its config records.
	Configs common.ConfigRegistry
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

// typeRenames maps legacy task type strings to their ontology-safe replacements.
var typeRenames = map[string]string{
	"arc":         "arc_task",
	"Rack Status": "rack_status",
}

// retiredTypes are task types released builds created that have no successor: the Lua
// sequence task and the pre-cutover heartbeat and OPC scanner tasks. Their rows are
// staged and removed rather than treated as unconvertible.
var retiredTypes = set.New("sequence", "heartbeat", "opcScanner")

// hashConfig is a frozen copy of the config hash as of this version: xxhash64 of the
// JSON encoding as 16 lowercase hex characters.
func hashConfig(config msgpack.EncodedJSON) (string, error) {
	if config == nil {
		config = msgpack.EncodedJSON{}
	}
	b, err := json.Marshal(map[string]any(config))
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
// its type's config record store, parents the record to the task, and recomputes the
// config hash from the canonical stored form. Rows it cannot convert are quarantined
// under QuarantineKVPrefix and removed from service.
func NewMigration(cfg MigrationConfig) migrate.Migration {
	return gorp.NewMigration(
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
	tasks, err := collectEntries(
		ctx,
		gorp.WrapReader[v2.Key, v2.Task](tx),
		func(v2.Task) bool { return true },
	)
	if err != nil || len(tasks) == 0 {
		return err
	}
	rows := gorp.WrapWriter[v2.Key, v2.Task](tx)
	otgW := cfg.Ontology.NewWriter(tx)
	for _, t := range tasks {
		if retiredTypes.Contains(t.Type) {
			if err := retire(ctx, tx, ins, otgW, t); err != nil {
				return err
			}
			continue
		}
		newType := t.Type
		if renamed, ok := typeRenames[t.Type]; ok {
			newType = renamed
		}
		store, ok := cfg.Configs.Store(ontology.ResourceType(newType))
		if !ok {
			if err := quarantine(
				ctx, tx, ins, otgW, t, "unknown task type "+t.Type,
			); err != nil {
				return err
			}
			continue
		}
		// Stored blobs are all console-era, which every store's legacy rewrite
		// handles as version 0.
		converted, err := store.Normalize(0, t.Config)
		if err != nil {
			if err := quarantine(ctx, tx, ins, otgW, t, err.Error()); err != nil {
				return err
			}
			continue
		}
		recordKey := uuid.New()
		if err := store.Write(ctx, tx, recordKey, converted); err != nil {
			if err := store.Delete(ctx, tx, recordKey); err != nil {
				return err
			}
			if err := quarantine(ctx, tx, ins, otgW, t, err.Error()); err != nil {
				return err
			}
			continue
		}
		canonical, err := store.Read(ctx, tx, recordKey)
		if err != nil {
			return err
		}
		hash, err := hashConfig(configContent(canonical))
		if err != nil {
			return err
		}
		t.Type, t.Config, t.ConfigHash = newType, nil, hash
		if err := rows.Set(ctx, t); err != nil {
			return err
		}
		taskID := ontology.ID{
			Type: ontology.ResourceTypeTask,
			Key:  t.Key.String(),
		}
		if err := otgW.DefineResources(ctx, taskID); err != nil {
			return err
		}
		if err := otgW.DefineRelationships(
			ctx,
			ontology.ID{
				Type: ontology.ResourceType(newType),
				Key:  recordKey.String(),
			},
			ontology.RelationshipTypeParentOf,
			taskID,
		); err != nil {
			return err
		}
	}
	return nil
}

// stageAndRemove stages the JSON encoding of the legacy row in KV and removes the
// row, its status, and its ontology presence.
func stageAndRemove(
	ctx context.Context,
	tx gorp.Tx,
	otgW ontology.Writer,
	t v2.Task,
) error {
	b, err := json.Marshal(t)
	if err != nil {
		return err
	}
	if err := tx.Set(ctx, QuarantineKVKey(t.Key), b); err != nil {
		return err
	}
	if err := gorp.WrapWriter[v2.Key, v2.Task](tx).Delete(ctx, t.Key); err != nil {
		return err
	}
	taskID := ontology.ID{Type: ontology.ResourceTypeTask, Key: t.Key.String()}
	statusKey := taskID.String()
	if err := gorp.WrapWriter[string, status.Status[v2.StatusDetails]](tx).
		Delete(ctx, statusKey); err != nil {
		return err
	}
	return otgW.DeleteResources(ctx, taskID)
}

// quarantine stages and removes a row whose config cannot be converted, logging a
// warning once.
func quarantine(
	ctx context.Context,
	tx gorp.Tx,
	ins alamos.Instrumentation,
	otgW ontology.Writer,
	t v2.Task,
	reason string,
) error {
	if err := stageAndRemove(ctx, tx, otgW, t); err != nil {
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
	otgW ontology.Writer,
	t v2.Task,
) error {
	if err := stageAndRemove(ctx, tx, otgW, t); err != nil {
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

// collectEntries drains a reader into a slice of the entries matching keep.
// Mutating a gorp table while iterating it is unsafe, so callers gather first and
// write after.
func collectEntries[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	r gorp.Reader[K, E],
	keep func(E) bool,
) (out []E, err error) {
	iter, err := r.OpenIterator(gorp.IterOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	for iter.First(); iter.Valid(); iter.Next() {
		if e := iter.Value(ctx); e != nil && keep(*e) {
			out = append(out, *e)
		}
	}
	return out, iter.Error()
}
