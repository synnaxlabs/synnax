// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package backup

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Flag names used by the backup command.
const (
	FlagNoData    = "no-data"
	FlagOverwrite = "overwrite"
)

var Cmd = &cobra.Command{
	Use:   "backup <src> <dst>",
	Short: "Copy a Synnax data directory to a backup location",
	Long: `Copy a Synnax data directory to a backup location.

The source must be the path to a Synnax data directory (the same path passed to
"synnax start --data"). The destination is the path where the directory will be
copied.

Use --no-data to skip channel data files so that only the cluster configuration
(channels, users, ranges, workspaces, and channel metadata) is copied. This is
useful for sharing or archiving a cluster's configuration without the bulk of
its telemetry.

The Synnax Core should be stopped before running backup. Copying a live data
directory may produce an inconsistent snapshot.`,
	Args: cobra.ExactArgs(2),
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		return viper.BindPFlags(cmd.Flags())
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return Backup(Config{
			Src:       args[0],
			Dst:       args[1],
			NoData:    viper.GetBool(FlagNoData),
			Overwrite: viper.GetBool(FlagOverwrite),
		})
	},
}

func init() {
	Cmd.Flags().Bool(
		FlagNoData,
		false,
		"Skip channel data files; copy only cluster configuration",
	)
	Cmd.Flags().Bool(
		FlagOverwrite,
		false,
		"Allow writing into a destination directory that already exists",
	)
}
