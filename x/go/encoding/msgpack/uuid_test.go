// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package msgpack_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xmsgpack "github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("UUID", func() {
	key := uuid.MustParse("01020304-0506-0708-090a-0b0c0d0e0f10")
	// bin8 header followed by the 16 bytes of key, the form every release before the
	// switch to the standard library package wrote.
	raw := []byte{
		0xc4, 0x10, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a,
		0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	}

	It("Should encode a UUID as its 16 raw bytes", func(ctx SpecContext) {
		Expect(xmsgpack.Codec.Encode(ctx, key)).To(Equal(raw))
	})
	It("Should decode a UUID from its 16 raw bytes", func(ctx SpecContext) {
		var decoded uuid.UUID
		Expect(xmsgpack.Codec.Decode(ctx, raw, &decoded)).To(Succeed())
		Expect(decoded).To(Equal(key))
	})
	It("Should decode a UUID from its textual form", func(ctx SpecContext) {
		b := MustSucceed(msgpack.Marshal([]byte(key.String())))
		var decoded uuid.UUID
		Expect(xmsgpack.Codec.Decode(ctx, b, &decoded)).To(Succeed())
		Expect(decoded).To(Equal(key))
	})
	It("Should return an error for an unparseable UUID", func() {
		b := MustSucceed(msgpack.Marshal([]byte("not-a-uuid")))
		var decoded uuid.UUID
		Expect(msgpack.Unmarshal(b, &decoded)).To(MatchError(
			ContainSubstring("failed to decode uuid from 10 bytes"),
		))
	})
})
