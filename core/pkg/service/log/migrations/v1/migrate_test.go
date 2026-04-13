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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v1"
	. "github.com/synnaxlabs/x/testutil"
)

func TestV1(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Log Migrations V1 Suite")
}

var _ = Describe("Migrate", func() {
	It("Should convert bare channel keys to config entries with defaults", func() {
		old := v0.Data{
			Channels:      []int{1, 2, 3},
			RemoteCreated: false,
		}
		result := MustSucceed(v1.Migrate(old))
		Expect(result.Channels).To(HaveLen(3))
		Expect(result.Channels[0].Channel).To(Equal(1))
		Expect(result.Channels[0].Color).To(Equal(""))
		Expect(result.Channels[0].Notation).To(Equal("standard"))
		Expect(result.Channels[0].Precision).To(Equal(-1))
		Expect(result.Channels[0].Alias).To(Equal(""))
		Expect(result.Channels[2].Channel).To(Equal(3))
	})

	It("Should preserve RemoteCreated", func() {
		old := v0.Data{Channels: []int{}, RemoteCreated: true}
		result := MustSucceed(v1.Migrate(old))
		Expect(result.RemoteCreated).To(BeTrue())
	})

	It("Should set correct v1 defaults", func() {
		old := v0.Data{Channels: []int{}, RemoteCreated: false}
		result := MustSucceed(v1.Migrate(old))
		Expect(result.TimestampPrecision).To(Equal(0))
		Expect(result.ShowChannelNames).To(BeTrue())
		Expect(result.ShowReceiptTimestamp).To(BeTrue())
	})

	It("Should handle empty channels", func() {
		old := v0.Data{Channels: []int{}, RemoteCreated: false}
		result := MustSucceed(v1.Migrate(old))
		Expect(result.Channels).To(HaveLen(0))
	})
})
