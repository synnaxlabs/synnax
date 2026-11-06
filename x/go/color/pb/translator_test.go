// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/color/pb"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/protobuf/proto"
)

func alphaEntries() []TableEntry {
	entries := make([]TableEntry, 256)
	for a := range entries {
		entries[a] = Entry(nil, a)
	}
	return entries
}

var _ = Describe("Translator", func() {
	It("Should pack the color into one field", func() {
		msg := MustSucceed(
			pb.ColorToPB(color.Color{R: 0x12, G: 0x34, B: 0x56, A: 0x78}),
		)
		Expect(msg.GetRgba()).To(Equal(uint32(0x12345678)))
	})

	DescribeTable("Should round-trip every alpha through the wire",
		func(a int) {
			original := color.Color{R: 10, G: 20, B: 30, A: uint8(a)}
			data := MustSucceed(proto.Marshal(MustSucceed(pb.ColorToPB(original))))
			var msg pb.Color
			Expect(proto.Unmarshal(data, &msg)).To(Succeed())
			Expect(pb.ColorFromPB(&msg)).To(Equal(original))
		},
		alphaEntries(),
	)

	It("Should translate a nil message to the zero color", func() {
		Expect(pb.ColorFromPB(nil)).To(Equal(color.Color{}))
	})
})
