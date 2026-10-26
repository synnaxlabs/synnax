// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package surface_test

import (
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/surface"
	"github.com/synnaxlabs/oracle/testutil"
)

func report(ctx context.Context, source string) string {
	GinkgoHelper()
	table, diag := analyzer.AnalyzeSource(
		ctx, source, "widget", testutil.NewMockFileLoader(),
	)
	Expect(diag.Ok()).To(BeTrue(), diag.String())
	var b strings.Builder
	Expect(surface.Write(&b, table)).To(Succeed())
	return b.String()
}

var _ = Describe("Write", func() {
	It("Should write structs, fields, enums, and unions", func(ctx SpecContext) {
		Expect(report(ctx, `
Size enum {
	small = "small"
	large = "large"
}

Labeled struct {
	label string = "valve"
	tags string[]?
}

Valve struct extends Labeled {
	size Size
}

Part union on kind {
	valve Valve
}
`)).To(Equal(`widget.Labeled struct
widget.Labeled.label string = "valve"
widget.Labeled.tags? Array<string>
widget.Part union on kind
widget.Part|valve widget.Valve
widget.Size enum
widget.Size.small = "small"
widget.Size.large = "large"
widget.Valve struct extends widget.Labeled
widget.Valve.size widget.Size
`))
	})

	It("Should ignore docs and formatting", func(ctx SpecContext) {
		Expect(report(ctx, `
// A thing that holds a name.
Valve struct {
	// The name of the valve.
	name   string
}
`)).To(Equal(report(ctx, "Valve struct { name string }")))
	})

	It("Should change when a field changes", func(ctx SpecContext) {
		Expect(report(ctx, "Valve struct { name string }")).
			ToNot(Equal(report(ctx, "Valve struct { name string? }")))
	})
})
