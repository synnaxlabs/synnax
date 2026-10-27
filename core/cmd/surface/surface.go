// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package surface writes the commands and flags of a command tree as text, one line
// per command and flag, so the reports of two revisions differ only when the command
// line changes. The Core binds every flag to a config key and an environment variable
// of the same name, so the report covers those too.
package surface

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Write writes the report of root and every command below it, sorted by command path.
// Help text is left out.
func Write(w io.Writer, root *cobra.Command) error {
	var lines []string
	collect(root, &lines)
	slices.Sort(lines)
	_, err := io.WriteString(w, strings.Join(lines, ""))
	return err
}

func collect(c *cobra.Command, lines *[]string) {
	path := c.CommandPath()
	line := path + " command"
	if c.Hidden {
		line += " hidden"
	}
	if len(c.Aliases) > 0 {
		line += " aliases=" + strings.Join(c.Aliases, ",")
	}
	*lines = append(*lines, line+"\n")
	c.LocalNonPersistentFlags().VisitAll(func(f *pflag.Flag) {
		*lines = append(*lines, flag(path, f, ""))
	})
	c.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		*lines = append(*lines, flag(path, f, " persistent"))
	})
	for _, sub := range c.Commands() {
		collect(sub, lines)
	}
}

func flag(path string, f *pflag.Flag, scope string) string {
	line := fmt.Sprintf("%s --%s", path, f.Name)
	if f.Shorthand != "" {
		line += " -" + f.Shorthand
	}
	line += fmt.Sprintf(" %s default=%q%s", f.Value.Type(), f.DefValue, scope)
	if f.Hidden {
		line += " hidden"
	}
	if f.Deprecated != "" {
		line += " deprecated"
	}
	return line + "\n"
}
