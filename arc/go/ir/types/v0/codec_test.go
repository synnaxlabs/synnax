// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/arc/ir/types/v0"
	"github.com/synnaxlabs/x/encoding/orc"
)

var _ = Describe("Codec", func() {
	DescribeTable("should round-trip encode and decode edges",
		func(original v0.Edge) {
			w := orc.NewWriter(0)
			Expect(original.EncodeOrc(w)).To(Succeed())
			var decoded v0.Edge
			r := orc.NewReader(nil)
			r.ResetBytes(w.Bytes())
			Expect(decoded.DecodeOrc(r)).To(Succeed())
			Expect(decoded).To(Equal(original))
		},
		Entry("fully populated", v0.Edge{
			Source: v0.Handle{Node: "sensor", Param: "output"},
			Target: v0.Handle{Node: "valve", Param: "command"},
			Kind:   v0.EdgeKindContinuous,
		}),
		Entry("zero value", v0.Edge{}),
	)
})
