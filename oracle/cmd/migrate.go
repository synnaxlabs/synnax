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
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/plugin/go/freeze"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	"github.com/synnaxlabs/x/errors"
)

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate [resource...]",
		Short: "Mint or amend version files and generate migration code",
		Long: `With explicit version chains, migrate mints the next version file for the
named resources (or every drifted resource when none are named) from the live
persisted shapes, then syncs. --amend rewrites the current version file in
place instead — for versions that have never shipped in a release.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runMigrate(cmd, args); err != nil {
				printError(err.Error())
				return err
			}
			return nil
		},
	}
	cmd.Flags().Bool("amend", false,
		"rewrite the named resources' current version files in place")
	return cmd
}

// runChainMigrate mints or amends version files for the targeted chains, then
// syncs.
func runChainMigrate(
	cmd *cobra.Command,
	repoRoot string,
	chains map[string]versions.Chain,
	args []string,
	amend bool,
) error {
	ctx := cmd.Context()
	loader := analyzer.NewStandardFileLoader(repoRoot)
	resolver := versions.NewResolver(chains, loader)

	normalizedFiles, err := pipeline.DiscoverSchemas(repoRoot)
	if err != nil {
		return err
	}
	live, diag := analyzer.Analyze(ctx, normalizedFiles, loader)
	if diag != nil {
		printDiagnostics(diag.String())
		if !diag.Ok() {
			return errors.New("schema analysis failed")
		}
	}
	if err := resolver.Annotate(ctx, live); err != nil {
		return err
	}

	targets, err := resolveTargets(
		ctx, repoRoot, resolver, live, chains, args, amend,
	)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		printDim("no drifted resources; nothing to migrate")
		return nil
	}

	for _, chain := range targets {
		current := chain.Current()
		f, err := resolver.File(ctx, chain.LivePath(), current)
		if err != nil {
			return err
		}
		pins, docs, pinned := freeze.FileInput(f)
		in := freeze.Input{
			Live:     live,
			Resolver: resolver,
			Chain:    chain,
			Pinned:   pinned,
		}
		if amend {
			in.N, in.Pins, in.Docs = current, pins, docs
		} else {
			in.N = current + 1
		}
		out, err := freeze.Canonical(ctx, in)
		if err != nil {
			return err
		}
		target := filepath.Join(repoRoot, chain.FilePath(in.N)+".oracle")
		if err := writeFileIfChanged(target, []byte(out)); err != nil {
			return errors.Wrapf(err, "failed to write %s", target)
		}
		printDim(fmt.Sprintf("  ✏️  %s", chain.FilePath(in.N)+".oracle"))
	}
	printDim("running sync...")
	return runSync(cmd)
}

// resolveTargets maps resource arguments to chains; with no arguments it
// selects every chain whose live schema drifts from its current file.
func resolveTargets(
	ctx context.Context,
	repoRoot string,
	resolver *versions.Resolver,
	live *resolution.Table,
	chains map[string]versions.Chain,
	args []string,
	amend bool,
) ([]versions.Chain, error) {
	if len(args) > 0 {
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
	if amend {
		return nil, errors.New("--amend requires naming the resources to amend")
	}
	var drifted []versions.Chain
	for _, livePath := range slices.Sorted(maps.Keys(chains)) {
		chain := chains[livePath]
		f, err := resolver.File(ctx, livePath, chain.Current())
		if err != nil {
			return nil, err
		}
		pins, docs, pinned := freeze.FileInput(f)
		canonical, err := freeze.Canonical(ctx, freeze.Input{
			Live:     live,
			Resolver: resolver,
			Chain:    chain,
			N:        chain.Current(),
			Pins:     pins,
			Docs:     docs,
			Pinned:   pinned,
		})
		if err != nil {
			return nil, err
		}
		onDisk, err := os.ReadFile(
			filepath.Join(repoRoot, chain.FilePath(chain.Current())+".oracle"),
		)
		if err != nil {
			return nil, err
		}
		if string(onDisk) != canonical {
			drifted = append(drifted, chain)
		}
	}
	return drifted, nil
}

func runMigrate(cmd *cobra.Command, args []string) error {
	printBanner()
	repoRoot, err := paths.RepoRoot()
	if err != nil {
		return errors.Wrap(err, "migrate must be run within a git repository")
	}

	amend, err := cmd.Flags().GetBool("amend")
	if err != nil {
		return err
	}
	chains, err := versions.Discover(repoRoot)
	if err != nil {
		return err
	}
	if len(chains) > 0 {
		return runChainMigrate(cmd, repoRoot, chains, args, amend)
	}
	return errors.New(
		"no version chains exist under schemas/; nothing to migrate",
	)
}

func writeFileIfChanged(path string, content []byte) error {
	existing, err := os.ReadFile(path)
	if err == nil && string(existing) == string(content) {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}
