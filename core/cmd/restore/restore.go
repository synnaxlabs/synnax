// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package restore provides the command that lists and restores the metadata backups
// of a Core.
package restore

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/cmd/start"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/errors"
)

// NewCmd returns the command that lists the backups of a data directory, or restores
// the one it names.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore [backup]",
		Short: "List or restore the metadata backups of a Core",
		Long:  "Lists the metadata backups in the data directory, or restores the named backup. The Core backs up its metadata before a new version first opens the data. A backup holds no telemetry. Stop the Core before a restore. A restore moves the telemetry of channels that the backup does not know to an orphaned directory in the backups directory.",
		Example: "synnax restore --data /mnt/ssd1\n" +
			"synnax restore 1793000000-0.60.1 --data /mnt/ssd1",
		Args: cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			dirname := viper.GetString(start.FlagData)
			if len(args) == 0 {
				return list(cmd.OutOrStdout(), dirname)
			}
			return restore(cmd.OutOrStdout(), dirname, args[0])
		},
	}
	start.AddDataFlag(cmd)
	return cmd
}

func list(w io.Writer, dirname string) error {
	backups, err := storage.ListBackups(dirname)
	if err != nil {
		return err
	}
	if len(backups) == 0 {
		_, err = fmt.Fprintf(w, "There are no backups in %s.\n", dirname)
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err = fmt.Fprintln(tw, "BACKUP\tTAKEN\tCORE VERSION"); err != nil {
		return err
	}
	for _, b := range backups {
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
	_, err = fmt.Fprintln(w, "\nTo restore a backup, run synnax restore <backup>.")
	return err
}

func restore(w io.Writer, dirname, name string) error {
	res, err := storage.Restore(dirname, name)
	if err != nil {
		return errors.Wrapf(err, "failed to restore backup %s", name)
	}
	lines := []string{fmt.Sprintf("Restored backup %s.", res.Backup.Name)}
	if res.OrphanedDir != "" {
		lines = append(lines, fmt.Sprintf(
			"Moved the telemetry of channels that the backup does not know to %s.",
			res.OrphanedDir,
		))
	}
	if res.Backup.Version == storage.UnknownVersion {
		lines = append(lines, "The backup does not record its Core version. Run "+
			"the Core version that ran on the data before the update.")
	} else {
		lines = append(lines, fmt.Sprintf(
			"Run Core %s on the data.",
			res.Backup.Version,
		))
	}
	for _, line := range lines {
		if _, err = fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}
