// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/v3"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("JSON", func() {
	Describe("WriteChannel", func() {
		channels := []v3.WriteChannel{
			{Variant: v3.EntryWriteChannel{
				BaseWriteChannel: v3.BaseWriteChannel{
					Key:          "a",
					CmdChannel:   5,
					StateChannel: 6,
					Device:       "d",
				},
				Index:     0x7000,
				SubIndex:  1,
				DataType:  telem.Uint8T,
				StopValue: 3,
			}},
			{Variant: v3.UnresolvedWriteChannel{
				BaseWriteChannel: v3.BaseWriteChannel{
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
			 "data_type": "uint8", "stop_value": 3},
			{"type": "unresolved", "key": "b", "name": "", "disabled": false,
			 "cmd_channel": 7, "state_channel": 0, "cmd_channel_name": "",
			 "state_channel_name": "", "device": "d", "pdo": "Output"}
		]`

		It("Should encode each channel with its type", func() {
			Expect(MustSucceed(json.Marshal(channels))).To(MatchJSON(encoded))
		})

		It("Should decode each channel by its type", func() {
			var decoded []v3.WriteChannel
			Expect(json.Unmarshal([]byte(encoded), &decoded)).To(Succeed())
			Expect(decoded).To(Equal(channels))
		})
	})
})
