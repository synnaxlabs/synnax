// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package instrumentation

import (
	_ "embed"

	"github.com/spf13/cobra"
	"github.com/synnaxlabs/synnax/cmd/flagdef"
)

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

//go:embed flags.json
var flagsJSON []byte

// FlagDefs are the parsed flag definitions for the instrumentation flag set.
var FlagDefs = flagdef.MustParse(flagsJSON)

// AddFlags adds the instrumentation flags to the given command.
func AddFlags(cmd *cobra.Command) {
	flagdef.MustRegister(cmd, FlagDefs)
}
