// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack

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

type embeddedRackRenameMigration struct{ cfg ServiceConfig }

var _ gorp.Migration = (*embeddedRackRenameMigration)(nil)

func (m *embeddedRackRenameMigration) Name() string { return "embedded_rack_rename_v1_to_v2" }

func (m *embeddedRackRenameMigration) Run(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) (err error) {
	hostKey := m.cfg.HostProvider.HostKey()
	v1Name := fmt.Sprintf("sy_node_%s_rack", hostKey)
	v2Name := fmt.Sprintf("Node %s Embedded Driver", hostKey)

	reader := gorp.WrapReader[Key, Rack](tx)
	writer := gorp.WrapWriter[Key, Rack](tx)

	iter, err := reader.OpenIterator(gorp.IterOptions{})
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()

	for iter.First(); iter.Valid(); iter.Next() {
		r := iter.Value(ctx)
		if err = iter.Error(); err != nil {
			return err
		}
		if r.Name == v1Name {
			r.Name = v2Name
			r.Embedded = true
			if err = writer.Set(ctx, *r); err != nil {
				return err
			}
			ins.L.Info("renamed embedded rack from v1 to v2",
				zap.String("old", v1Name),
				zap.String("new", v2Name),
			)
			return nil
		}
	}
	return err
}

type statusBackfillMigration struct{ cfg ServiceConfig }

var _ gorp.Migration = (*statusBackfillMigration)(nil)

func (m *statusBackfillMigration) Name() string { return "rack_status_backfill" }

func (m *statusBackfillMigration) Run(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) (err error) {
	reader := gorp.WrapReader[Key, Rack](tx)
	iter, err := reader.OpenIterator(gorp.IterOptions{})
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()

	var racks []Rack
	for iter.First(); iter.Valid(); iter.Next() {
		r := iter.Value(ctx)
		if err = iter.Error(); err != nil {
			return err
		}
		racks = append(racks, *r)
	}
	if len(racks) == 0 {
		return nil
	}

	statusKeys := make([]string, len(racks))
	for i, r := range racks {
		statusKeys[i] = OntologyID(r.Key).String()
	}
	var existingStatuses []status.Status[StatusDetails]
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
	var missingStatuses []status.Status[StatusDetails]
	for _, r := range racks {
		key := OntologyID(r.Key).String()
		if !existingKeys.Contains(key) {
			missingStatuses = append(missingStatuses, status.Status[StatusDetails]{
				Key:     key,
				Name:    r.Name,
				Time:    telem.Now(),
				Variant: xstatus.VariantWarning,
				Message: "Status unknown",
				Details: StatusDetails{Rack: r.Key},
			})
		}
	}
	if len(missingStatuses) == 0 {
		return nil
	}
	ins.L.Info("creating unknown statuses for existing racks", zap.Int("count", len(missingStatuses)))
	return status.NewWriter[StatusDetails](m.cfg.Status, nil).SetMany(ctx, &missingStatuses)
}
