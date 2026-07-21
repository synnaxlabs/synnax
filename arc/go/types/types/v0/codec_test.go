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
	v0 "github.com/synnaxlabs/arc/types/types/v0"
	"github.com/synnaxlabs/x/encoding/orc"
)

var _ = Describe("Codec", func() {
	DescribeTable("should round-trip encode and decode types",
		func(original v0.Type) {
			w := orc.NewWriter(0)
			Expect(original.EncodeOrc(w)).To(Succeed())
			var decoded v0.Type
			r := orc.NewReader(nil)
			r.ResetBytes(w.Bytes())
			Expect(decoded.DecodeOrc(r)).To(Succeed())
			Expect(decoded).To(Equal(original))
		},
		Entry("scalar", v0.Type{Kind: v0.KindF64, Name: "f64"}),
		Entry("channel with direction", v0.Type{
			Kind:          v0.KindChan,
			Name:          "chan",
			Elem:          &v0.Type{Kind: v0.KindF32},
			ChanDirection: v0.ChanDirectionRead,
		}),
		Entry("zero value", v0.Type{}),
	)
})
