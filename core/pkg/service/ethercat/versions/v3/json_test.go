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
	Describe("ReadChannel", func() {
		channels := []v3.ReadChannel{
			{Variant: v3.EntryReadChannel{
				BaseReadChannel: v3.BaseReadChannel{Key: "a", Channel: 5, Device: "d"},
				Index:           0x6000,
				SubIndex:        1,
				DataType:        telem.Int16T,
				Scale: v3.Scale{
					Variant: v3.LinearScale{Slope: 2, Offset: 1},
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
			 "scale": {"type": "linear", "slope": 2, "offset": 1}},
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
})
