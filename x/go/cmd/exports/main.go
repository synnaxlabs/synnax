// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Command exports prints the exported Go identifiers that no other package uses and
// exits with status 1 when it finds any.
//
//	go run ./cmd/exports [-goos linux,windows] <module dir>...
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/synnaxlabs/x/exports"
)

func main() {
	goos := flag.String(
		"goos", "darwin,linux,windows", "comma-separated GOOS values to load",
	)
	flag.Parse()
	findings, err := exports.Check(context.Background(), exports.Config{
		Modules: flag.Args(),
		GOOS:    strings.Split(*goos, ","),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	for _, f := range findings {
		if rel, err := filepath.Rel(wd, f.Position.Filename); err == nil {
			f.Position.Filename = rel
		}
		fmt.Println(f)
	}
	if len(findings) > 0 {
		fmt.Fprintf(
			os.Stderr,
			"%d exported identifiers no other package uses\n",
			len(findings),
		)
		os.Exit(1)
	}
}
