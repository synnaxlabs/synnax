// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel/calculation"
	xstatus "github.com/synnaxlabs/x/status"
)

var _ = Describe("Status", func() {
	Describe("StatusKey", func() {
		It("Should return a consistent key for a given channel key", func() {
			k := calculation.StatusKey(channel.Key(42))
			Expect(k).To(Equal(channel.OntologyID(42).String()))
		})

		It("Should return different keys for different channel keys", func() {
			Expect(calculation.StatusKey(channel.Key(1))).ToNot(Equal(calculation.StatusKey(channel.Key(2))))
		})
	})

	Describe("StatusFromError", func() {
		It("Should build a status with the correct fields", func() {
			st := calculation.StatusFromError(
				channel.Key(42),
				"my_channel",
				"expression parse failed",
				errors.New("unexpected token"),
			)
			Expect(st.Key).To(Equal(calculation.StatusKey(42)))
			Expect(st.Name).To(Equal("my_channel"))
			Expect(st.Variant).To(Equal(xstatus.VariantError))
			Expect(st.Message).To(Equal("expression parse failed"))
			Expect(st.Description).To(Equal("unexpected token"))
			Expect(st.Time).ToNot(BeZero())
		})
	})
})
