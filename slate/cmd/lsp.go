// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/slate/lsp"
	"github.com/synnaxlabs/slate/lsp/transport"
	xos "github.com/synnaxlabs/x/os"
)

var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Start the Slate Language Server",
	Long:  `Start the Slate Language Server Protocol (LSP) server`,
	RunE: func(cmd *cobra.Command, args []string) error {
		server, err := lsp.New()
		if err != nil {
			return err
		}
		
		// Use context.Background() since we're running as a long-lived server
		ctx := context.Background()
		return transport.ServeJSONRPC(ctx, server, xos.Stdio)
	},
}
