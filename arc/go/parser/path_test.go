// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package parser_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/parser"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Bare paths", func() {
	primaryName := func(src string) parser.Name {
		GinkgoHelper()
		return parser.PrimaryName(
			parser.GetPrimaryExpression(MustSucceed(parser.ParseExpression(src))),
		)
	}
	// name matches a qualified name by its segments and its joined text.
	name := func(head, tail string) OmegaMatcher {
		return And(
			HaveField("Head", head),
			HaveField("Tail", tail),
			WithTransform(parser.Name.String, Equal(head+"."+tail)),
		)
	}

	DescribeTable("Should read every segment of a path in an expression",
		func(src, head, tail string) {
			Expect(primaryName(src)).To(name(head, tail))
		},
		Entry("two segments", "stand.pt_01", "stand", "pt_01"),
		Entry("three segments", "stand.sensors.pt_01", "stand", "sensors.pt_01"),
		Entry("five segments", "a.b.c.d.e", "a", "b.c.d.e"),
		Entry("spaces around the dots", "a . b . c", "a", "b.c"),
	)

	It("Should read a path at a flow node", func() {
		prog := MustSucceed(parser.Parse("stand.sensors.pt_01 -> stand.out"))
		nodes := prog.AllTopLevelItem()[0].FlowStatement().AllFlowNode()
		Expect(parser.IdentifierName(nodes[0].Identifier())).
			To(name("stand", "sensors.pt_01"))
		Expect(parser.IdentifierName(nodes[1].Identifier())).
			To(name("stand", "out"))
	})

	DescribeTable("Should read a path at an assignment target",
		func(src string) {
			stmt := MustSucceed(parser.ParseStatement(src))
			Expect(parser.AssignmentName(stmt.Assignment())).
				To(name("stand", "valves.cmd"))
		},
		Entry("an assignment", "stand.valves.cmd = 1"),
		Entry("a compound assignment", "stand.valves.cmd += 1"),
	)

	It("Should read a path at an authority entry", func() {
		prog := MustSucceed(parser.Parse("authority (stand.valves.cmd 200)"))
		entry := prog.AllTopLevelItem()[0].AuthorityBlock().AllAuthorityEntry()[0]
		Expect(parser.AuthorityEntryName(entry)).
			To(name("stand", "valves.cmd"))
	})

	DescribeTable("Should mark a qualified name that a call follows",
		func(src string, callee bool) {
			Expect(primaryName(src).Callee).To(Equal(callee))
		},
		Entry("a module call", "math.avg(x)", true),
		Entry("a call of three segments", "a.b.c(x)", true),
		Entry("a module constant", "math.inf", false),
		Entry("a path", "stand.pt_01", false),
		Entry("a path that is indexed", "stand.pt_01[0]", false),
	)

	It("Should not mark a bare name that a call follows", func() {
		Expect(primaryName("avg(x)").Callee).To(BeFalse())
	})

	DescribeTable("Should refuse a keyword segment that its form does not allow",
		func(src string) {
			_, diag := parser.Parse(src)
			Expect(diag.String()).To(ContainSubstring("syntax error"))
		},
		Entry("for in the middle", "a.for.b -> out"),
		Entry("authority after the first segment", "a.authority -> out"),
	)
})
