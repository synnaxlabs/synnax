// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package main

import (
	"flag"
	"fmt"
	"os"
)

var verboseFlag = flag.Bool("v", false, "verbose")

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Usage: program [-v] <configfile>")
		os.Exit(1)
	}

	testConfigFileName := flag.Arg(0)

	exitCode := runTest(testConfigFileName, *verboseFlag)
	os.Exit(exitCode)
}
