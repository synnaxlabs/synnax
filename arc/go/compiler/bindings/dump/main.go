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
