// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("Chan", func() {
	It("Should wrap the value type without a direction", func() {
		ch := types.Chan(types.I32())
		Expect(ch.Kind).To(Equal(types.KindChan))
		Expect(ch.Unwrap()).To(Equal(types.I32()))
		Expect(ch.ChanDirection.IsSet()).To(BeFalse())
	})
})

var _ = Describe("ReadChan", func() {
	It("Should return the inner type for read channels", func() {
		readChan := types.ReadChan(types.I32())
		Expect(readChan.Unwrap()).To(Equal(types.I32()))
	})

	It("Should return true on IsRead", func() {
		readonlyChan := types.ReadChan(types.I32())
		Expect(readonlyChan.ChanDirection.IsRead()).To(BeTrue())
	})

	It("Should return false on IsWrite", func() {
		readonlyChan := types.ReadChan(types.I32())
		Expect(readonlyChan.ChanDirection.IsWrite()).To(BeFalse())
	})

	It("Should return true on IsSet", func() {
		readonlyChan := types.ReadChan(types.I32())
		Expect(readonlyChan.ChanDirection.IsSet()).To(BeTrue())
	})
})

var _ = Describe("WriteChan", func() {
	It("Should return the inner type for write channels", func() {
		writeChan := types.WriteChan(types.I32())
		Expect(writeChan.Unwrap()).To(Equal(types.I32()))
	})

	It("Should return false on IsRead", func() {
		writeChan := types.WriteChan(types.I32())
		Expect(writeChan.ChanDirection.IsRead()).To(BeFalse())
	})

	It("Should return true on IsWrite", func() {
		writeChan := types.WriteChan(types.I32())
		Expect(writeChan.ChanDirection.IsWrite()).To(BeTrue())
	})

	It("Should return true on IsSet", func() {
		writeChan := types.WriteChan(types.I32())
		Expect(writeChan.ChanDirection.IsSet()).To(BeTrue())
	})
})

var _ = Describe("NewChannels", func() {
	It("Should create empty non-nil read and write sets", func() {
		channels := types.NewChannels()
		Expect(channels.Read).ToNot(BeNil())
		Expect(channels.Read).To(BeEmpty())
		Expect(channels.Write).ToNot(BeNil())
		Expect(channels.Write).To(BeEmpty())
	})
})
