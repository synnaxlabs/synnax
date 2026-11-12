// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package update provides the commands that update the Core and undo an update.
package update

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/cmd/instrumentation"
	"github.com/synnaxlabs/synnax/cmd/start"
	"github.com/synnaxlabs/synnax/pkg/release"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

const (
	flagFile    = "file"
	flagTimeout = "timeout"
)

// NewCmd returns the command that updates the Core, with the subcommands that list the
// metadata backups, undo an update, and allow a failed update to run again. The command
// downloads releases as cfg sets.
func NewCmd(cfg release.Config) *cobra.Command {
	group := &cobra.Command{
		Use:   "update [version]",
		Short: "Update the Core, or undo an update",
		Long: "Replaces the installed Core with Core <version>, which the command " +
			"downloads from the Synnax releases and checks against its signature, " +
			"or with the Core binary that --file names. For a Core service, the " +
			"command stops the service, replaces the binary, starts the service, " +
			"and waits until the new Core commits the update. If the new Core fails " +
			"and undoes the update, the command puts the old Core back and starts " +
			"it. Without a service, stop the Core before the update and start it " +
			"after. Run the command as an administrator.\n\nBefore a new version " +
			"first opens the data, the Core backs up its metadata. The subcommands " +
			"list the backups, undo an update, and allow a failed update to run " +
			"again. A backup holds no telemetry. Without --data, the subcommands use " +
			"the data directory of the Core service, if there is one.",
		Example: "sudo synnax update 0.61.0\n" +
			"sudo synnax update --file ./synnax-v0.61.0-linux",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file, err := cmd.Flags().GetString(flagFile)
			if err != nil {
				return err
			}
			timeout, err := cmd.Flags().GetDuration(flagTimeout)
			if err != nil {
				return err
			}
			ctx, w := cmd.Context(), cmd.OutOrStdout()
			switch {
			case len(args) == 1 && file != "":
				return errors.Mark(
					errors.New("Give a version or --file, not both."),
					validate.ErrValidation,
				)
			case len(args) == 1:
				cmd.SilenceUsage = true
				return fromRelease(ctx, w, cfg, args[0], timeout)
			case file != "":
				cmd.SilenceUsage = true
				return fromFile(ctx, w, file, timeout)
			default:
				return cmd.Help()
			}
		},
	}
	group.Flags().String(flagFile, "", "Path of the Core binary to update to")
	group.Flags().Duration(
		flagTimeout,
		10*time.Minute,
		"How long to wait for the new Core service to commit the update",
	)
	undoCmd := &cobra.Command{
		Use:   "undo [backup]",
		Short: "Undo the last update, or go back to an older backup",
		Long: "Puts back the metadata of the newest backup, or of the named one, " +
			"and prints the Core version to run on the data. Stop the Core first. " +
			"Telemetry stays, but the telemetry of channels created after the " +
			"backup moves to an orphaned directory in the backups directory. When " +
			"the binary that synnax update replaced is the Core version to run, the " +
			"command puts it back.",
		Example: "synnax update undo --data /mnt/ssd1\n" +
			"synnax update undo 1793000000-0.60.1 --data /mnt/ssd1",
		Args:    cobra.MaximumNArgs(1),
		PreRunE: bindFlags,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			start.DisablePermissionBits()
			ins := instrumentation.Configure()
			defer instrumentation.Cleanup(cmd.Context(), ins)
			backups, dir, err := openBackups(ins)
			if err != nil {
				return err
			}
			return undo(cmd.Context(), cmd.OutOrStdout(), backups, dir, args)
		},
	}
	retryCmd := &cobra.Command{
		Use:   "retry",
		Short: "Allow a failed update to run again",
		Long: "After a failed update, the Core refuses to start the version that " +
			"failed. Fix the cause of the failure, then run this command, and the " +
			"version updates the data again at its next start.",
		Example: "synnax update retry --data /mnt/ssd1",
		Args:    cobra.NoArgs,
		PreRunE: bindFlags,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true
			ins := instrumentation.Configure()
			defer instrumentation.Cleanup(cmd.Context(), ins)
			backups, _, err := openBackups(ins)
			if err != nil {
				return err
			}
			failed, err := backups.Retry()
			if err != nil {
				return err
			}
			return printLines(cmd.OutOrStdout(), fmt.Sprintf(
				"Core %s updates the data again at its next start.",
				failed,
			))
		},
	}
	historyCmd := &cobra.Command{
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
			backups, _, err := openBackups(ins)
			if err != nil {
				return err
			}
			return history(cmd.OutOrStdout(), backups)
		},
	}
	addFlags(undoCmd)
	addFlags(historyCmd)
	addFlags(retryCmd)
	group.AddCommand(undoCmd, historyCmd, retryCmd)
	return group
}

func bindFlags(cmd *cobra.Command, _ []string) error {
	return viper.BindPFlags(cmd.Flags())
}

func undo(
	ctx context.Context,
	w io.Writer,
	backups *storage.Backups,
	dir string,
	args []string,
) error {
	inst, installed, err := installation()
	if err != nil {
		return err
	}
	// Without a Core service, or with one that uses dir, the binary belongs to dir.
	owned := !installed || filepath.Clean(inst.Data) == dir
	var res storage.Restored
	if len(args) == 1 {
		if res, err = backups.Restore(args[0]); err != nil {
			return errors.Wrapf(err, "failed to restore backup %s", args[0])
		}
	} else if res, err = backups.Undo(); errors.Is(err, storage.ErrUndone) && owned {
		// The data is back, but the binary of the update may still be in place.
		return putBackUndone(ctx, w, backups, inst.Binary, err)
	} else if err != nil {
		return err
	}
	if err = matchOwner(dir); err != nil {
		return errors.Wrap(err, "failed to give the restored files to the data owner")
	}
	lines := []string{fmt.Sprintf("Restored backup %s.", res.Backup.Name)}
	if res.OrphanedDir != "" {
		lines = append(lines, fmt.Sprintf(
			"Moved the telemetry of channels created after the backup to %s.",
			res.OrphanedDir,
		))
	}
	if err = printLines(w, lines...); err != nil {
		return err
	}
	run := fmt.Sprintf("Run Core %s on the data.", res.Backup.Version)
	if res.Backup.Version == storage.UnknownVersion {
		run = "The backup does not record its Core version. Run the Core version " +
			"that ran on the data before the update."
	}
	if !owned {
		return printLines(w, run)
	}
	back, err := putBackFor(ctx, inst.Binary, res.Backup.Version)
	if err != nil {
		return errors.Combine(
			errors.Wrapf(err, "failed to put back the binary at %s", inst.Binary),
			printLines(w, run),
		)
	}
	if back != "" {
		return printLines(w, fmt.Sprintf(
			"Put back Core %s at %s. Start the Core.",
			back,
			inst.Binary,
		))
	}
	return printLines(w, run)
}

// putBackUndone puts back the binary that an undone update replaced, as putBackFor does
// for the newest backup. Without such a binary, it returns undone.
func putBackUndone(
	ctx context.Context,
	w io.Writer,
	backups *storage.Backups,
	binary string,
	undone error,
) error {
	all, err := backups.List()
	if err != nil {
		return errors.Combine(undone, err)
	}
	back, err := putBackFor(ctx, binary, all[len(all)-1].Version)
	if err != nil || back == "" {
		return errors.Combine(undone, err)
	}
	return printLines(
		w,
		"The Core already undid the update.",
		fmt.Sprintf("Put back Core %s at %s. Start the Core.", back, binary),
	)
}

// putBackFor swaps binary with the binary that an update replaced, when that binary is
// Core version v of a restored backup. For a backup with no version, that binary must
// be older than binary. putBackFor returns the version it put back, or an empty string
// when it put back nothing.
func putBackFor(ctx context.Context, binary, v string) (string, error) {
	if v == storage.UnknownVersion {
		old, err := oldVersion(ctx, binary)
		if err != nil || old == "" {
			return "", err
		}
		current, err := binaryVersion(ctx, binary)
		if err != nil || version.Compare(old, current) >= 0 {
			return "", err
		}
		v = old
	}
	putBack, err := putBackOld(ctx, binary, v)
	if err != nil || !putBack {
		return "", err
	}
	return v, nil
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
