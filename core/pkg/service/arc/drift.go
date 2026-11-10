// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"maps"
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc/drift"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
)

// RetrieveDrift returns the drift record in snap of each automation in keys whose task
// has one, by automation. A task has a record from its first write by the Core until it
// is deleted. While the Driver reports running a task's config, its stored hashes read
// as deployed.
func (s *Service) RetrieveDrift(
	ctx context.Context,
	snap gorp.Snapshot,
	keys ...Key,
) (map[Key]drift.Record, error) {
	records := make(map[Key]drift.Record, len(keys))
	for _, key := range keys {
		var tasks []task.Task
		if err := s.cfg.Task.NewRetrieve().
			Where(task.And(s.cfg.MatchTasks(key), task.MatchSnapshot(false))).
			Entries(&tasks).
			Exec(ctx, snap); err != nil {
			return nil, err
		}
		if len(tasks) == 0 {
			continue
		}
		var rec drift.Record
		err := s.drifts.NewRetrieve().
			Where(gorp.MatchKeys[drift.Key, drift.Record](tasks[0].Key)).
			Entry(&rec).
			Exec(ctx, snap)
		if errors.Is(err, query.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		deployed, err := runs(ctx, s.cfg.Status, snap, tasks[0])
		if err != nil {
			return nil, err
		}
		if deployed {
			rec.Deployed = rec.Stored
		}
		records[key] = rec
	}
	return records, nil
}

// Changes names each difference between the content rec's automation holds and the
// content its task last ran. The automation comes first, then modules by path. A record
// with no deployed content names nothing.
func Changes(rec drift.Record) []Change {
	if rec.Deployed.Text == "" {
		return nil
	}
	var changes []Change
	if rec.Stored.Text != rec.Deployed.Text {
		changes = append(changes, Change{Type: TypeAutomation, Kind: ChangeKindChanged})
	}
	paths := set.New[string]()
	for path := range rec.Stored.Modules {
		paths.Add(path)
	}
	for path := range rec.Deployed.Modules {
		paths.Add(path)
	}
	for _, path := range slices.Sorted(maps.Keys(paths)) {
		stored, inStored := rec.Stored.Modules[path]
		deployed, inDeployed := rec.Deployed.Modules[path]
		var kind ChangeKind
		switch {
		case !inDeployed:
			kind = ChangeKindAdded
		case !inStored:
			kind = ChangeKindRemoved
		case stored != deployed:
			kind = ChangeKindChanged
		default:
			continue
		}
		changes = append(changes, Change{Type: TypeModule, Path: path, Kind: kind})
	}
	return changes
}

// record returns the drift record of the task with key, or an empty record for the key
// when none is stored.
func (w Writer) record(ctx context.Context, key task.Key) (drift.Record, error) {
	var rec drift.Record
	err := w.drifts.NewRetrieve().
		Where(gorp.MatchKeys[drift.Key, drift.Record](key)).
		Entry(&rec).
		Exec(ctx, w.tx)
	if errors.Is(err, query.ErrNotFound) {
		return drift.Record{Task: key}, nil
	}
	return rec, err
}

// runs reports whether the Driver reports running the config tsk holds in snap.
func runs(
	ctx context.Context,
	statuses *status.Service,
	snap gorp.Snapshot,
	tsk task.Task,
) (bool, error) {
	stat, ok, err := taskStatus(ctx, statuses, snap, tsk.Key)
	return ok && stat.Details.Running && stat.Details.ConfigHash == tsk.ConfigHash, err
}

// writeRecord stores hashes as the content of rec's automation after a write of its
// task. tsk is the task as it was before the write. When the Driver reports running
// that task, the hashes it held become the deployed ones first.
func (w Writer) writeRecord(
	ctx context.Context,
	tsk task.Task,
	rec drift.Record,
	hashes drift.Hashes,
) error {
	deployed, err := runs(ctx, w.status, w.tx, tsk)
	if err != nil {
		return err
	}
	if deployed {
		rec.Deployed = rec.Stored
	}
	rec.Stored = hashes
	return w.drifts.NewCreate().Entry(&rec).Exec(ctx, w.tx)
}
