// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package update provides the commands that undo an update of the Core.
package update

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/cmd/instrumentation"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/errors"
)

var Cmd = &cobra.Command{
	Use:   "update",
	Short: "Undo an update of the Core",
	Long: "Before a new version first opens the data, the Core backs up its " +
		"metadata. These commands list the backups and undo an update with one. A " +
		"backup holds no telemetry.",
	Args: cobra.NoArgs,
}

var undoCmd = &cobra.Command{
	Use:   "undo [backup]",
	Short: "Undo the last update, or go back to an older backup",
	Long: "Puts back the metadata of the newest backup, or of the named one, and " +
		"prints the Core version to run on the data. Stop the Core first. Telemetry " +
		"stays, but the telemetry of channels created after the backup moves to an " +
		"orphaned directory in the backups directory.",
	Example: "synnax update undo --data /mnt/ssd1\n" +
		"synnax update undo 1793000000-0.60.1 --data /mnt/ssd1",
	Args:    cobra.MaximumNArgs(1),
	PreRunE: bindFlags,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		ins := instrumentation.Configure()
		defer instrumentation.Cleanup(cmd.Context(), ins)
		backups, err := storage.NewBackups(buildBackupsConfig(ins))
		if err != nil {
			return err
		}
		return undo(cmd.OutOrStdout(), backups, args)
	},
}

var historyCmd = &cobra.Command{
	Use:     "history",
	Short:   "List the backups that can undo an update",
	Long:    "Lists the metadata backups in the data directory, oldest first.",
	Example: "synnax update history --data /mnt/ssd1",
	Args:    cobra.NoArgs,
	PreRunE: bindFlags,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		ins := instrumentation.Configure()
		defer instrumentation.Cleanup(cmd.Context(), ins)
		backups, err := storage.NewBackups(buildBackupsConfig(ins))
		if err != nil {
			return err
		}
		return history(cmd.OutOrStdout(), backups)
	},
}

func init() {
	addFlags(undoCmd)
	addFlags(historyCmd)
	Cmd.AddCommand(undoCmd, historyCmd)
}

func bindFlags(cmd *cobra.Command, _ []string) error {
	return viper.BindPFlags(cmd.Flags())
}

func undo(w io.Writer, backups *storage.Backups, args []string) error {
	var name string
	if len(args) == 1 {
		name = args[0]
	} else {
		all, err := backups.List()
		if err != nil {
			return err
		}
		if len(all) == 0 {
			return errors.New("there is no update to undo: the data has no backups")
		}
		name = all[len(all)-1].Name
	}
	res, err := backups.Restore(name)
	if err != nil {
		return errors.Wrapf(err, "failed to restore backup %s", name)
	}
	lines := []string{fmt.Sprintf("Restored backup %s.", res.Backup.Name)}
	if res.OrphanedDir != "" {
		lines = append(lines, fmt.Sprintf(
			"Moved the telemetry of channels created after the backup to %s.",
			res.OrphanedDir,
		))
	}
	if res.Backup.Version == storage.UnknownVersion {
		lines = append(lines, "The backup does not record its Core version. Run "+
			"the Core version that ran on the data before the update.")
	} else {
		lines = append(
			lines,
			fmt.Sprintf("Run Core %s on the data.", res.Backup.Version),
		)
	}
	for _, line := range lines {
		if _, err = fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

func history(w io.Writer, backups *storage.Backups) error {
	all, err := backups.List()
	if err != nil {
		return err
	}
	if len(all) == 0 {
		_, err = fmt.Fprintln(w, "There are no backups.")
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err = fmt.Fprintln(tw, "BACKUP\tTAKEN\tCORE VERSION"); err != nil {
		return err
	}
	for _, b := range all {
		if _, err = fmt.Fprintf(
			tw,
			"%s\t%s\t%s\n",
			b.Name,
			b.Time.Local().Format(time.DateTime),
			b.Version,
		); err != nil {
			return err
		}
	}
	if err = tw.Flush(); err != nil {
		return err
	}
	_, err = fmt.Fprintln(
		w,
		"\nTo undo the last update, run synnax update undo. To go back to an older "+
			"backup, run synnax update undo <backup>.",
	)
	return err
}
