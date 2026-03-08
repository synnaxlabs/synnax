// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package state_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/control/state"
)

var _ = Describe("State", func() {
	var s *state.State

	BeforeEach(func() {
		s = &state.State{}
	})

	Describe("Set", func() {
		It("Should buffer a global authority change", func() {
			s.Set(nil, 200)
			changes := s.Flush()
			Expect(changes).To(HaveLen(1))
			Expect(changes[0].Channel).To(BeNil())
			Expect(changes[0].Authority).To(Equal(uint8(200)))
		})

		It("Should buffer a channel-specific authority change", func() {
			ch := uint32(42)
			s.Set(&ch, 100)
			changes := s.Flush()
			Expect(changes).To(HaveLen(1))
			Expect(*changes[0].Channel).To(Equal(uint32(42)))
			Expect(changes[0].Authority).To(Equal(uint8(100)))
		})

		It("Should accumulate multiple changes in order", func() {
			ch1 := uint32(1)
			ch2 := uint32(2)
			s.Set(&ch1, 50)
			s.Set(nil, 0)
			s.Set(&ch2, 255)
			changes := s.Flush()
			Expect(changes).To(HaveLen(3))
			Expect(*changes[0].Channel).To(Equal(uint32(1)))
			Expect(changes[0].Authority).To(Equal(uint8(50)))
			Expect(changes[1].Channel).To(BeNil())
			Expect(changes[1].Authority).To(Equal(uint8(0)))
			Expect(*changes[2].Channel).To(Equal(uint32(2)))
			Expect(changes[2].Authority).To(Equal(uint8(255)))
		})

		It("Should handle authority value zero", func() {
			s.Set(nil, 0)
			changes := s.Flush()
			Expect(changes[0].Authority).To(Equal(uint8(0)))
		})

		It("Should handle authority value at max uint8", func() {
			s.Set(nil, math.MaxUint8)
			changes := s.Flush()
			Expect(changes[0].Authority).To(Equal(uint8(255)))
		})

		It("Should handle channel key zero", func() {
			ch := uint32(0)
			s.Set(&ch, 100)
			changes := s.Flush()
			Expect(changes).To(HaveLen(1))
			Expect(*changes[0].Channel).To(Equal(uint32(0)))
		})

		It("Should handle channel key at max uint32", func() {
			ch := uint32(math.MaxUint32)
			s.Set(&ch, 100)
			changes := s.Flush()
			Expect(*changes[0].Channel).To(Equal(uint32(math.MaxUint32)))
		})

		It("Should handle duplicate channel changes", func() {
			ch := uint32(5)
			s.Set(&ch, 100)
			s.Set(&ch, 200)
			changes := s.Flush()
			Expect(changes).To(HaveLen(2))
			Expect(changes[0].Authority).To(Equal(uint8(100)))
			Expect(changes[1].Authority).To(Equal(uint8(200)))
		})

		It("Should store the pointer directly so mutations are visible", func() {
			ch := uint32(5)
			s.Set(&ch, 100)
			ch = 10
			changes := s.Flush()
			Expect(*changes[0].Channel).To(Equal(uint32(10)))
		})

		It("Should handle many changes", func() {
			for i := range 1000 {
				ch := uint32(i)
				s.Set(&ch, uint8(i%256))
			}
			changes := s.Flush()
			Expect(changes).To(HaveLen(1000))
			Expect(changes[0].Authority).To(Equal(uint8(0)))
			Expect(*changes[999].Channel).To(Equal(uint32(999)))
		})
	})

	Describe("Flush", func() {
		It("Should return nil when no changes are buffered", func() {
			Expect(s.Flush()).To(BeNil())
		})

		It("Should clear the buffer after flushing", func() {
			s.Set(nil, 100)
			changes := s.Flush()
			Expect(changes).To(HaveLen(1))
			Expect(s.Flush()).To(BeNil())
		})

		It("Should allow buffering new changes after flush", func() {
			s.Set(nil, 50)
			s.Flush()
			s.Set(nil, 75)
			changes := s.Flush()
			Expect(changes).To(HaveLen(1))
			Expect(changes[0].Authority).To(Equal(uint8(75)))
		})

		It("Should return nil on consecutive empty flushes", func() {
			Expect(s.Flush()).To(BeNil())
			Expect(s.Flush()).To(BeNil())
		})

		It("Should handle flush-set-flush cycles", func() {
			for i := range 10 {
				s.Set(nil, uint8(i))
				changes := s.Flush()
				Expect(changes).To(HaveLen(1))
				Expect(changes[0].Authority).To(Equal(uint8(i)))
			}
			Expect(s.Flush()).To(BeNil())
		})
	})

	Describe("Zero Value", func() {
		It("Should be usable without explicit construction", func() {
			var zeroState state.State
			zeroState.Set(nil, 128)
			changes := zeroState.Flush()
			Expect(changes).To(HaveLen(1))
			Expect(changes[0].Authority).To(Equal(uint8(128)))
		})

		It("Should return nil flush on zero value", func() {
			var zeroState state.State
			Expect(zeroState.Flush()).To(BeNil())
		})
	})

	Describe("AuthorityChange", func() {
		It("Should distinguish nil channel from zero channel", func() {
			ch := uint32(0)
			s.Set(nil, 100)
			s.Set(&ch, 200)
			changes := s.Flush()
			Expect(changes[0].Channel).To(BeNil())
			Expect(changes[1].Channel).ToNot(BeNil())
			Expect(*changes[1].Channel).To(Equal(uint32(0)))
		})
	})
})
