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
	"log"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/synnaxlabs/oracle/lsp"
	xos "github.com/synnaxlabs/x/os"
)

func newLSPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lsp",
		Short: "Start the language server",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := log.New(cmd.ErrOrStderr(), "oracle lsp: ", log.LstdFlags)
			return lsp.New(logger).Serve(cmd.Context(), xos.StdIO)
		},
	}
	// LSP clients conventionally launch servers with --stdio; accept and
	// discard it so cobra does not reject the flag.
	cmd.Flags().Bool("stdio", true, "")
	lo.Must0(cmd.Flags().MarkHidden("stdio"))
	return cmd
}
