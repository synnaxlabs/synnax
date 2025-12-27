// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Command dump prints all import function names that SetupImports registers.
// Used by validation scripts to compare against C++ runtime.
//
// Usage: go run ./arc/go/compiler/bindings/dump
package main

import (
	"fmt"

	"github.com/synnaxlabs/arc/compiler/bindings"
	"github.com/synnaxlabs/arc/compiler/wasm"
)

func main() {
	m := wasm.NewModule()
	bindings.SetupImports(m)
	for _, name := range m.ImportNames() {
		fmt.Println(name)
	}
}
