// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device_test

import (
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/labjack/device"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Update", func() {
	identity := device.Identity{
		Rack:       1,
		Serial:     470010001,
		Connection: device.ConnectionWifi,
		Model:      device.ModelT7,
	}
	configuration := device.Configuration{
		Name:            "T7",
		Identifier:      "lj",
		ReadIndex:       1,
		WriteStateIndex: 2,
		Ports:           device.Ports{Variant: device.T7Ports{Mux80Attached: true}},
	}

	DescribeTable("Should keep the group through JSON",
		func(update device.Update) {
			var got device.Update
			Expect(json.Unmarshal(
				MustSucceed(json.Marshal(update)),
				&got,
			)).To(Succeed())
			Expect(got).To(Equal(update))
		},
		Entry("identity", device.Update{
			Variant: device.IdentityUpdate{Identity: identity},
		}),
		Entry("configuration", device.Update{
			Variant: device.ConfigurationUpdate{Configuration: configuration},
		}),
	)

	DescribeTable("Should reject an unknown discriminator in JSON",
		func(raw string) {
			var got device.Update
			Expect(json.Unmarshal([]byte(raw), &got)).
				To(MatchError(validate.ErrValidation))
		},
		Entry("group", `{"group":"other"}`),
		Entry("model", `{"group":"configuration","name":"T7","ports":{"model":"t9"}}`),
	)

	DescribeTable("Should reject an invalid update",
		func(update device.Update, field string) {
			Expect(update.Validate()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(field)),
			))
		},
		Entry("with no group", device.Update{}, "group"),
		Entry("with no rack", device.Update{Variant: device.IdentityUpdate{
			Identity: device.Identity{
				Serial:     1,
				Connection: device.ConnectionUSB,
				Model:      device.ModelT7,
			},
		}}, "rack"),
		Entry("with no serial", device.Update{Variant: device.IdentityUpdate{
			Identity: device.Identity{
				Rack:       1,
				Connection: device.ConnectionUSB,
				Model:      device.ModelT7,
			},
		}}, "serial"),
		Entry("with an unknown connection", device.Update{
			Variant: device.IdentityUpdate{Identity: device.Identity{
				Rack:       1,
				Serial:     1,
				Connection: "LJM_ctTCP",
				Model:      device.ModelT7,
			}},
		}, "connection"),
		Entry("with an unknown model", device.Update{
			Variant: device.IdentityUpdate{Identity: device.Identity{
				Rack:       1,
				Serial:     1,
				Connection: device.ConnectionUSB,
				Model:      "t9",
			}},
		}, "model"),
		Entry("with no name", device.Update{Variant: device.ConfigurationUpdate{
			Configuration: device.Configuration{
				Ports: device.Ports{Variant: device.T7Ports{}},
			},
		}}, "name"),
		Entry("with no ports", device.Update{Variant: device.ConfigurationUpdate{
			Configuration: device.Configuration{Name: "T7"},
		}}, "ports"),
	)

	DescribeTable("Should check the identifier",
		func(identifier string, valid bool) {
			c := configuration
			c.Identifier = identifier
			err := device.Update{
				Variant: device.ConfigurationUpdate{Configuration: c},
			}.Validate()
			if valid {
				Expect(err).ToNot(HaveOccurred())
				return
			}
			Expect(err).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("Identifier must be 2 to 12 characters")),
			))
		},
		Entry("empty", "", true),
		Entry("two characters", "lj", true),
		Entry("twelve characters with digits and underscores", "LabJack_T7_1", true),
		Entry("one character", "l", false),
		Entry("thirteen characters", "LabJack_T7_12", false),
		Entry("a leading digit", "7lj", false),
		Entry("a leading underscore", "_lj", false),
		Entry("a hyphen", "lab-jack", false),
	)

	It("Should accept a valid update", func() {
		Expect(device.Update{
			Variant: device.ConfigurationUpdate{Configuration: configuration},
		}.Validate()).To(Succeed())
	})
})
