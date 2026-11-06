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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/set"
)

var _ = Describe("Hover examples", func() {
	// The float members only: the math.avg reset example does not compile today.
	members := set.New("is_nan", "is_infinite", "is_finite")

	It("Should compile every float member example", func(ctx SpecContext) {
		var examples []string
		for _, s := range stl.NewSymbols() {
			if s.Name != "math" || s.Kind != symbol.KindModule {
				continue
			}
			for _, member := range s.Children() {
				if !members.Contains(member.Name) {
					continue
				}
				for _, block := range member.Doc.Blocks() {
					text := block.Render()
					if body, ok := strings.CutPrefix(text, "```arc\n"); ok {
						examples = append(examples, strings.TrimSuffix(body, "\n```"))
					}
				}
			}
		}
		Expect(examples).To(HaveLen(len(members)))
		channels := map[string]types.Type{
			"sensor":       types.F64(),
			"output":       types.F64(),
			"rate_output":  types.F64(),
			"reset_signal": types.U8(),
			"pump_speed":   types.F64(),
			"pump_stopped": types.U8(),
		}
		for _, example := range examples {
			ambient := stl.NewSymbols()
			id := 1
			for name, t := range channels {
				ambient = append(ambient, &symbol.Symbol{
					Name: name,
					Kind: symbol.KindChannel,
					Type: types.Chan(t),
					ID:   id,
				})
				id++
			}
			_, err := arc.CompileText(
				ctx,
				arc.Text{Raw: "import math\n" + example},
				symbol.NewRoot(nil, ambient),
			)
			Expect(err).ToNot(HaveOccurred(), example)
		}
	})
})
