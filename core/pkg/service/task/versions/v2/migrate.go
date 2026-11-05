// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"strconv"
	"strings"
	"uuid"

	"github.com/cespare/xxhash/v2"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	rack "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v2"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	v1 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v1"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// LegacyKeyKVPrefix is the KV key prefix under which the re-key migration stages the
// mapping from a task's legacy uint64 key to its new UUID. The remainder of the key
// is the legacy key's decimal string; the value is the UUID string. The panel
// service's composition migration, which the service layer runs after every table is
// open, consumes these entries to re-key task references in converted layouts.
const LegacyKeyKVPrefix = "sy_task_legacy_key/"

// LegacyKeyKVKey returns the staging KV key holding the UUID for the task with the
// given legacy key.
func LegacyKeyKVKey(legacy v1.Key) []byte {
	return []byte(LegacyKeyKVPrefix + strconv.FormatUint(uint64(legacy), 10))
}

// MigrationConfig configures the migration that NewMigration returns.
type MigrationConfig struct {
	// Status rewrites the owners of task statuses.
	Status *status.Service
}

// NewMigration returns the migration that re-keys stored tasks from legacy composite
// uint64 keys to UUIDs.
func NewMigration(cfg MigrationConfig) gorp.Migration {
	return migrate.New(
		"v56_task_uuid_key",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			return migrateKeysToUUID(ctx, tx, cfg)
		},
	)
}

// resourceID returns the resource identifier for the task with the given key.
func resourceID(key Key) resource.ID {
	return resource.ID{Type: resource.TypeTask, Key: key.String()}
}

// hashConfig is a frozen copy of the config hash as of this version: xxhash64 of the
// JSON encoding as 16 lowercase hex characters. encoding/json sorts map keys, so equal
// configs hash equally.
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

// migrateKeysToUUID re-keys every task from the legacy composite uint64 key to a
// freshly minted UUID. The rack encoded in the legacy key's high 32 bits becomes the
// task's rack field. Task rows, task statuses, ontology resources, and ontology
// relationships are all rewritten; each legacy-to-UUID pair is staged in KV under
// LegacyKeyKVPrefix for the panel migration. The whole re-key runs in one migration
// transaction, so it commits atomically.
func migrateKeysToUUID(ctx context.Context, tx gorp.Tx, cfg MigrationConfig) error {
	stale, err := collectEntries(
		ctx,
		gorp.WrapReader[v1.Key, v1.Task](tx),
		func(v1.Task) bool { return true },
	)
	if err != nil || len(stale) == 0 {
		return err
	}
	keys := make(map[v1.Key]Key, len(stale))
	tasks := make([]Task, len(stale))
	oldKeys := make([]v1.Key, len(stale))
	for i, old := range stale {
		key := uuid.New()
		keys[old.Key] = key
		oldKeys[i] = old.Key
		configHash, err := hashConfig(old.Config)
		if err != nil {
			return err
		}
		tasks[i] = Task{
			Key:        key,
			Rack:       rack.Key(old.Key >> 32),
			Name:       old.Name,
			Type:       old.Type,
			Config:     old.Config,
			ConfigHash: configHash,
			Internal:   old.Internal,
			Snapshot:   old.Snapshot,
		}
	}
	byKey := make(map[Key]Task, len(tasks))
	for _, t := range tasks {
		byKey[t.Key] = t
	}
	if err := gorp.WrapWriter[Key, Task](tx).Set(ctx, tasks...); err != nil {
		return err
	}
	if err := gorp.WrapWriter[v1.Key, v1.Task](tx).Delete(ctx, oldKeys...); err != nil {
		return err
	}
	if err := rewriteStatuses(
		ctx,
		cfg.Status.NewMigrationWriter(tx),
		tx,
		keys,
		byKey,
	); err != nil {
		return err
	}
	if err := rewriteResources(ctx, tx, keys); err != nil {
		return err
	}
	if err := rewriteRelationships(ctx, tx, keys); err != nil {
		return err
	}
	for old, key := range keys {
		if err := tx.Set(ctx, LegacyKeyKVKey(old), []byte(key.String())); err != nil {
			return err
		}
	}
	return nil
}

// parseLegacyKey parses a legacy uint64 task key out of a resource ID key, returning
// false when the string is not a legacy key.
func parseLegacyKey(key string) (v1.Key, bool) {
	n, err := strconv.ParseUint(key, 10, 64)
	if err != nil {
		return 0, false
	}
	return v1.Key(n), true
}

// rewriteStatuses points every task status owned by a legacy task key at the task's
// UUID key, and rewrites the rack inside its details. A status whose task has no live
// record (already deleted pre-migration) is deleted.
func rewriteStatuses(
	ctx context.Context,
	w *gorp.Writer[status.Key, status.Status[any]],
	tx gorp.Tx,
	keys map[v1.Key]Key,
	byKey map[Key]Task,
) error {
	var stale []status.Status[v1.StatusDetails]
	if err := gorp.NewRetrieve[string, status.Status[v1.StatusDetails]]().
		Where(gorp.Match(func(
			_ gorp.Context,
			s *status.Status[v1.StatusDetails],
		) (bool, error) {
			return s.Owner != nil && s.Owner.Type == resource.TypeTask, nil
		})).
		Entries(&stale).
		Exec(ctx, tx); err != nil {
		return err
	}
	for _, s := range stale {
		legacy, ok := parseLegacyKey(s.Owner.Key)
		if !ok {
			continue
		}
		key, ok := keys[legacy]
		if !ok {
			if err := w.Delete(ctx, s.Key); err != nil {
				return err
			}
			continue
		}
		if err := w.Set(ctx, status.Status[any]{
			Key:         s.Key,
			Owner:       new(resourceID(key)),
			Name:        s.Name,
			Variant:     s.Variant,
			Message:     s.Message,
			Description: s.Description,
			Time:        s.Time,
			Labels:      s.Labels,
			Details: StatusDetails{
				Running: s.Details.Running,
				Cmd:     s.Details.Cmd,
				Rack:    byKey[key].Rack,
				Data:    s.Details.Data,
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

// rewriteResources re-keys every task ontology resource to its UUID and every task
// status resource to the re-keyed status key.
func rewriteResources(ctx context.Context, tx gorp.Tx, keys map[v1.Key]Key) error {
	statusKeyPrefix := string(resource.TypeTask) + ":"
	stale, err := collectEntries(
		ctx,
		gorp.WrapReader[string, ontology.Resource](tx),
		func(r ontology.Resource) bool {
			if r.ID.Type == resource.TypeTask {
				return true
			}
			return r.ID.Type == resource.TypeStatus &&
				strings.HasPrefix(r.ID.Key, statusKeyPrefix)
		},
	)
	if err != nil {
		return err
	}
	w := gorp.WrapWriter[string, ontology.Resource](tx)
	for _, r := range stale {
		var (
			oldKey    = r.GorpKey()
			legacyStr = strings.TrimPrefix(r.ID.Key, statusKeyPrefix)
		)
		legacy, ok := parseLegacyKey(legacyStr)
		if !ok {
			continue
		}
		key, ok := keys[legacy]
		if !ok {
			continue
		}
		if r.ID.Type == resource.TypeTask {
			r.ID = resourceID(key)
		} else {
			r.ID.Key = resourceID(key).String()
		}
		if err := w.Set(ctx, r); err != nil {
			return err
		}
		if err := w.Delete(ctx, oldKey); err != nil {
			return err
		}
	}
	return nil
}

// rewriteRelationships repoints every relationship endpoint referencing a task or a
// task status at the re-keyed resource, preserving direction and type.
func rewriteRelationships(ctx context.Context, tx gorp.Tx, keys map[v1.Key]Key) error {
	statusKeyPrefix := string(resource.TypeTask) + ":"
	rewriteEndpoint := func(id resource.ID) (resource.ID, bool) {
		var legacyStr string
		switch {
		case id.Type == resource.TypeTask:
			legacyStr = id.Key
		case id.Type == resource.TypeStatus &&
			strings.HasPrefix(id.Key, statusKeyPrefix):
			legacyStr = strings.TrimPrefix(id.Key, statusKeyPrefix)
		default:
			return id, false
		}
		legacy, ok := parseLegacyKey(legacyStr)
		if !ok {
			return id, false
		}
		key, ok := keys[legacy]
		if !ok {
			return id, false
		}
		if id.Type == resource.TypeTask {
			id.Key = key.String()
		} else {
			id.Key = resourceID(key).String()
		}
		return id, true
	}
	stale, err := collectEntries(
		ctx,
		gorp.WrapReader[string, ontology.Relationship](tx),
		func(rel ontology.Relationship) bool {
			_, from := rewriteEndpoint(rel.From)
			_, to := rewriteEndpoint(rel.To)
			return from || to
		},
	)
	if err != nil {
		return err
	}
	w := gorp.WrapWriter[string, ontology.Relationship](tx)
	for _, rel := range stale {
		oldKey := rel.GorpKey()
		rel.From, _ = rewriteEndpoint(rel.From)
		rel.To, _ = rewriteEndpoint(rel.To)
		if err := w.Set(ctx, rel); err != nil {
			return err
		}
		if err := w.Delete(ctx, oldKey); err != nil {
			return err
		}
	}
	return nil
}

// collectEntries drains a reader into a slice of the entries matching keep. Mutating a
// Gorp table while iterating it is unsafe, so callers gather first and write after.
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

// NameMigration names each stored task that has an empty name "<type> <key>", where
// <key> is the first eight characters of its key. A task with no type uses "Task".
var NameMigration = migrate.NewEntry(
	"v60_require_name",
	func(_ context.Context, t Task) (Task, error) {
		if t.Name != "" {
			return t, nil
		}
		prefix := t.Type
		if prefix == "" {
			prefix = "Task"
		}
		t.Name = prefix + " " + t.Key.String()[:8]
		return t, nil
	},
)
