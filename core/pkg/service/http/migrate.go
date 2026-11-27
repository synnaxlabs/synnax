// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
)

// ICDMigration returns the migration that moves the staged fields of each read config
// into an ICD named after its task.
func (s *Service) ICDMigration(names config.TaskNames) gorp.Migration {
	return config.StagedMigration(
		"v63_http_read_icd",
		v5.StagedFieldsKVPrefix,
		names,
		func(
			ctx context.Context,
			tx gorp.Tx,
			ins alamos.Instrumentation,
			task string,
			cfg *ReadConfig,
			staged v5.InlineRead,
		) error {
			cfg.Inline = &staged
			ic, _, err := v5.ResolveReadConfig(
				ctx, tx, s.resolver.channels, task, cfg, v5.NoteLift(ins, cfg.Key),
			)
			if err != nil {
				return err
			}
			return s.resolver.icds.NewWriter(tx).Create(ctx, &ic)
		},
	)
}

// WriteICDMigration returns the migration that moves the staged channel and fields of
// each write config into an ICD named after its task.
func (s *Service) WriteICDMigration(names config.TaskNames) gorp.Migration {
	return config.StagedMigration(
		"v63_http_write_icd",
		v5.StagedWriteFieldsKVPrefix,
		names,
		func(
			ctx context.Context,
			tx gorp.Tx,
			ins alamos.Instrumentation,
			task string,
			cfg *WriteConfig,
			staged v5.InlineWrite,
		) error {
			cfg.Inline = &staged
			ic, _, err := v5.ResolveWriteConfig(
				ctx, tx, s.writes.channels, task, cfg, v5.NoteWriteLift(ins, cfg.Key),
			)
			if err != nil {
				return err
			}
			return s.writes.icds.NewWriter(tx).Create(ctx, &ic)
		},
	)
}
