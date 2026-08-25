// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package listener

import "github.com/spf13/cobra"

// FlagListen names the flag that selects the address, or list of listeners, the Core
// binds.
const FlagListen = "listen"

// AddFlags adds the listener flags to the given command.
func AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringP(
		FlagListen,
		"l",
		"localhost:9090",
		"The address to listen for client connections",
	)
}
