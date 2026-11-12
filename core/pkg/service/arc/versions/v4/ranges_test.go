// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
	"github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
)

// analyze returns the diagnostics of the strict analyzer for src.
func analyze(ctx SpecContext, src string) *diagnostics.Diagnostics {
	GinkgoHelper()
	t := MustSucceed(text.Parse(text.Text{Raw: src}))
	_, diag := text.Analyze(ctx, t, symbol.NewRoot(nil, stl.NewSymbols()))
	return diag
}

func wrap(body string) string {
	return "func f(n i32, m u64, b u8) {\n" + body + "\n}"
}

var _ = Describe("CastRanges", func() {
	DescribeTable("Should cast mixed typed arguments to the type 0.60 picked",
		func(ctx SpecContext, old, want string) {
			Expect(analyze(ctx, wrap(old)).String()).To(ContainSubstring("in range()"))
			got := v4.CastRanges(ctx, wrap(old))
			Expect(got).To(Equal(wrap(want)))
			diag := analyze(ctx, got)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
		},
		Entry("the issue's program",
			"for i := range(n, i64(100)) {}",
			"for i := range(i64(n), i64(100)) {}"),
		Entry("signed and unsigned of one width",
			"for i := range(i64(0), m) {}",
			"for i := range(i64(0), i64(m)) {}"),
		Entry("a narrow signed and a wide unsigned",
			"for i := range(n, m) {}",
			"for i := range(i64(n), i64(m)) {}"),
		Entry("unsigned arguments only",
			"for i := range(b, u32(10)) {}",
			"for i := range(u32(b), u32(10)) {}"),
		Entry("an untyped step",
			"for i := range(n, i64(100), 2) {}",
			"for i := range(i64(n), i64(100), 2) {}"),
		Entry("an expression argument",
			"for i := range(n + 1, i64(100)) {}",
			"for i := range(i64(n + 1), i64(100)) {}"),
		Entry("nested calls",
			"for i := range(n, i64(3)) {\n    for j := range(b, n) {}\n}",
			"for i := range(i64(n), i64(3)) {\n    for j := range(i32(b), n) {}\n}"),
		Entry("a nested call that reads a widened loop variable",
			"for i := range(n, i64(3)) {\n    for j := range(i, n) {}\n}",
			"for i := range(i64(n), i64(3)) {\n    for j := range(i, i64(n)) {}\n}"),
		Entry("a nested call that casts a widened loop variable once",
			"for i := range(n, i64(3)) {\n    for j := range(i, b) {}\n}",
			"for i := range(i64(n), i64(3)) {\n    for j := range(i, i64(b)) {}\n}"),
		Entry("several calls",
			"for i := range(n, i64(3)) {}\nfor j := range(i64(1), b) {}",
			"for i := range(i64(n), i64(3)) {}\nfor j := range(i64(1), i64(b)) {}"),
	)

	DescribeTable("Should keep a program with no mixed range() byte for byte",
		func(ctx SpecContext, src string) {
			Expect(v4.CastRanges(ctx, wrap(src))).To(Equal(wrap(src)))
		},
		Entry("untyped literals", "for i := range(0, 10, 2) {}"),
		Entry("one typed argument and literals", "for i := range(0, n, 2) {}"),
		Entry("arguments of one type", "for i := range(n, i32(10)) {}"),
		Entry("a series loop", "for i, v := [1, 2] {}"),
		Entry("a mixed call with an argument that does not resolve",
			"for i := range(n, sensor) {}"),
	)

	It("Should cast the range() calls of a program with dashed names", func(
		ctx SpecContext,
	) {
		src := "func my-fn(n i32) {\n    for i := range(n, i64(100)) {}\n}"
		Expect(v4.CastRanges(ctx, src)).To(Equal(src))
		Expect(v4.CastRanges(ctx, src, parser.Config{AllowDashedNames: true})).
			To(Equal("func my-fn(n i32) {\n    for i := range(i64(n), i64(100)) {}\n}"))
	})

	It("Should leave a program that does not parse unchanged", func(ctx SpecContext) {
		src := "func f( {\n    for i := range(n, i64(100)) {}\n"
		Expect(v4.CastRanges(ctx, src)).To(Equal(src))
	})
})
