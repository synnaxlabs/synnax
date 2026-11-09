// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package math_test

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/lsp/doc"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

// hoverExamples returns one table entry for each Arc code block in the hover docs of
// the math module and its members, and the names of the members that have one.
func hoverExamples() ([]TableEntry, set.Set[string]) {
	var (
		entries []TableEntry
		names   = make(set.Set[string])
	)
	add := func(name string, d doc.Doc) {
		for i, block := range d.Blocks() {
			if code, ok := strings.CutPrefix(block.Render(), "```arc\n"); ok {
				names.Add(name)
				entries = append(entries, Entry(
					fmt.Sprintf("%s block %d", name, i),
					strings.TrimSuffix(code, "\n```"),
				))
			}
		}
	}
	for _, s := range stl.NewSymbols() {
		if s.Name != "math" || s.Kind != symbol.KindModule {
			continue
		}
		add("math", s.Doc)
		for _, member := range s.Children() {
			add("math."+member.Name, member.Doc)
		}
	}
	return entries, names
}

// newExampleRoot returns a root with the channels the hover examples use.
func newExampleRoot() *symbol.Symbol {
	ambient := stl.NewSymbols()
	for i, name := range []string{
		"sensor", "output", "rate_output", "pump_speed", "pump_stopped",
	} {
		ambient = append(ambient, &symbol.Symbol{
			Name: name,
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   i + 1,
		})
	}
	return symbol.NewRoot(nil, ambient)
}

var _ = Describe("Hover examples", func() {
	entries, names := hoverExamples()

	It("Should find an example for every documented member", func() {
		Expect(names).To(Equal(set.New(
			"math.avg",
			"math.min",
			"math.max",
			"math.derivative",
			"math.is_nan",
			"math.is_infinite",
			"math.is_finite",
		)))
	})

	DescribeTable("Should compile",
		func(ctx SpecContext, code string) {
			MustSucceed(arc.CompileText(
				ctx,
				arc.Text{Raw: "import math\n" + code},
				newExampleRoot(),
			))
		},
		entries,
	)
})
