// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ethercat/versions"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/v3"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func entry(dataType string) msgpack.EncodedJSON {
	return msgpack.EncodedJSON{"channels": []any{map[string]any{
		"type":      "entry",
		"key":       "a",
		"device":    "slave",
		"index":     0x6000,
		"sub_index": 1,
		"data_type": dataType,
	}}}
}

var _ = Describe("ReadConfigLadder", func() {
	DescribeTable("Should lift the data type of a v3 entry",
		func(ctx SpecContext, dataType string, expected *telem.DataType) {
			cfg := MustSucceed(versions.ReadConfigLadder.Decode(
				ctx, v3.Version, entry(dataType),
			))
			ch, ok := cfg.Channels[0].Variant.(versions.EntryReadChannel)
			Expect(ok).To(BeTrue())
			Expect(ch.DataType).To(Equal(expected))
		},
		Entry("an empty data type as absent", "", nil),
		Entry("a data type as itself", "int16", new(telem.Int16T)),
	)
})

var _ = Describe("WriteConfigLadder", func() {
	DescribeTable("Should lift the data type of a v3 entry",
		func(ctx SpecContext, dataType string, expected *telem.DataType) {
			cfg := MustSucceed(versions.WriteConfigLadder.Decode(
				ctx, v3.Version, entry(dataType),
			))
			ch, ok := cfg.Channels[0].Variant.(versions.EntryWriteChannel)
			Expect(ok).To(BeTrue())
			Expect(ch.DataType).To(Equal(expected))
		},
		Entry("an empty data type as absent", "", nil),
		Entry("a data type as itself", "uint8", new(telem.Uint8T)),
	)
})
