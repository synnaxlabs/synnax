// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cli

// Generate-specific flag constants
const (
	pluginsFlag = "plugin"
)

func configureGenerateFlags() {
	generateCmd.Flags().StringSliceP(
		pluginsFlag,
		"p",
		nil,
		"Plugins to run (e.g., 'go', 'zod', 'py'). If not specified, all available plugins are run.",
	)
}
