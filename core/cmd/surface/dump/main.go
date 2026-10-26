// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Command dump prints the commands, flags, and config keys of the Core. Compare the
// output of two revisions to see if a pull request changes them.
//
//	go run ./cmd/surface/dump
package main

import (
	"fmt"
	"os"

	"github.com/synnaxlabs/synnax/cmd"
	"github.com/synnaxlabs/synnax/cmd/surface"
)

func main() {
	if err := surface.Write(os.Stdout, cmd.Cmd); err != nil {
		if _, printErr := fmt.Fprintln(os.Stderr, err); printErr != nil {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
