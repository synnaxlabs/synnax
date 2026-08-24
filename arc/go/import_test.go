// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/telem"
)

// analyze parses and runs full source-level analysis with the STL prelude
// attached, plus an optional flat list of channels attached to the same
// ambient. Returns the diagnostics it produced.
func analyze(source string, extras ...[]symbol.Symbol) *diagnostics.Diagnostics {
	t, parseDiag := text.Parse(text.Text{Raw: source})
	if parseDiag != nil && !parseDiag.Ok() {
		return parseDiag
	}
	root := symbol.NewRoot(nil, stl.NewSymbols())
	for _, set := range extras {
		for i := range set {
			s := set[i]
			root.Parent.AddChild(&s)
		}
	}
	_, diag := text.Analyze(context.Background(), t, root)
	return diag
}

// messages flattens every diagnostic's user-facing message into a single
// string for substring assertions.
func messages(d *diagnostics.Diagnostics) string {
	if d == nil {
		return ""
	}
	parts := make([]string, 0, len(*d))
	for _, m := range *d {
		parts = append(parts, m.Message)
		for _, n := range m.Notes {
			parts = append(parts, n.Message)
		}
	}
	return strings.Join(parts, "\n")
}

var _ = Describe("Imports", func() {
	chans := channelSymbols(map[string]channelDef{
		"trig":      {types.U8(), 100},
		"now_out":   {types.I64(), 101},
		"sensor":    {types.F64(), 102},
		"output":    {types.F64(), 103},
		"valve_cmd": {types.U8(), 104},
	})

	Describe("required for qualified module access", func() {
		It("Should reject time.now without an import", func() {
			d := analyze(`
trig -> get_now{} -> now_out
func get_now(t u8) i64 {
    return time.now()
}
`, chans)
			Expect(d.Ok()).To(BeFalse())
			Expect(messages(d)).To(ContainSubstring(`module "time" is not imported`))
			Expect(messages(d)).To(ContainSubstring(`add ` + "`import time`"))
		})

		It("Should accept time.now with import time", func() {
			d := analyze(`
import time

trig -> get_now{} -> now_out
func get_now(t u8) i64 {
    return time.now()
}
`, chans)
			Expect(d.Ok()).To(BeTrue(), messages(d))
		})

		It("Should accept time.now as a flow node with import time", func() {
			d := analyze(`
import time

trig -> time.now{} -> now_out
`, chans)
			Expect(d.Ok()).To(BeTrue(), messages(d))
		})

		It("Should accept multiple modules in one block", func() {
			d := analyze(`
import ( time control )

trig -> control.set_authority{value=200, channel=valve_cmd}
func get_now() i64 { return time.now() }
`, chans)
			Expect(d.Ok()).To(BeTrue(), messages(d))
		})

		It("Should accept control.set_authority with import control", func() {
			d := analyze(`
import control

trig -> control.set_authority{value=200, channel=valve_cmd}
`, chans)
			Expect(d.Ok()).To(BeTrue(), messages(d))
		})
	})

	Describe("aliases", func() {
		It("Should bind the alias as the qualifier", func() {
			d := analyze(`
import time as t

trig -> t.now{} -> now_out
`, chans)
			Expect(d.Ok()).To(BeTrue(), messages(d))
		})

		It("Should reject the original name when aliased", func() {
			d := analyze(`
import time as t

trig -> time.now{} -> now_out
`, chans)
			Expect(d.Ok()).To(BeFalse())
			Expect(messages(d)).To(ContainSubstring(`module "time" is not imported`))
		})

		It("Should rewrite the alias to the canonical module in IR node types", func() {
			t, parseDiag := text.Parse(text.Text{Raw: `
import time as t

trig -> t.now{} -> now_out
`})
			if parseDiag != nil {
				Expect(parseDiag.Ok()).To(BeTrue())
			}
			root := symbol.NewRoot(nil, stl.NewSymbols())
			for i := range chans {
				s := chans[i]
				root.Parent.AddChild(&s)
			}
			i, d := text.Analyze(context.Background(), t, root)
			Expect(d.Ok()).To(BeTrue(), messages(d))
			var found bool
			for _, n := range i.Nodes {
				if n.Type == "time.now" {
					found = true
				}
				Expect(
					n.Type,
				).ToNot(Equal("t.now"), "IR node type still uses the alias")
			}
			Expect(
				found,
			).To(BeTrue(), "expected an IR node with canonical type time.now")
		})
	})

	Describe("hierarchical paths", func() {
		It("Should error on unknown module 'math.trig'", func() {
			// math.trig is grammatically valid but no module is registered
			// under that path, so the analyzer rejects the import itself.
			d := analyze(`
import math.trig

sensor -> trig.sin{} -> output
`, chans)
			Expect(d.Ok()).To(BeFalse())
			Expect(messages(d)).To(ContainSubstring(`unknown module "math.trig"`))
		})
	})

	Describe("diagnostics", func() {
		It("Should report unused imports", func() {
			d := analyze(`
import time

trig -> set_authority{value=200, channel=valve_cmd}
`, chans)
			Expect(d.Ok()).To(BeFalse())
			Expect(messages(d)).To(ContainSubstring(`imported module "time" is unused`))
		})

		It("Should report duplicate imports", func() {
			d := analyze(`
import ( time time )

trig -> get_now{} -> now_out
func get_now(t u8) i64 {
    return time.now()
}
`, chans)
			Expect(d.Ok()).To(BeFalse())
			Expect(messages(d)).To(ContainSubstring(`duplicate import`))
		})

		It("Should report unknown modules", func() {
			d := analyze(`
import banana

sensor -> avg{} -> output
`, chans)
			Expect(d.Ok()).To(BeFalse())
			Expect(messages(d)).To(ContainSubstring(`unknown module "banana"`))
		})

		It("Should not double-report an unknown module as unused", func() {
			d := analyze(`
import banana

sensor -> avg{} -> output
`, chans)
			Expect(
				messages(d),
			).ToNot(ContainSubstring(`imported module "banana" is unused`))
		})

		It("Should not flag a misspelled member as an unused import", func() {
			d := analyze(`
import time

trig -> time.doesnotexist{} -> now_out
`, chans)
			Expect(d.Ok()).To(BeFalse())
			Expect(
				messages(d),
			).ToNot(ContainSubstring(`imported module "time" is unused`))
		})
	})

	Describe("ordering", func() {
		It("Should reject an import that follows a function declaration", func() {
			d := analyze(`
func get_now() i64 { return 0 }

import time

trig -> get_now{} -> now_out
`, chans)
			Expect(d.Ok()).To(BeFalse())
			Expect(messages(d)).To(ContainSubstring(
				`import statements must appear before any other top-level declarations`,
			))
		})

		It("Should reject an import that follows a flow statement", func() {
			d := analyze(`
sensor -> output

import time
`, chans)
			Expect(d.Ok()).To(BeFalse())
			Expect(messages(d)).To(ContainSubstring(
				`import statements must appear before any other top-level declarations`,
			))
		})

		It("Should reject an import that follows the authority block", func() {
			d := analyze(`
authority 100

import control

trig -> control.set_authority{value=200, channel=valve_cmd}
`, chans)
			Expect(d.Ok()).To(BeFalse())
			Expect(messages(d)).To(ContainSubstring(
				`import statements must appear before any other top-level declarations`,
			))
		})

		It("Should accept imports interleaved among themselves at the top", func() {
			d := analyze(`
import time
import control

trig -> control.set_authority{value=200, channel=valve_cmd}
func get_now() i64 { return time.now() }
`, chans)
			Expect(d.Ok()).To(BeTrue(), messages(d))
		})
	})

	Describe("authority block ordering", func() {
		It(
			"Should compile and run import + authority + dotted call",
			func(ctx SpecContext) {
				resolver := channelSymbols(map[string]channelDef{
					"trigger_cmd": {types.U8(), 100},
					"valve_cmd":   {types.U8(), 101},
				})
				h := newRuntimeHarness(ctx, `
				import control
				authority 100
				trigger_cmd -> control.set_authority{value=200, channel=valve_cmd}
			`, resolver,
					channels.Digest{Key: 100, DataType: telem.Uint8T},
					channels.Digest{Key: 101, DataType: telem.Uint8T},
				)
				defer h.Close(ctx)

				h.Ingest(100, telem.NewSeriesV[uint8](1))
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()

				changes := h.FlushAuthority()
				Expect(changes).To(HaveLen(1),
					"control.set_authority should buffer one authority change per activation")
				Expect(changes[0].Authority).To(Equal(uint8(200)))
				Expect(changes[0].Channel).ToNot(BeNil())
				Expect(*changes[0].Channel).To(Equal(uint32(101)))
			},
		)
	})
})
