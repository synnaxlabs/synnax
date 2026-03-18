// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel/calculation/status"
	xstatus "github.com/synnaxlabs/x/status"
)

var _ = Describe("Status", func() {
	Describe("Key", func() {
		It("Should return a consistent key for a given channel key", func() {
			k := status.Key(channel.Key(42))
			Expect(k).To(Equal(channel.OntologyID(42).String()))
		})

		It("Should return different keys for different channel keys", func() {
			Expect(status.Key(channel.Key(1))).ToNot(Equal(status.Key(channel.Key(2))))
		})
	})

	Describe("Error", func() {
		It("Should build a status with the correct fields", func() {
			st := status.Error(
				channel.Key(42),
				"my_channel",
				"expression parse failed",
				errors.New("unexpected token"),
			)
			Expect(st.Key).To(Equal(status.Key(42)))
			Expect(st.Name).To(Equal("my_channel"))
			Expect(st.Variant).To(Equal(xstatus.VariantError))
			Expect(st.Message).To(Equal("expression parse failed"))
			Expect(st.Description).To(Equal("unexpected token"))
			Expect(st.Details.Channel).To(Equal(channel.Key(42)))
			Expect(st.Time).ToNot(BeZero())
		})
	})
})
