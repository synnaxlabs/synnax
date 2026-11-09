// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package update

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/cmd/instrumentation"
	"github.com/synnaxlabs/synnax/cmd/service"
	"github.com/synnaxlabs/synnax/cmd/start"
	"github.com/synnaxlabs/synnax/pkg/storage"
)

func addFlags(cmd *cobra.Command) {
	instrumentation.AddFlags(cmd)
	start.AddDataFlag(cmd)
}

// openBackups opens the backups of the data directory that the flags name, or of the
// Core service's data directory when they name none. It also returns the directory.
func openBackups(ins alamos.Instrumentation) (*storage.Backups, string, error) {
	dir, err := dataDir()
	if err != nil {
		return nil, "", err
	}
	backups, err := storage.NewBackups(
		storage.BackupsConfig{Instrumentation: ins, Dirname: dir},
	)
	return backups, dir, err
}

func dataDir() (string, error) {
	if !viper.IsSet(start.FlagData) {
		inst, installed, err := service.Lookup()
		if err != nil {
			return "", err
		}
		if installed {
			return filepath.Clean(inst.Data), nil
		}
	}
	return filepath.Abs(viper.GetString(start.FlagData))
}
