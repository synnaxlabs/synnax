// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("ParseKey", func() {
	It("Should correctly parse a key from its string representation", func() {
		Expect(MustSucceed(channel.ParseKey("123456"))).To(Equal(channel.Key(123456)))
	})
	It("Should return an error when the key is not a valid integer", func() {
		Expect(channel.ParseKey("123456a")).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("123456a is not a valid channel key")),
		))
	})
})

var _ = Describe("NewKey", func() {
	It("Should compose a key from a leaseholder and a local key", func() {
		k := channel.NewKey(1, 2)
		Expect(k.Lease()).To(Equal(node.Key(1)))
		Expect(k.LocalKey()).To(Equal(channel.LocalKey(2)))
	})
})

var _ = Describe("KeysFromUint32", func() {
	It("Should reinterpret a slice of uint32 as Keys", func() {
		Expect(
			channel.KeysFromUint32([]uint32{1, 2, 3}),
		).To(Equal(channel.Keys{1, 2, 3}))
	})
})

var _ = Describe("KeysFromChannels", func() {
	It("Should return the keys of the given channels", func() {
		channels := []channel.Channel{
			{Leaseholder: 1, LocalKey: 1},
			{Leaseholder: 2, LocalKey: 3},
		}
		Expect(channel.KeysFromChannels(channels)).To(Equal(channel.Keys{
			channel.NewKey(1, 1),
			channel.NewKey(2, 3),
		}))
	})
	It("Should return an empty slice when given no channels", func() {
		Expect(channel.KeysFromChannels(nil)).To(BeEmpty())
	})
})

var _ = Describe("Names", func() {
	It("Should return the names of the given channels", func() {
		channels := []channel.Channel{{Name: "alpha"}, {Name: "beta"}}
		Expect(channel.Names(channels)).To(Equal([]string{"alpha", "beta"}))
	})
})
