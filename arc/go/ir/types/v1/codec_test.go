// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/arc/ir/types/v1"
	"github.com/synnaxlabs/x/encoding/orc"
)

var _ = Describe("Codec", func() {
	Describe("Edge", func() {
		DescribeTable("should round-trip encode and decode",
			func(original v1.Edge) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v1.Edge
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("fully populated", v1.Edge{
				Source: v1.Handle{Node: "sensor", Param: "output"},
				Target: v1.Handle{Node: "valve", Param: "command"},
				Kind:   v1.EdgeKindConditional,
			}),
			Entry("zero value", v1.Edge{}),
		)
	})

	Describe("Transition", func() {
		DescribeTable("should round-trip encode and decode",
			func(original v1.Transition) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v1.Transition
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("with target", v1.Transition{
				On:        v1.Handle{Node: "n1", Param: "done"},
				TargetKey: new(string("s2")),
			}),
			Entry("exit transition", v1.Transition{
				On: v1.Handle{Node: "n1", Param: "done"},
			}),
		)
	})
})
