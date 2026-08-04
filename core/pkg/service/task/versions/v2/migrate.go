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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	rack "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v2"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	v1 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v1"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/zyn"
)

// LegacyKeyKVPrefix is the KV key prefix under which the re-key migration stages the
// mapping from a task's legacy uint64 key to its new UUID. The remainder of the key
// is the legacy key's decimal string; the value is the UUID string. The panel
// migration consumes these entries to re-key task references in converted layouts.
const LegacyKeyKVPrefix = "sy_task_legacy_key/"

// LegacyKeyKVKey returns the staging KV key holding the UUID for the task with the
// given legacy key.
func LegacyKeyKVKey(legacy v1.Key) []byte {
	return []byte(LegacyKeyKVPrefix + strconv.FormatUint(uint64(legacy), 10))
}

// Migration re-keys stored tasks from legacy composite uint64 keys to UUIDs.
var Migration = gorp.NewMigration("v56_task_uuid_key", migrateKeysToUUID)

// ontologyID returns the ontology identifier for the task with the given key.
func ontologyID(key Key) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeTask, Key: key.String()}
}

// resourceSchema is a frozen copy of the task ontology resource schema as of this
// version, used to rebuild re-keyed task resources.
var resourceSchema = zyn.Object(map[string]zyn.Schema{
	"key":      zyn.UUID(),
	"name":     zyn.String(),
	"type":     zyn.String(),
	"snapshot": zyn.Bool(),
})

// hashConfig is a frozen copy of the config hash as of this version: xxhash64 of the
// JSON encoding as 16 lowercase hex characters. encoding/json sorts map keys, so
// equal configs hash equally.
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

// migrateKeysToUUID re-keys every task from the legacy composite uint64 key to a
// freshly minted UUID. The rack encoded in the legacy key's high 32 bits becomes the
// task's rack field. Task rows, task statuses, ontology resources, and ontology
// relationships are all rewritten; each legacy-to-UUID pair is staged in KV under
// LegacyKeyKVPrefix for the panel migration. The whole re-key runs in one migration
// transaction, so it commits atomically.
func migrateKeysToUUID(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
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
	if err := rewriteStatuses(ctx, tx, keys, byKey); err != nil {
		return err
	}
	if err := rewriteResources(ctx, tx, keys, byKey); err != nil {
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

// legacyKeyFromOntology parses a legacy uint64 task key out of an ontology key
// string, returning false when the string is not a legacy key.
func legacyKeyFromOntology(key string) (v1.Key, bool) {
	n, err := strconv.ParseUint(key, 10, 64)
	if err != nil {
		return 0, false
	}
	return v1.Key(n), true
}

// rewriteStatuses re-keys every task status row from "task:<legacy>" to
// "task:<uuid>" and rewrites the task and rack references inside its details. A
// status row whose task has no live record (already deleted pre-migration) is
// deleted rather than left stuck under its legacy key forever.
func rewriteStatuses(
	ctx context.Context,
	tx gorp.Tx,
	keys map[v1.Key]Key,
	byKey map[Key]Task,
) error {
	statusKeyPrefix := string(ontology.ResourceTypeTask) + ":"
	var stale []status.Status[v1.StatusDetails]
	if err := gorp.NewRetrieve[string, status.Status[v1.StatusDetails]]().
		WherePrefix([]byte(statusKeyPrefix)).
		Entries(&stale).
		Exec(ctx, tx); err != nil {
		return err
	}
	w := gorp.WrapWriter[string, Status](tx)
	del := gorp.WrapWriter[string, status.Status[v1.StatusDetails]](tx)
	for _, s := range stale {
		oldKey := s.Key
		legacy, ok := legacyKeyFromOntology(strings.TrimPrefix(s.Key, statusKeyPrefix))
		if !ok {
			continue
		}
		key, ok := keys[legacy]
		if !ok {
			if err := del.Delete(ctx, oldKey); err != nil {
				return err
			}
			continue
		}
		if err := w.Set(ctx, Status{
			Key:         ontologyID(key).String(),
			Name:        s.Name,
			Variant:     s.Variant,
			Message:     s.Message,
			Description: s.Description,
			Time:        s.Time,
			Labels:      s.Labels,
			Details: StatusDetails{
				Task:    key,
				Running: s.Details.Running,
				Cmd:     s.Details.Cmd,
				Rack:    byKey[key].Rack,
				Data:    s.Details.Data,
			},
		}); err != nil {
			return err
		}
		if err := del.Delete(ctx, oldKey); err != nil {
			return err
		}
	}
	return nil
}

// rewriteResources re-keys every task ontology resource to its UUID and every task
// status resource to the re-keyed status key. Task resource data is rebuilt from the
// migrated record so it reflects the new key and rack.
func rewriteResources(
	ctx context.Context,
	tx gorp.Tx,
	keys map[v1.Key]Key,
	byKey map[Key]Task,
) error {
	statusKeyPrefix := string(ontology.ResourceTypeTask) + ":"
	stale, err := collectEntries(
		ctx,
		gorp.WrapReader[string, ontology.Resource](tx),
		func(r ontology.Resource) bool {
			if r.ID.Type == ontology.ResourceTypeTask {
				return true
			}
			return r.ID.Type == ontology.ResourceTypeStatus &&
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
		legacy, ok := legacyKeyFromOntology(legacyStr)
		if !ok {
			continue
		}
		key, ok := keys[legacy]
		if !ok {
			continue
		}
		if r.ID.Type == ontology.ResourceTypeTask {
			t := byKey[key]
			r = ontology.NewResource(resourceSchema, ontologyID(key), t.Name, t)
		} else {
			r.ID.Key = ontologyID(key).String()
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
	statusKeyPrefix := string(ontology.ResourceTypeTask) + ":"
	rewriteEndpoint := func(id ontology.ID) (ontology.ID, bool) {
		var legacyStr string
		switch {
		case id.Type == ontology.ResourceTypeTask:
			legacyStr = id.Key
		case id.Type == ontology.ResourceTypeStatus &&
			strings.HasPrefix(id.Key, statusKeyPrefix):
			legacyStr = strings.TrimPrefix(id.Key, statusKeyPrefix)
		default:
			return id, false
		}
		legacy, ok := legacyKeyFromOntology(legacyStr)
		if !ok {
			return id, false
		}
		key, ok := keys[legacy]
		if !ok {
			return id, false
		}
		if id.Type == ontology.ResourceTypeTask {
			id.Key = key.String()
		} else {
			id.Key = ontologyID(key).String()
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
