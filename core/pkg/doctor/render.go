// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package doctor

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/synnaxlabs/cesium/inspect"
	"github.com/synnaxlabs/x/errors"
)

// order fixes the order severities are rendered in.
var order = []Severity{SeverityError, SeverityWarning, SeverityInfo}

// RenderJSON writes the report as one JSON document.
func RenderJSON(w io.Writer, r Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// Render writes the report as text. Verbose adds info findings and the per-channel
// statistics table.
func Render(w io.Writer, r Report, verbose bool) error {
	t := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	renderHeader(t, r)
	renderFindings(t, r, verbose)
	if verbose && r.TS != nil {
		renderChannels(t, r.TS)
	}
	renderVerdict(t, r, verbose)
	return errors.Wrap(t.Flush(), "rendering report")
}

// renderHeader writes the directory, cluster identity, and per-store totals.
func renderHeader(w io.Writer, r Report) {
	fmt.Fprintf(w, "directory\t%s\n", r.Dirname)
	if r.KV != nil {
		c := r.KV.Cluster
		if len(c.Nodes) == 0 {
			fmt.Fprintf(w, "cluster\tnot initialized\n")
		} else {
			fmt.Fprintf(w, "cluster\t%s\n", c.Key)
			fmt.Fprintf(w, "host node\t%s of %d\n", c.HostKey, len(c.Nodes))
		}
		fmt.Fprintf(
			w,
			"key-value\t%d entries, %s\n",
			r.KV.Entries,
			formatBytes(r.KV.Bytes),
		)
	}
	if r.KVUnavailable != "" {
		fmt.Fprintf(w, "key-value\tnot read: %s\n", r.KVUnavailable)
	}
	if r.TS != nil {
		t := r.TS.Totals
		fmt.Fprintf(
			w,
			"time-series\t%d channels, %s on disk, %s garbage\n",
			len(r.TS.Channels),
			formatBytes(int64(t.DiskBytes)),
			formatBytes(int64(t.GarbageBytes)),
		)
	}
	fmt.Fprintln(w)
}

// renderFindings writes every finding, grouped by severity.
func renderFindings(w io.Writer, r Report, verbose bool) {
	for _, severity := range order {
		if severity == SeverityInfo && !verbose {
			continue
		}
		matched := make([]Finding, 0, len(r.Findings))
		for _, f := range r.Findings {
			if f.Severity == severity {
				matched = append(matched, f)
			}
		}
		if len(matched) == 0 {
			continue
		}
		fmt.Fprintf(w, "%ss\n", severity)
		for _, f := range matched {
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", f.Check, f.Subject, f.Message, f.Hint)
		}
		fmt.Fprintln(w)
	}
}

// renderChannels writes the per-channel statistics table.
func renderChannels(w io.Writer, r *inspect.Report) {
	fmt.Fprintln(w, "channel\tname\tdomains\tsamples\tdisk\tgarbage\ttime range")
	for _, ch := range r.Channels {
		s := ch.Stats
		fmt.Fprintf(
			w,
			"  %d\t%s\t%d\t%d\t%s\t%s\t%s\n",
			ch.Key,
			ch.Channel.Name,
			s.Domains,
			s.Samples,
			formatBytes(int64(s.DiskBytes)),
			formatBytes(int64(s.GarbageBytes)),
			s.TimeRange,
		)
	}
	fmt.Fprintln(w)
}

// renderVerdict writes the closing count of findings by severity.
func renderVerdict(w io.Writer, r Report, verbose bool) {
	counts := make(map[Severity]int, len(order))
	for _, f := range r.Findings {
		counts[f.Severity]++
	}
	if counts[SeverityError] == 0 && counts[SeverityWarning] == 0 {
		fmt.Fprintln(w, "no problems found")
		if !verbose && counts[SeverityInfo] > 0 {
			fmt.Fprintf(
				w,
				"%d notes hidden; run with --verbose\n",
				counts[SeverityInfo],
			)
		}
		return
	}
	fmt.Fprintf(
		w,
		"%d errors, %d warnings\n",
		counts[SeverityError],
		counts[SeverityWarning],
	)
}

// units names each power of 1024 the byte formatter uses.
var units = []string{"B", "KB", "MB", "GB", "TB"}

// formatBytes renders a byte count in the largest unit that keeps it above one.
func formatBytes(b int64) string {
	value := float64(b)
	for _, unit := range units {
		if value < 1024 || unit == units[len(units)-1] {
			if unit == "B" {
				return fmt.Sprintf("%d%s", b, unit)
			}
			return fmt.Sprintf("%.1f%s", value, unit)
		}
		value /= 1024
	}
	return fmt.Sprintf("%dB", b)
}
