// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schemadiff_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin/go/internal/schemadiff"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("StructurallyEqual", func() {
	analyze := func(source string) *resolution.Table {
		table := resolution.NewTable()
		diag := analyzer.AnalyzeSeeded(
			GinkgoT().Context(), source,
			"schemas/synnax/channel.oracle", "channel",
			testutil.NewMockFileLoader(), table,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		return table
	}
	typeOf := func(t *resolution.Table, name string) resolution.Type {
		typ, ok := t.Get("channel." + name)
		Expect(ok).To(BeTrue())
		return typ
	}

	It("Should equate identical declarations", func() {
		a := analyze("Channel struct {\n\tkey uuid @key\n\tname string\n}\n")
		b := analyze("Channel struct {\n\tkey uuid @key\n\tname string\n}\n")
		Expect(schemadiff.StructurallyEqual(
			typeOf(a, "Channel"), typeOf(b, "Channel"), a, b,
		)).To(BeTrue())
	})

	It("Should distinguish declarations by field list", func() {
		a := analyze("Channel struct {\n\tkey uuid @key\n}\n")
		b := analyze("Channel struct {\n\tkey uuid @key\n\tname string\n}\n")
		Expect(schemadiff.StructurallyEqual(
			typeOf(a, "Channel"), typeOf(b, "Channel"), a, b,
		)).To(BeFalse())
	})

	It("Should distinguish declarations by field marshal value", func() {
		a := analyze("Channel struct {\n\tkey uuid @key\n\tname string\n}\n")
		b := analyze(
			"Channel struct {\n\tkey uuid @key\n\tname string {\n" +
				"\t\t@go marshal omit\n\t}\n}\n",
		)
		Expect(schemadiff.StructurallyEqual(
			typeOf(a, "Channel"), typeOf(b, "Channel"), a, b,
		)).To(BeFalse())
	})

	It("Should distinguish enums by member list", func() {
		a := analyze("State enum {\n\tidle = 0\n}\n")
		b := analyze("State enum {\n\tidle = 0\n\trunning = 1\n}\n")
		Expect(schemadiff.StructurallyEqual(
			typeOf(a, "State"), typeOf(b, "State"), a, b,
		)).To(BeFalse())
	})

	It("Should equate identical enums", func() {
		a := analyze("State enum {\n\tidle = 0\n\trunning = 1\n}\n")
		b := analyze("State enum {\n\tidle = 0\n\trunning = 1\n}\n")
		Expect(schemadiff.StructurallyEqual(
			typeOf(a, "State"), typeOf(b, "State"), a, b,
		)).To(BeTrue())
	})
})
