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
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/labjack/device/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Ports", func() {
	DescribeTable("Should keep the model through msgpack",
		func(ports v0.Ports) {
			var got v0.Ports
			Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(ports)), &got)).
				To(Succeed())
			Expect(got).To(Equal(ports))
		},
		Entry("T4", v0.Ports{Variant: v0.T4Ports{}}),
		Entry("T7", v0.Ports{Variant: v0.T7Ports{Mux80Attached: true}}),
		Entry("T8", v0.Ports{Variant: v0.T8Ports{}}),
		Entry("no model", v0.Ports{}),
	)

	It("Should match the JSON shape in msgpack", func() {
		ports := v0.Ports{Variant: v0.T7Ports{Mux80Attached: true}}
		var wire, fromJSON map[string]any
		Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(ports)), &wire)).
			To(Succeed())
		Expect(json.Unmarshal(MustSucceed(json.Marshal(ports)), &fromJSON)).
			To(Succeed())
		Expect(wire).To(Equal(fromJSON))
	})

	It("Should reject an unknown model in msgpack", func() {
		raw := MustSucceed(msgpack.Marshal(map[string]any{"model": "t9"}))
		var got v0.Ports
		Expect(msgpack.Unmarshal(raw, &got)).To(MatchError(validate.ErrValidation))
	})
})
