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
	"reflect"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v1"
)

var _ = Describe("Data", func() {
	Describe("ToMap", func() {
		It("Should produce a key for every json-tagged field on Data", func() {
			m := v1.Data{}.ToMap()
			t := reflect.TypeOf(v1.Data{})
			for i := range t.NumField() {
				tag := strings.Split(t.Field(i).Tag.Get("json"), ",")[0]
				Expect(m).To(HaveKey(tag), "field %s missing from Data.ToMap", tag)
			}
		})

		It("Should preserve typed values without coercing through JSON", func() {
			d := v1.Data{
				Channels:             []v1.ChannelEntry{{Channel: 7, Color: "red"}},
				RemoteCreated:        true,
				TimestampPrecision:   3,
				ShowChannelNames:     false,
				ShowReceiptTimestamp: true,
			}
			m := d.ToMap()
			Expect(m["channels"]).To(Equal(d.Channels))
			Expect(m["remote_created"]).To(Equal(true))
			Expect(m["timestamp_precision"]).To(Equal(3))
			Expect(m["show_channel_names"]).To(Equal(false))
			Expect(m["show_receipt_timestamp"]).To(Equal(true))
		})
	})
})
