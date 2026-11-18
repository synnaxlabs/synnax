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
	"fmt"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

// TEMPORARY matrix hunt for SY-6254. Delete before commit.
var _ = Describe("MATRIX SY-6254", func() {
	type variable struct {
		name     string
		decl     string
		accept   []string
		topLevel bool
	}
	type use struct {
		name string
		// lines takes the placeholder-rendered values and returns the use lines. %s
		// receives nothing, the variable is always v.
		lines    string
		wrap     func(string) string
		topLevel bool
		nested   bool
	}
	ident := func(s string) string { return s }
	variables := []variable{
		{"literal", `v := 1.23`, []string{"1.23"}, true},
		{"typed literal", `v f64 := 1.23`, []string{"1.23"}, true},
		{"int literal", `v := 5`, []string{"5"}, true},
		{"string literal", `v := "abc"`, []string{"abc"}, true},
		{"bool literal", `v := true`, []string{"true"}, true},
		{"computed", `v := 1.0 + 0.23`, []string{"1.23"}, true},
		{"channel read/write", `v := sensor`, []string{"1.23", "0"}, true},
		{"channel read", `v := sensor * 1.0`, []string{"1.23", "0"}, true},
		{"stateful", `v f64 $= 1.23`, []string{"1.23"}, false},
		{"reassigned", "v := 0.5\nv = 1.23", []string{"1.23", "0.5"}, false},
		{"flow written", "v f64 := 0.5\nsensor -> v", []string{"1.23", "0.5"}, false},
		{"rebound", "v := sensor2\nv = sensor", []string{"1.23", "9.9", "0"}, false},
	}
	uses := []use{
		{name: "bare", lines: `f"{v}" -> log`, wrap: ident, topLevel: true},
		{name: "triggered", lines: `trig -> f"{v}" -> log`, wrap: ident, topLevel: true},
		{
			name:     "interval",
			lines:    `time.interval(1ms) -> f"{v}" -> log`,
			wrap:     ident,
			topLevel: true,
		},
		{name: "assign step", lines: `log = f"{v}"`, wrap: ident},
		{
			name:     "through a variable",
			lines:    "msg := f\"{v}\"\nmsg -> log",
			wrap:     ident,
			topLevel: true,
		},
		{
			name:     "through a variable, triggered",
			lines:    "msg := f\"{v}\"\ntrig -> msg -> log",
			wrap:     ident,
			topLevel: true,
		},
		{
			name:     "prefix text",
			lines:    `f"x={v}" -> log`,
			wrap:     func(s string) string { return "x=" + s },
			topLevel: true,
		},
		{
			name:     "twice",
			lines:    `f"{v}|{v}" -> log`,
			wrap:     func(s string) string { return s + "|" + s },
			topLevel: true,
		},
		{
			name:     "concat expression",
			lines:    `f"{v}" + "!" -> log`,
			wrap:     func(s string) string { return s + "!" },
			topLevel: true,
		},
		{
			name:     "inline stage",
			lines:    "trig -> stage {\nf\"{v}\" -> log\n}",
			wrap:     ident,
			topLevel: true,
		},
		{
			name:     "inline sequence",
			lines:    "trig -> sequence {\nf\"{v}\" -> log\n}",
			wrap:     ident,
			topLevel: true,
		},
		{
			name:     "function input",
			lines:    `trig -> show(f"{v}")`,
			wrap:     ident,
			topLevel: true,
		},
		{
			name:     "function trigger",
			lines:    `f"{v}" -> show_trig()`,
			wrap:     ident,
			topLevel: true,
		},
		{
			name:     "function trigger, triggered",
			lines:    `trig -> f"{v}" -> show_trig()`,
			wrap:     ident,
			topLevel: true,
		},
	}
	const prelude = `import time
func show(s str) {
    log = s
}
func show_trig() {s str} {
    log = s
}
`
	// layouts place the declaration and the use in scopes.
	type layout struct {
		name          string
		build         func(decl, use string) string
		declTopLevel  bool
		useTopLevel   bool
		needsTopLevel bool
	}
	layouts := []layout{
		{
			name:         "top / top",
			build:        func(d, u string) string { return d + "\n" + u },
			declTopLevel: true, useTopLevel: true,
		},
		{
			name: "top / sequence",
			build: func(d, u string) string {
				return d + "\nsequence main {\n" + u + "\n}\ntrig => main"
			},
			declTopLevel: true,
		},
		{
			name: "top / stage",
			build: func(d, u string) string {
				return d + "\nsequence main {\nstage s1 {\n" + u + "\n}\n}\ntrig => main"
			},
			declTopLevel: true,
		},
		{
			name: "sequence / sequence",
			build: func(d, u string) string {
				return "sequence main {\n" + d + "\n" + u + "\n}\ntrig => main"
			},
		},
		{
			name: "sequence / stage",
			build: func(d, u string) string {
				return "sequence main {\n" + d + "\nstage s1 {\n" + u +
					"\n}\n}\ntrig => main"
			},
		},
		{
			name: "stage / stage",
			build: func(d, u string) string {
				return "sequence main {\nstage s1 {\n" + d + "\n" + u +
					"\n}\n}\ntrig => main"
			},
		},
	}

	run := func(ctx SpecContext, program string) (logged []string, failure string) {
		resolver := channelSymbols(map[string]channelDef{
			"trig":    {types.U8(), 100},
			"log":     {types.String(), 101},
			"sensor":  {types.F64(), 102},
			"sensor2": {types.F64(), 103},
		})
		syms := stl.NewSymbols()
		for i := range resolver {
			syms = append(syms, &resolver[i])
		}
		if _, err := arc.CompileText(
			ctx, arc.Text{Raw: program}, symbol.NewRoot(nil, syms),
		); err != nil {
			return nil, "COMPILE: " + strings.ReplaceAll(err.Error(), "\n", " | ")
		}
		failures := InterceptGomegaFailures(func() {
			h := newRuntimeHarness(ctx, program, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.StringT},
				channels.Digest{Key: 102, DataType: telem.Float64T},
				channels.Digest{Key: 103, DataType: telem.Float64T},
			)
			defer h.Close(ctx)
			collect := func() {
				out, _ := h.Flush()
				for _, s := range out.Get(101).Series {
					logged = append(logged, s.Unmarshal[string]()...)
				}
			}
			for range 3 {
				h.Ingest(102, telem.NewSeriesV(1.23))
				h.Ingest(103, telem.NewSeriesV(9.9))
				h.Ingest(100, telem.NewSeriesV[uint8](1))
				for range 5 {
					h.Tick(ctx, telem.Millisecond)
					h.channelState.ClearReads()
					collect()
				}
			}
		})
		if len(failures) > 0 {
			return logged, "RUNTIME: " + strings.Join(failures, " | ")
		}
		return logged, ""
	}

	It("hunts", func(ctx SpecContext) {
		var anomalies, rejected, silent []string
		total := 0
		for _, l := range layouts {
			for _, v := range variables {
				if l.declTopLevel && !v.topLevel {
					continue
				}
				for _, u := range uses {
					if l.useTopLevel && !u.topLevel {
						continue
					}
					total++
					program := prelude + l.build(v.decl, u.lines)
					label := fmt.Sprintf("[%s] [%s] [%s]", l.name, v.name, u.name)
					logged, failure := run(ctx, program)
					if failure != "" {
						rejected = append(rejected, label+" "+failure)
						continue
					}
					if len(logged) == 0 {
						silent = append(silent, label)
						continue
					}
					accept := make([]string, len(v.accept))
					for i, a := range v.accept {
						accept[i] = u.wrap(a)
					}
					distinct := slices.Compact(slices.Sorted(slices.Values(logged)))
					bad := false
					for _, got := range distinct {
						if !slices.Contains(accept, got) {
							bad = true
						}
					}
					if bad || !slices.Contains(distinct, accept[0]) {
						anomalies = append(anomalies, fmt.Sprintf(
							"%s logged %q, accept %q\n%s\n", label, distinct, accept, program,
						))
					}
				}
			}
		}
		GinkgoWriter.Printf("TOTAL %d\n", total)
		GinkgoWriter.Printf("\n=== ANOMALIES (%d) ===\n", len(anomalies))
		for _, a := range anomalies {
			GinkgoWriter.Println(a)
		}
		GinkgoWriter.Printf("\n=== SILENT (%d) ===\n", len(silent))
		for _, s := range silent {
			GinkgoWriter.Println(s)
		}
		GinkgoWriter.Printf("\n=== REJECTED (%d) ===\n", len(rejected))
		for _, r := range rejected {
			GinkgoWriter.Println(r)
		}
		Expect(anomalies).To(BeEmpty())
	})
})
