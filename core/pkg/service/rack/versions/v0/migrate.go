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
	"github.com/synnaxlabs/synnax/pkg/service/cluster"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// MigrationConfig is the configuration for newMigration.
type MigrationConfig struct {
	// HostProvider resolves the host node, used to rename the embedded rack.
	HostProvider cluster.HostProvider
	// Status is the status service used to backfill unknown statuses for racks missing
	// them.
	Status *status.Service
}

// newMigration returns the v0 migration, which renames the host's embedded rack and
// backfills an unknown status for every rack missing one.
func newMigration(cfg MigrationConfig) migrate.Migration {
	return gorp.NewMigration(
		"v0.embedded_rack_rename_status_backfill",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			if err := renameEmbeddedRack(ctx, tx, ins, cfg); err != nil {
				return err
			}
			return backfillStatuses(ctx, tx, ins, cfg)
		},
	)
}

func renameEmbeddedRack(
	ctx context.Context,
	tx gorp.Tx,
	ins alamos.Instrumentation,
	cfg MigrationConfig,
) error {
	hostKey := cfg.HostProvider.HostKey()
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

func backfillStatuses(
	ctx context.Context,
	tx gorp.Tx,
	ins alamos.Instrumentation,
	cfg MigrationConfig,
) error {
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
		statusKeys[i] = r.OntologyID().String()
	}
	var existingStatuses []status.Status[StatusDetails]
	if err = status.NewRetrieve[StatusDetails](cfg.Status).
		Where(status.MatchKeys[StatusDetails](statusKeys...)).
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
		key := r.OntologyID().String()
		if !existingKeys.Contains(key) {
			missingStatuses = append(missingStatuses, status.Status[StatusDetails]{
				Key:     key,
				Name:    r.Name,
				Time:    telem.Now(),
				Variant: status.VariantWarning,
				Message: "Status unknown",
				Details: StatusDetails{Rack: r.Key},
			})
		}
	}
	if len(missingStatuses) == 0 {
		return nil
	}
	ins.L.Info("creating unknown statuses for existing racks", zap.Int("count", len(missingStatuses)))
	return status.NewWriter[StatusDetails](cfg.Status, tx).SetMany(ctx, &missingStatuses)
}

// codecMigration re-encodes stored racks from MessagePack to Orc.
var codecMigration = gorp.CodecMigration[Key, Rack]("msgpack_to_orc")

// NewMigrations returns the ordered set of migrations introduced at this version.
func NewMigrations(cfg MigrationConfig) []migrate.Migration {
	return []migrate.Migration{newMigration(cfg), codecMigration}
}
