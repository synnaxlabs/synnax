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
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/versions"
	"github.com/synnaxlabs/x/errors"
)

func newMigrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate <resource>...",
		Short: "Scaffold the next version file for the named resources",
		Long: `Migrate writes v(N+1).oracle for each named resource: every current
type as an alias line to its defining version, pinned declarations redeclared,
imports carried forward. Edit the scaffold — convert the changed types to full
declarations, bump stale pins — then run oracle sync. To amend a version that
has not shipped, edit its file directly; there is no amend command.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runMigrate(cmd, args); err != nil {
				printError(err.Error())
				return err
			}
			return nil
		},
	}
}

func runMigrate(cmd *cobra.Command, args []string) error {
	printBanner()
	repoRoot, err := paths.RepoRoot()
	if err != nil {
		return errors.Wrap(err, "migrate must be run within a git repository")
	}
	if len(args) == 0 {
		return errors.New(
			"name the resources to bump, e.g. `oracle migrate channel`",
		)
	}
	chains, err := versions.Discover(repoRoot)
	if err != nil {
		return err
	}
	if len(chains) == 0 {
		return errors.New(
			"no version chains exist under schemas/; nothing to migrate",
		)
	}
	targets, err := resolveTargets(chains, args)
	if err != nil {
		return err
	}
	resolver := versions.NewResolver(
		chains, analyzer.NewStandardFileLoader(repoRoot),
	)
	ctx := cmd.Context()
	for _, chain := range targets {
		out, err := versions.Scaffold(ctx, resolver, chain)
		if err != nil {
			return err
		}
		next := chain.FilePath(chain.Current()+1) + ".oracle"
		target := filepath.Join(repoRoot, next)
		if _, err := os.Stat(target); err == nil {
			return errors.Newf("%s already exists", next)
		}
		if err := os.WriteFile(target, []byte(out), 0o644); err != nil {
			return errors.Wrapf(err, "failed to write %s", next)
		}
		printDim(fmt.Sprintf("  ✏️  %s", next))
	}
	printDim(
		"convert the changed types to full declarations, then run `oracle sync`",
	)
	return nil
}

// resolveTargets maps resource arguments to chains.
func resolveTargets(
	chains map[string]versions.Chain, args []string,
) ([]versions.Chain, error) {
	targets := make([]versions.Chain, 0, len(args))
	for _, arg := range args {
		var matches []versions.Chain
		for _, chain := range chains {
			if chain.Resource == arg || chain.LivePath() == arg {
				matches = append(matches, chain)
			}
		}
		switch len(matches) {
		case 1:
			targets = append(targets, matches[0])
		case 0:
			return nil, errors.Newf("no version chain matches %q", arg)
		default:
			return nil, errors.Newf(
				"%q is ambiguous; use a live path like %q",
				arg, matches[0].LivePath(),
			)
		}
	}
	return targets, nil
}
