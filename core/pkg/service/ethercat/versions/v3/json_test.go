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
	task "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("JSON", func() {
	Describe("ReadChannel", func() {
		channels := []v3.ReadChannel{
			{Variant: v3.EntryReadChannel{
				BaseReadChannel: v3.BaseReadChannel{Key: "a", Channel: 5, Device: "d"},
				Index:           0x6000,
				SubIndex:        1,
				DataType:        telem.Int16T,
				Calibrated:      true,
				Scale: task.Scale{
					Variant: task.LinearScale{Slope: 2, Offset: 1},
				},
			}},
			{Variant: v3.UnresolvedReadChannel{
				BaseReadChannel: v3.BaseReadChannel{Key: "b", Channel: 6, Device: "d"},
				Pdo:             "Input",
			}},
		}
		encoded := `[
			{"type": "entry", "key": "a", "name": "", "disabled": false, "channel": 5,
			 "device": "d", "index": 24576, "sub_index": 1, "data_type": "int16",
			 "calibrated": true, "scale": {"type": "linear", "slope": 2, "offset": 1}},
			{"type": "unresolved", "key": "b", "name": "", "disabled": false,
			 "channel": 6, "device": "d", "pdo": "Input"}
		]`

		It("Should encode each channel with its type", func() {
			Expect(MustSucceed(json.Marshal(channels))).To(MatchJSON(encoded))
		})

		It("Should decode each channel by its type", func() {
			var decoded []v3.ReadChannel
			Expect(json.Unmarshal([]byte(encoded), &decoded)).To(Succeed())
			Expect(decoded).To(Equal(channels))
		})
	})

	Describe("SlaveStartupParameters", func() {
		parameters := []v3.SlaveStartupParameters{{
			Device: "d",
			Parameters: []v3.StartupParameter{
				{Index: 0x8040, SubIndex: 1, DataType: telem.Uint16T, Value: 6},
			},
		}}
		encoded := `[{"device": "d", "parameters": [{
			"index": 32832, "sub_index": 1, "data_type": "uint16", "value": 6,
			"name": ""
		}]}]`

		It("Should encode the parameters of each slave", func() {
			Expect(MustSucceed(json.Marshal(parameters))).To(MatchJSON(encoded))
		})

		It("Should decode the parameters of each slave", func() {
			var decoded []v3.SlaveStartupParameters
			Expect(json.Unmarshal([]byte(encoded), &decoded)).To(Succeed())
			Expect(decoded).To(Equal(parameters))
		})
	})

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
