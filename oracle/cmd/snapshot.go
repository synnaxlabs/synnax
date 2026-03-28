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
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/snapshot"
	"github.com/synnaxlabs/x/errors"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Snapshot all schemas at the current core version for migration diffing",
	Long: `Creates a hermetic copy of all .oracle schema files at the current core
version. Run this once per release (typically right after bumping the version).
The snapshot serves as the baseline for oracle migrate on feature branches.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runSnapshot(cmd); err != nil {
			printError(err.Error())
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(snapshotCmd)
}

func runSnapshot(_ *cobra.Command) error {
	printBanner()
	repoRoot, err := paths.RepoRoot()
	if err != nil {
		return errors.Wrap(err, "snapshot must be run within a git repository")
	}

	coreVersion, err := readCoreVersion(repoRoot)
	if err != nil {
		return errors.Wrap(err, "failed to read core version")
	}

	schemasDir := filepath.Join(repoRoot, "schemas")
	snapshotsDir := filepath.Join(schemasDir, ".snapshots")

	if err := snapshot.Create(schemasDir, snapshotsDir, coreVersion); err != nil {
		return errors.Wrap(err, "failed to create schema snapshot")
	}

	printDim(fmt.Sprintf("snapshot v%d created", coreVersion))
	return nil
}
