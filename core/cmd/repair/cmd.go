// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package repair provides the command that repairs damage of the telemetry files that
// stops the Core from opening them.
package repair

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/cmd/instrumentation"
	"github.com/synnaxlabs/synnax/cmd/service"
	"github.com/synnaxlabs/synnax/cmd/start"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

const flagYes = "yes"

// NewCmd returns the repair command of a Core at version vsn, which repairs the data
// with layer as the base of its storage config. The command calls configure for its
// instrumentation once its flags are parsed.
func NewCmd(
	vsn string,
	configure func() alamos.Instrumentation,
	layer storage.LayerConfig,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repair",
		Short: "Repair damage of the telemetry files that stops the Core from opening",
		Long: "Finds the damage of the log of the telemetry files that stops the " +
			"Core from opening them, lists what the repair rebuilds and what it " +
			"cannot place, and asks before it writes. The repair keeps the damaged " +
			"files under new names. Stop the Core first. Without --data, the command " +
			"uses the data directory of the Core service, if there is one. With no " +
			"terminal to ask on, it repairs only with --yes.",
		Example: "synnax repair --data /mnt/ssd1\n" +
			"synnax repair --data /mnt/ssd1 --yes",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true
			ins := configure()
			defer instrumentation.Cleanup(cmd.Context(), ins)
			dir, err := service.DataDir()
			if err != nil {
				return err
			}
			layer.Instrumentation, layer.Dirname, layer.Version = ins, dir, vsn
			return run(
				cmd.Context(),
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
				layer,
				viper.GetBool(flagYes),
			)
		},
	}
	instrumentation.AddFlags(cmd)
	start.AddDataFlag(cmd)
	cmd.Flags().Bool(flagYes, false, "Repair with no question")
	return cmd
}

func run(
	ctx context.Context,
	in io.Reader,
	w io.Writer,
	layer storage.LayerConfig,
	yes bool,
) error {
	var asked error
	rep, err := storage.Repair(ctx, layer, func(rep cesium.Report) bool {
		if asked = printReport(w, rep); asked != nil {
			return false
		}
		var ok bool
		ok, asked = confirm(in, w, yes)
		return ok
	})
	if err = errors.Join(err, asked); err != nil {
		return err
	}
	switch {
	case len(rep.Damage) == 0 && len(rep.Buried) == 0:
		_, err = fmt.Fprintln(w, "nothing to repair: the database opens")
	case rep.Repaired:
		err = printRenames(w, rep)
	default:
		_, err = fmt.Fprintln(w, "Stopped the repair. No file changed.")
	}
	return err
}

// errNoTerminal is the refusal of a repair with no terminal to ask on and no --yes.
var errNoTerminal = errors.New(
	"stopped the repair: there is no terminal to ask on. To repair, run the command " +
		"again with --yes",
)

// confirm asks to repair on in, and reports whether the answer is yes. With yes, it
// asks nothing. It returns an error that names --yes when in is a file that is not a
// terminal.
func confirm(in io.Reader, w io.Writer, yes bool) (bool, error) {
	if yes {
		return true, nil
	}
	if f, ok := in.(*os.File); ok {
		if info, err := f.Stat(); err != nil || info.Mode()&os.ModeCharDevice == 0 {
			return false, errNoTerminal
		}
	}
	if _, err := fmt.Fprint(w, "Repair? [y/N] "); err != nil {
		return false, err
	}
	answer, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	a := strings.ToLower(strings.TrimSpace(answer))
	return a == "y" || a == "yes", nil
}

// table is one list of a report: a title, the head of its columns, and its lines.
type table struct {
	title, head string
	rows        []string
}

// printReport prints each list of rep that holds lines, as a table.
func printReport(w io.Writer, rep cesium.Report) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, t := range tables(rep) {
		if len(t.rows) == 0 {
			continue
		}
		if _, err := fmt.Fprintf(
			tw,
			"%s\n%s\n%s\n\n",
			t.title,
			t.head,
			strings.Join(t.rows, "\n"),
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}

// tables returns the lists of rep as tables, in the order that the command prints them.
func tables(rep cesium.Report) []table {
	return []table{
		{
			title: "Damaged log frames",
			head:  "LOG\tOFFSET\tENTRIES",
			rows: rowsOf(rep.Damage, func(d cesium.Damage) string {
				return fmt.Sprintf("%d\t%d\t%d to %d", d.Generation, d.Offset,
					d.FirstSequence, d.LastSequence)
			}),
		},
		rangeTable("Rebuilt ranges", rep.Rebuilt),
		rangeTable("Ranges that the repair cannot place", rep.Unplaced),
		rangeTable("Last blocks that nothing checks", rep.Tails),
		{
			title: "Files that the repair keeps aside",
			head:  "CHANNEL\tSEGMENT\tBYTES\tREASON",
			rows: rowsOf(rep.Unrecovered, func(f cesium.File) string {
				return fmt.Sprintf(
					"%d\t%d\t%d\t%s",
					f.Key,
					f.Segment,
					f.Bytes,
					f.Reason,
				)
			}),
		},
		segmentTable("Segments whose files are gone", rep.Removed),
		segmentTable("Sealed segments with no data", rep.Buried),
		recordTable(
			"Records of the damaged frames, which the repair does not apply",
			rep.Lost,
		),
		recordTable("Later records that the repair skips", rep.Skipped),
	}
}

func rangeTable(title string, ranges []cesium.Range) table {
	return table{
		title: title,
		head:  "CHANNEL\tSEGMENT\tBYTES\tSAMPLES\tTIME RANGE\tREASON",
		rows: rowsOf(ranges, func(r cesium.Range) string {
			return fmt.Sprintf("%d\t%d\t%d\t%d\t%s\t%s", r.Key, r.Segment, r.Bytes,
				r.Samples, r.TimeRange, r.Reason)
		}),
	}
}

func segmentTable(title string, segs []cesium.Segment) table {
	return table{
		title: title,
		head:  "CHANNEL\tSEGMENT",
		rows: rowsOf(segs, func(s cesium.Segment) string {
			return fmt.Sprintf("%d\t%d", s.Key, s.Segment)
		}),
	}
}

func recordTable(title string, records []cesium.Record) table {
	return table{
		title: title,
		head:  "ENTRY\tRECORD\tCHANNEL\tTIME RANGE\tREASON",
		rows: rowsOf(records, func(r cesium.Record) string {
			return fmt.Sprintf("%d\t%s\t%d\t%s\t%s", r.Sequence, r.Kind, r.Key,
				r.TimeRange, r.Reason)
		}),
	}
}

// rowsOf returns the line of each of items.
func rowsOf[T any](items []T, row func(T) string) []string {
	rows := make([]string, len(items))
	for i, item := range items {
		rows[i] = row(item)
	}
	return rows
}

// printRenames prints the names that the repair gave the damaged logs and the files
// that it keeps aside.
func printRenames(w io.Writer, rep cesium.Report) error {
	if _, err := fmt.Fprintln(
		w,
		"Repaired. The repair kept these files under new names:",
	); err != nil {
		return err
	}
	seen := set.New[string]()
	for _, d := range rep.Damage {
		if !seen.Contains(d.Name) {
			seen.Add(d.Name)
			if _, err := fmt.Fprintln(w, "  "+d.Name); err != nil {
				return err
			}
		}
	}
	for _, f := range rep.Unrecovered {
		if _, err := fmt.Fprintln(w, "  "+f.Name); err != nil {
			return err
		}
	}
	return nil
}
