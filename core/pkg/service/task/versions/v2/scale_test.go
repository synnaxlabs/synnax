// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Scale", func() {
	decode := func(value any) (v2.Scale, error) {
		var s v2.Scale
		err := msgpack.Unmarshal(MustSucceed(msgpack.Marshal(value)), &s)
		return s, err
	}

	DescribeTable("Should round trip a scale through msgpack",
		func(s v2.Scale) {
			var out v2.Scale
			Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(s)), &out)).
				To(Succeed())
			Expect(out).To(Equal(s))
		},
		Entry("no variant", v2.Scale{}),
		Entry("a linear scale", v2.Scale{
			Variant: v2.LinearScale{Slope: 2, Offset: 3},
		}),
		Entry("a map scale", v2.Scale{Variant: v2.MapScale{
			PreScaledMin: 1, PreScaledMax: 2, ScaledMin: 3, ScaledMax: 4,
		}}),
	)

	It("Should fill the default of a field that msgpack input leaves out", func() {
		Expect(decode(map[string]any{"type": "linear"})).
			To(Equal(v2.Scale{Variant: v2.LinearScale{Slope: 1}}))
	})

	It("Should fill a default group that msgpack input leaves out", func() {
		Expect(decode(map[string]any{"type": "map", "scaled_max": 5.0})).
			To(Equal(v2.Scale{Variant: v2.MapScale{PreScaledMax: 1, ScaledMax: 5}}))
	})

	It("Should refuse an unknown type", func() {
		Expect(decode(map[string]any{"type": "cubic"})).Error().
			To(MatchError(validate.ErrValidation))
	})
})
