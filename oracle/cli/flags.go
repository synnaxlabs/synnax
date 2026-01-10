// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// bindFlags binds cobra flags to viper for a command.
// This allows flags to be set via environment variables.
func bindFlags(cmd *cobra.Command) {
	if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
		fmt.Printf("warning: failed to bind persistent flags: %v\n", err)
	}
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		fmt.Printf("warning: failed to bind flags: %v\n", err)
	}
}
