// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Command exports prints the exported Go identifiers that no other package uses and
// exits with status 1 when it finds any. With -base, it prints only the findings that
// the same module directories under the base checkout lack. -workspace adds modules
// that the checked modules depend on, without checking them.
//
//	go run ./cmd/exports [-goos linux] [-base dir] [-workspace a,b] <module dir>...
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
	base := flag.String(
		"base", "", "checkout to compare with; print only the findings it lacks",
	)
	workspace := flag.String(
		"workspace", "", "comma-separated modules to add to the workspace",
	)
	flag.Parse()
	cfg := exports.Config{
		Modules: flag.Args(),
		GOOS:    strings.Split(*goos, ","),
		Base:    *base,
	}
	if *workspace != "" {
		cfg.Workspace = strings.Split(*workspace, ",")
	}
	findings, err := exports.Check(context.Background(), cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	summary := "%d exported identifiers no other package uses\n"
	if *base != "" {
		summary = "%d new exported identifiers no other package uses\n"
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
		fmt.Fprintf(os.Stderr, summary, len(findings))
		os.Exit(1)
	}
}
