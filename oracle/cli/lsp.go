// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cli

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/synnaxlabs/oracle/lsp"
)

var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Start the Oracle language server",
	Long: `Starts the Oracle Language Server Protocol (LSP) server.

The LSP server communicates over stdin/stdout using JSON-RPC. It provides:
- Syntax error diagnostics
- Code completion for keywords, types, and domains
- Hover information for built-in types and keywords
- Semantic token highlighting

This command is typically invoked by an IDE or editor extension.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		server := lsp.New()
		return server.Serve(context.Background(), os.Stdin, os.Stdout)
	},
}

func init() {
	// Accept --stdio flag (used by vscode-languageclient) but ignore it
	// since we always use stdio transport
	lspCmd.Flags().Bool("stdio", true, "Use stdio transport (default, always enabled)")
	rootCmd.AddCommand(lspCmd)
}
