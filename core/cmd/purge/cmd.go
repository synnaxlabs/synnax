// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package purge provides the command that removes data that has no channel.
package purge

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/cmd/instrumentation"
	"github.com/synnaxlabs/synnax/cmd/service"
	"github.com/synnaxlabs/synnax/cmd/start"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// NewCmd returns the purge command of a Core at version vsn. The command calls
// configure for its instrumentation once its flags are parsed.
func NewCmd(vsn string, configure func() alamos.Instrumentation) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "purge [channel...]",
		Short: "List or remove data that has no channel",
		Long: "Lists the channels that hold data but have no channel record and no " +
			"delete, with the bytes of their data. With channel keys, the command " +
			"removes the data of those channels. Stop the Core first. Without " +
			"--data, the command uses the data directory of the Core service, if " +
			"there is one.",
		Example: "synnax purge --data /mnt/ssd1\n" +
			"synnax purge 12 13 --data /mnt/ssd1",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			keys, err := parseKeys(args)
			if err != nil {
				return err
			}
			ins := configure()
			defer instrumentation.Cleanup(cmd.Context(), ins)
			return run(cmd.Context(), cmd.OutOrStdout(), ins, vsn, keys)
		},
	}
	instrumentation.AddFlags(cmd)
	start.AddDataFlag(cmd)
	return cmd
}

func parseKeys(args []string) ([]channel.Key, error) {
	keys := make([]channel.Key, len(args))
	for i, arg := range args {
		k, err := strconv.ParseUint(arg, 10, 32)
		if err != nil {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"%q is not a channel key",
				arg,
			)
		}
		keys[i] = channel.Key(k)
	}
	return keys, nil
}

func run(
	ctx context.Context,
	w io.Writer,
	ins alamos.Instrumentation,
	vsn string,
	keys []channel.Key,
) (err error) {
	dir, err := service.DataDir()
	if err != nil {
		return err
	}
	// Only a start of vsn keeps data, and a purge must not start an update.
	if !version.IsDev(vsn) {
		if err = checkVersion(ins, dir, vsn); err != nil {
			return err
		}
	}
	layer, err := storage.OpenLayer(ctx, storage.LayerConfig{
		Instrumentation: ins,
		Dirname:         dir,
		Version:         vsn,
	})
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, layer.Close()) }()
	kept, err := channel.Purge(ctx, layer.Gorp, layer.TS, keys)
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		_, err = fmt.Fprintf(w, "Removed the data of %s.\n", channels(keys))
		return err
	}
	if len(kept) == 0 {
		_, err = fmt.Fprintln(w, "No channel holds data without a channel record.")
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err = fmt.Fprintln(tw, "CHANNEL\tBYTES"); err != nil {
		return err
	}
	for _, k := range kept {
		if _, err = fmt.Fprintf(tw, "%d\t%d\n", k.Key, k.Size); err != nil {
			return err
		}
	}
	if err = tw.Flush(); err != nil {
		return err
	}
	_, err = fmt.Fprintln(
		w,
		"\nTo remove the data, run synnax purge <channel>. To keep it, restore the "+
			"channel records from a backup.",
	)
	return err
}

func checkVersion(ins alamos.Instrumentation, dir, vsn string) error {
	backups, err := storage.NewBackups(
		storage.BackupsConfig{Instrumentation: ins, Dirname: dir},
	)
	if err != nil {
		return err
	}
	state, err := backups.UpdateState(vsn)
	if err != nil {
		return err
	}
	if state != storage.UpdateCommitted {
		return errors.Newf(
			"Core %s did not open the data last. Start Core %s on the data, stop it, "+
				"and run the purge again.",
			vsn,
			vsn,
		)
	}
	return nil
}

// channels names keys as prose: "channel 12", "channels 12 and 13", or "channels 12,
// 13, and 14".
func channels(keys []channel.Key) string {
	names := make([]string, len(keys))
	for i, k := range keys {
		names[i] = strconv.FormatUint(uint64(k), 10)
	}
	switch len(names) {
	case 1:
		return "channel " + names[0]
	case 2:
		return "channels " + names[0] + " and " + names[1]
	}
	last := len(names) - 1
	return "channels " + strings.Join(names[:last], ", ") + ", and " + names[last]
}
