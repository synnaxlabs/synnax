// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package instrumentation

import "github.com/spf13/cobra"

// Flag names used for configuring instrumentation.
const (
	FlagVerbose           = "verbose"
	FlagDebug             = "debug"
	FlagLogFilePath       = "log-file-path"
	FlagLogFileMaxSize    = "log-file-max-size"
	FlagLogFileMaxBackups = "log-file-max-backups"
	FlagLogFileMaxAge     = "log-file-max-age"
	FlagLogFileCompress   = "log-file-compress"
)

// BindFlags binds the instrumentation flags to the given command.
func BindFlags(cmd *cobra.Command) {
	cmd.Flags().BoolP(FlagVerbose, "v", false, "Enable verbose debugging")
	cmd.Flags().Bool(FlagDebug, false, "Enable debug logging")
	cmd.Flags().String(FlagLogFilePath, "./synnax-logs/synnax.log", "Log file path")
	cmd.Flags().Int(FlagLogFileMaxSize, 50, "Maximum size of log file in MB")
	cmd.Flags().Int(FlagLogFileMaxBackups, 5, "Maximum number of log files to keep")
	cmd.Flags().Int(FlagLogFileMaxAge, 30, "Maximum age of log files in days")
	cmd.Flags().Bool(FlagLogFileCompress, false, "Compress log files")
}
