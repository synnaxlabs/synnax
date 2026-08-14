// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package legacy_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/modbus/versions/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

func channels(types ...string) msgpack.EncodedJSON {
	chs := make([]any, len(types))
	for i, t := range types {
		chs[i] = map[string]any{"type": t}
	}
	return msgpack.EncodedJSON{"channels": chs}
}

var _ = Describe("Read", func() {
	DescribeTable("channel type tags",
		func(in, want string) {
			Expect(legacy.Read.Apply(channels(in))).To(Equal(channels(want)))
		},
		Entry("coil_input", "coil_input", "coil"),
		Entry("discrete_input unchanged", "discrete_input", "discrete_input"),
		Entry("holding_register_input", "holding_register_input", "holding_register"),
		Entry("register_input", "register_input", "input_register"),
		Entry("canonical coil is a fixed point", "coil", "coil"),
		Entry(
			"canonical holding_register is a fixed point",
			"holding_register",
			"holding_register",
		),
		Entry(
			"canonical input_register is a fixed point",
			"input_register",
			"input_register",
		),
	)

	It("Should flip the channel enabled polarity", func() {
		in := msgpack.EncodedJSON{
			"dataSaving": true,
			"channels":   []any{map[string]any{"type": "coil_input", "enabled": false}},
		}
		Expect(legacy.Read.Apply(in)).To(Equal(msgpack.EncodedJSON{
			"data_saving_disabled": false,
			"channels": []any{
				map[string]any{"type": "coil", "disabled": true},
			},
		}))
	})
})

var _ = Describe("Write", func() {
	DescribeTable("channel type tags",
		func(in, want string) {
			Expect(legacy.Write.Apply(channels(in))).To(Equal(channels(want)))
		},
		Entry("coil_output", "coil_output", "coil"),
		Entry("holding_register_output", "holding_register_output", "holding_register"),
		Entry("canonical coil is a fixed point", "coil", "coil"),
		Entry(
			"canonical holding_register is a fixed point",
			"holding_register",
			"holding_register",
		),
	)
})

var _ = Describe("Scan", func() {
	It("Should rename the driver scan fields", func() {
		in := msgpack.EncodedJSON{"scan_rate": 0.5, "enabled": true}
		Expect(legacy.Scan.Apply(in)).To(Equal(msgpack.EncodedJSON{
			"rate":     0.5,
			"disabled": false,
		}))
	})
})
