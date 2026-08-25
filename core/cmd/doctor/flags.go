// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package doctor

import (
	"github.com/spf13/cobra"
	"github.com/synnaxlabs/synnax/cmd/start"
)

// Flag names used for inspecting stored data. Every flag excludes work; a run with no
// flags performs the full inspection.
const (
	FlagChannels = "channels"
	FlagSkipDeep = "skip-deep"
	FlagSkipKV   = "skip-kv"
	FlagSkipTS   = "skip-ts"
	FlagJSON     = "json"
	FlagVerbose  = "verbose"
)

// AddFlags adds the doctor flags to the given command.
func AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringP(
		start.FlagData,
		"d",
		"synnax-data",
		"Directory holding the data to inspect",
	)
	cmd.Flags().UintSlice(
		FlagChannels,
		nil,
		"Keys of the channels to inspect. Defaults to every channel",
	)
	cmd.Flags().Bool(FlagSkipDeep, false, "Skip the checks that read sample bytes")
	cmd.Flags().Bool(FlagSkipKV, false, "Skip the key-value checks")
	cmd.Flags().Bool(FlagSkipTS, false, "Skip the time-series checks")
	cmd.Flags().Bool(FlagJSON, false, "Write the report as JSON instead of text")
	cmd.Flags().BoolP(
		FlagVerbose,
		"v",
		false,
		"Include notes and the per-channel statistics table",
	)
}
