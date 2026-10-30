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
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/v2"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("JSON", func() {
	Describe("ReadChannel", func() {
		channels := []v2.ReadChannel{
			{Variant: v2.EntryReadChannel{
				BaseReadChannel: v2.BaseReadChannel{Key: "a", Channel: 5, Device: "d"},
				Index:           0x6000,
				SubIndex:        1,
				DataType:        telem.Int16T,
			}},
			{Variant: v2.UnresolvedReadChannel{
				BaseReadChannel: v2.BaseReadChannel{Key: "b", Channel: 6, Device: "d"},
				Pdo:             "Input",
			}},
		}
		encoded := `[
			{"type": "entry", "key": "a", "name": "", "disabled": false, "channel": 5,
			 "device": "d", "index": 24576, "sub_index": 1, "data_type": "int16"},
			{"type": "unresolved", "key": "b", "name": "", "disabled": false,
			 "channel": 6, "device": "d", "pdo": "Input"}
		]`

		It("Should encode each channel with its type", func() {
			Expect(MustSucceed(json.Marshal(channels))).To(MatchJSON(encoded))
		})

		It("Should decode each channel by its type", func() {
			var decoded []v2.ReadChannel
			Expect(json.Unmarshal([]byte(encoded), &decoded)).To(Succeed())
			Expect(decoded).To(Equal(channels))
		})

		It("Should reject a channel with an unknown type", func() {
			var decoded []v2.ReadChannel
			Expect(json.Unmarshal([]byte(`[{"type": "manual"}]`), &decoded)).
				To(MatchError(ContainSubstring(`ReadChannel: unknown type "manual"`)))
		})
	})

	Describe("WriteChannel", func() {
		channels := []v2.WriteChannel{
			{Variant: v2.EntryWriteChannel{
				BaseWriteChannel: v2.BaseWriteChannel{
					Key:          "a",
					CmdChannel:   5,
					StateChannel: 6,
					Device:       "d",
				},
				Index:    0x7000,
				SubIndex: 1,
				DataType: telem.Uint8T,
			}},
			{Variant: v2.UnresolvedWriteChannel{
				BaseWriteChannel: v2.BaseWriteChannel{
					Key:        "b",
					CmdChannel: 7,
					Device:     "d",
				},
				Pdo: "Output",
			}},
		}
		encoded := `[
			{"type": "entry", "key": "a", "name": "", "disabled": false,
			 "cmd_channel": 5, "state_channel": 6, "cmd_channel_name": "",
			 "state_channel_name": "", "device": "d", "index": 28672, "sub_index": 1,
			 "data_type": "uint8"},
			{"type": "unresolved", "key": "b", "name": "", "disabled": false,
			 "cmd_channel": 7, "state_channel": 0, "cmd_channel_name": "",
			 "state_channel_name": "", "device": "d", "pdo": "Output"}
		]`

		It("Should encode each channel with its type", func() {
			Expect(MustSucceed(json.Marshal(channels))).To(MatchJSON(encoded))
		})

		It("Should decode each channel by its type", func() {
			var decoded []v2.WriteChannel
			Expect(json.Unmarshal([]byte(encoded), &decoded)).To(Succeed())
			Expect(decoded).To(Equal(channels))
		})

		It("Should reject a channel with an unknown type", func() {
			var decoded []v2.WriteChannel
			Expect(json.Unmarshal([]byte(`[{"type": "manual"}]`), &decoded)).
				To(MatchError(ContainSubstring(`WriteChannel: unknown type "manual"`)))
		})
	})
})
