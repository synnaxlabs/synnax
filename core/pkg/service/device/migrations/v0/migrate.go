// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"context"
	"fmt"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/query"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

type MigrationConfig struct {
	Status *status.Service
}

func Migration(cfg MigrationConfig) migrate.Migration {
	return gorp.NewMigration(
		"v53_device_migration",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			reader := gorp.WrapReader[Key, Device](tx)
			iter, err := reader.OpenIterator(gorp.IterOptions{})
			if err != nil {
				return err
			}
			defer func() {
				err = errors.Combine(err, iter.Close())
			}()

			var devices []Device
			for iter.First(); iter.Valid(); iter.Next() {
				d := iter.Value(ctx)
				if err = iter.Error(); err != nil {
					return err
				}
				devices = append(devices, *d)
			}
			if len(devices) == 0 {
				return nil
			}

			statusKeys := make([]string, len(devices))
			for i, d := range devices {
				statusKeys[i] = OntologyID(d.Key).String()
			}
			var existingStatuses []status.Status[StatusDetails]
			if err = status.NewRetrieve[StatusDetails](cfg.Status).
				WhereKeys(statusKeys...).
				Entries(&existingStatuses).
				Exec(ctx, nil); err != nil && !errors.Is(err, query.ErrNotFound) {
				return err
			}
			existingKeys := make(map[string]bool)
			for _, stat := range existingStatuses {
				existingKeys[stat.Key] = true
			}
			var missingStatuses []status.Status[StatusDetails]
			for _, d := range devices {
				key := OntologyID(d.Key).String()
				if !existingKeys[key] {
					missingStatuses = append(missingStatuses, status.Status[StatusDetails]{
						Key:     key,
						Name:    d.Name,
						Time:    telem.Now(),
						Variant: xstatus.VariantWarning,
						Message: fmt.Sprintf("%s state unknown", d.Name),
						Details: StatusDetails{Rack: d.Rack, Device: d.Key},
					})
				}
			}
			if len(missingStatuses) == 0 {
				return nil
			}
			ins.L.Info("creating unknown statuses for existing devices", zap.Int("count", len(missingStatuses)))
			return status.NewWriter[StatusDetails](cfg.Status, tx).SetMany(ctx, &missingStatuses)
		})
}
