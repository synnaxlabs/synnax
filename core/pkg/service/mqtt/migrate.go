// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	v3 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
)

// ICDMigration returns the migration that moves the staged plain targets of each write
// config into an ICD named after its task.
func (s *Service) ICDMigration(names config.TaskNames) gorp.Migration {
	return config.StagedMigration(
		"v63_mqtt_write_resolve",
		v3.StagedTargetsKVPrefix,
		names,
		func(
			ctx context.Context,
			tx gorp.Tx,
			ins alamos.Instrumentation,
			task string,
			cfg *WriteConfig,
			staged v3.InlineWrite,
		) error {
			cfg.Inline = &staged
			ic, _, err := v3.ResolveWriteConfig(
				ctx, tx, s.writes.channels, task, cfg, v3.NoteLift(ins, cfg.Key),
			)
			if err != nil {
				return err
			}
			return s.writes.icds.NewWriter(tx).Create(ctx, &ic)
		},
	)
}
