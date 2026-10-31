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
	"github.com/spf13/cobra"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/surface"
	"github.com/synnaxlabs/x/errors"
)

func newSurfaceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "surface",
		Short: "Print the types and fields of every schema",
		Long: `Print the types and fields of every schema as sorted text.

Docs, annotations, and formatting are left out. Compare the output of two revisions
to see if a pull request changes a stored shape or a wire type.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return runSurface(cmd) },
	}
}

func runSurface(cmd *cobra.Command) error {
	repoRoot, err := paths.RepoRoot()
	if err != nil {
		return errors.Wrap(err, "surface must be run within a git repository")
	}
	schemas, err := pipeline.DiscoverSchemas(repoRoot)
	if err != nil {
		return err
	}
	res, err := pipeline.Run(
		cmd.Context(),
		pipeline.Options{RepoRoot: repoRoot, Schemas: schemas},
	)
	if err != nil {
		return err
	}
	if res.Resolutions == nil {
		return errors.Wrap(res.Diagnostics.Flat(), "analysis failed")
	}
	return surface.Write(cmd.OutOrStdout(), res.Resolutions)
}
