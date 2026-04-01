// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/synnaxlabs/oracle/lsp"
	xos "github.com/synnaxlabs/x/os"
)

func newLSPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lsp",
		Short: "Start the language server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return lsp.New().Serve(context.Background(), xos.StdIO)
		},
	}
	cmd.Flags().Bool("stdio", true, "")
	_ = cmd.Flags().MarkHidden("stdio")
	return cmd
}
