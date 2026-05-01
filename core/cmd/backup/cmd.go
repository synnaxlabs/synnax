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

// Flag names used by the backup command. FlagData intentionally matches the flag
// used by "synnax start", so that the same value can be supplied via --data, the
// SYNNAX_DATA environment variable, or the YAML config file the user already uses
// to launch a Synnax Core.
const (
	FlagData      = "data"
	FlagNoData    = "no-data"
	FlagOverwrite = "overwrite"
)

var Cmd = &cobra.Command{
	Use:   "backup <dst>",
	Short: "Copy a Synnax data directory to a backup location",
	Long: `Copy a Synnax data directory to a backup location.

The source defaults to the Synnax data directory configured via the --data flag,
the SYNNAX_DATA environment variable, or the YAML config file (the same source
"synnax start" reads). Pass --data to override.

Use --no-data to skip channel data files so that only the cluster configuration
(channels, users, ranges, workspaces, and channel metadata) is copied. This is
useful for sharing or archiving a cluster's configuration without the bulk of
its telemetry.

The Synnax Core should be stopped before running backup. Copying a live data
directory may produce an inconsistent snapshot.`,
	Example: `  synnax backup ./backup
  synnax backup --data /mnt/ssd1 /mnt/backup
  synnax backup --no-data ./config-only`,
	Args: cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		return viper.BindPFlags(cmd.Flags())
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return Backup(Config{
			Src:       viper.GetString(FlagData),
			Dst:       args[0],
			NoData:    viper.GetBool(FlagNoData),
			Overwrite: viper.GetBool(FlagOverwrite),
		})
	},
}

func init() {
	Cmd.Flags().StringP(
		FlagData,
		"d",
		"synnax-data",
		"Source Synnax data directory (also read from SYNNAX_DATA / config file)",
	)
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
