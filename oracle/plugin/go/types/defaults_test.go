// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/synnaxlabs/oracle/plugin/go/types"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("ApplyDefaults and Validate generation", func() {
	var (
		loader   *MockFileLoader
		goPlugin *types.Plugin
	)
	BeforeEach(func() {
		loader = NewMockFileLoader()
		goPlugin = types.New(types.DefaultOptions())
	})

	It("Should generate ApplyDefaults filling non-zero static defaults", func(ctx SpecContext) {
		source := `
			@go output "core/pkg/service/x"

			Level enum {
				h1 = "h1"
				h2 = "h2"
			}

			Cfg struct {
				rolling int32   = 1
				scale   float64 = 1.5
				name    string  = "untitled"
				level   Level   = LevelH2
			}
		`
		resp := MustGenerate(ctx, source, "x", loader, goPlugin)
		ExpectContent(resp, "types.gen.go").ToContain(
			"func (c Cfg) ApplyDefaults() Cfg {",
			"c.Rolling = 1",
			"c.Scale = 1.5",
			`c.Name = "untitled"`,
			"c.Level = LevelH2",
			"return c",
		)
	})

	It("Should generate Validate asserting enum membership", func(ctx SpecContext) {
		source := `
			@go output "core/pkg/service/x"

			Level enum {
				h1 = "h1"
				h2 = "h2"
			}

			Cfg struct {
				level Level = LevelH2
			}
		`
		resp := MustGenerate(ctx, source, "x", loader, goPlugin)
		ExpectContent(resp, "types.gen.go").ToContain(
			`"github.com/synnaxlabs/x/validate"`,
			"func (c Cfg) Validate() error {",
			`validate.New("Cfg")`,
			"!c.Level.IsValid()",
		)
	})

	It("Should not generate ApplyDefaults when every default equals the zero value", func(ctx SpecContext) {
		source := `
			@go output "core/pkg/service/x"

			Cfg struct {
				name  string = ""
				count int32 = 0
			}
		`
		resp := MustGenerate(ctx, source, "x", loader, goPlugin)
		ExpectContent(resp, "types.gen.go").ToNotContain("ApplyDefaults")
	})

	It("Should key enum validation by the wire field name, not the Go name", func(ctx SpecContext) {
		source := `
			@go output "core/pkg/service/x"

			Level enum {
				h1 = "h1"
				h2 = "h2"
			}

			Cfg struct {
				label_level Level = LevelH2
			}
		`
		resp := MustGenerate(ctx, source, "x", loader, goPlugin)
		ExpectContent(resp, "types.gen.go").ToContain(
			`v.Ternaryf("label_level"`,
			"!c.LabelLevel.IsValid()",
			"invalid label_level: %v",
		)
	})
})
