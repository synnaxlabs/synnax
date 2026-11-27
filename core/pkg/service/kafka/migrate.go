// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	icdv1 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	v1 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
)

// ICDMigration returns the migration that moves the staged records of each read config
// into an ICD named after its task.
func (s *Service) ICDMigration(names config.TaskNames) gorp.Migration {
	return config.StagedMigration(
		"v63_kafka_read_icd",
		v1.StagedRecordsKVPrefix,
		names,
		func(
			ctx context.Context,
			tx gorp.Tx,
			ins alamos.Instrumentation,
			task string,
			cfg *ReadConfig,
			staged v1.InlineRead,
		) error {
			cfg.Inline = &staged
			old, _, err := v1.ResolveReadConfig(
				ctx, tx, s.resolver.channels, task, cfg, v1.NoteLift(ins, cfg.Key),
			)
			if err != nil {
				return err
			}
			ic, err := icdv1.MigrateICD(ctx, old)
			if err != nil {
				return err
			}
			return s.resolver.icds.NewWriter(tx).Create(ctx, &ic)
		},
	)
}

// WriteICDMigration returns the migration that moves the staged channels of each write
// config into targets of an ICD named after its task.
func (s *Service) WriteICDMigration(names config.TaskNames) gorp.Migration {
	return config.StagedMigration(
		"v63_kafka_write_icd",
		v1.StagedChannelsKVPrefix,
		names,
		func(
			ctx context.Context,
			tx gorp.Tx,
			ins alamos.Instrumentation,
			task string,
			cfg *WriteConfig,
			staged v1.InlineWrite,
		) error {
			cfg.Inline = &staged
			ic, _, err := v1.ResolveWriteConfig(
				ctx, tx, s.writes.channels, task, cfg, v1.NoteLift(ins, cfg.Key),
			)
			if err != nil {
				return err
			}
			return s.writes.icds.NewWriter(tx).Create(ctx, &ic)
		},
	)
}
