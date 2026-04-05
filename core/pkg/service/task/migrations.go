// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"context"
	"fmt"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

type statusBackfillMigration struct{ cfg ServiceConfig }

var _ gorp.Migration = (*statusBackfillMigration)(nil)

func (m *statusBackfillMigration) Name() string { return "task_status_backfill" }

func (m *statusBackfillMigration) Run(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) (err error) {
	reader := gorp.WrapReader[Key, Task](tx)
	iter, err := reader.OpenIterator(gorp.IterOptions{})
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()

	var tasks []Task
	for iter.First(); iter.Valid(); iter.Next() {
		t := iter.Value(ctx)
		if err = iter.Error(); err != nil {
			return err
		}
		tasks = append(tasks, *t)
	}
	if len(tasks) == 0 {
		return nil
	}

	statusKeys := make([]string, len(tasks))
	for i, t := range tasks {
		statusKeys[i] = OntologyID(t.Key).String()
	}
	var existingStatuses []Status
	if err = status.NewRetrieve[StatusDetails](m.cfg.Status).
		WhereKeys(statusKeys...).
		Entries(&existingStatuses).
		Exec(ctx, nil); err != nil && !errors.Is(err, query.ErrNotFound) {
		return err
	}
	existingKeys := make(set.Set[string])
	for _, stat := range existingStatuses {
		existingKeys.Add(stat.Key)
	}
	var missingStatuses []Status
	for _, t := range tasks {
		key := OntologyID(t.Key).String()
		if !existingKeys.Contains(key) {
			missingStatuses = append(missingStatuses, Status{
				Key:     key,
				Name:    t.Name,
				Time:    telem.Now(),
				Variant: xstatus.VariantWarning,
				Message: fmt.Sprintf("%s status unknown", t.Name),
				Details: StatusDetails{Task: t.Key},
			})
		}
	}
	if len(missingStatuses) == 0 {
		return nil
	}
	ins.L.Info("creating unknown statuses for existing tasks", zap.Int("count", len(missingStatuses)))
	return status.NewWriter[StatusDetails](m.cfg.Status, nil).SetMany(ctx, &missingStatuses)
}
