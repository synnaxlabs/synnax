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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/strings/state"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("State", func() {
	var s *state.State

	BeforeEach(func() {
		s = state.New()
	})

	Describe("Create", func() {
		It("Should return a non-zero handle", func() {
			h := s.Create("hello")
			Expect(h).ToNot(BeZero())
		})

		It("Should return unique handles for successive creates", func() {
			h1 := s.Create("a")
			h2 := s.Create("b")
			Expect(h1).ToNot(Equal(h2))
		})

		It("Should return monotonically increasing handles", func() {
			h1 := s.Create("a")
			h2 := s.Create("b")
			h3 := s.Create("c")
			Expect(h2).To(Equal(h1 + 1))
			Expect(h3).To(Equal(h2 + 1))
		})

		It("Should handle empty strings", func() {
			h := s.Create("")
			Expect(MustBeOk(s.Get(h))).To(Equal(""))
		})

		It("Should handle strings with special characters", func() {
			h := s.Create("hello\nworld\ttab\x00null")
			Expect(MustBeOk(s.Get(h))).To(Equal("hello\nworld\ttab\x00null"))
		})

		It("Should handle UTF-8 multi-byte characters", func() {
			h := s.Create("こんにちは世界 🌍")
			Expect(MustBeOk(s.Get(h))).To(Equal("こんにちは世界 🌍"))
		})

		It("Should handle very long strings", func() {
			long := strings.Repeat("x", 10000)
			h := s.Create(long)
			Expect(MustBeOk(s.Get(h))).To(Equal(long))
		})

		It("Should store duplicate string values with different handles", func() {
			h1 := s.Create("same")
			h2 := s.Create("same")
			Expect(h1).ToNot(Equal(h2))
			Expect(MustBeOk(s.Get(h1))).To(Equal("same"))
			Expect(MustBeOk(s.Get(h2))).To(Equal("same"))
		})
	})

	Describe("CreateConfig", func() {
		It("Should return a handle in the config range", func() {
			h := s.CreateConfig("config_value")
			Expect(h).To(BeNumerically(">=", uint32(1<<24)))
		})

		It("Should return unique handles for successive config creates", func() {
			h1 := s.CreateConfig("a")
			h2 := s.CreateConfig("b")
			Expect(h1).ToNot(Equal(h2))
		})

		It("Should not collide with transient handles", func() {
			th := s.Create("transient")
			ch := s.CreateConfig("config")
			Expect(th).ToNot(Equal(ch))
		})

		It("Should handle empty config strings", func() {
			h := s.CreateConfig("")
			Expect(MustBeOk(s.Get(h))).To(Equal(""))
		})

		It("Should handle UTF-8 config strings", func() {
			h := s.CreateConfig("配置")
			Expect(MustBeOk(s.Get(h))).To(Equal("配置"))
		})

		It("Should not collide even after many transient creates", func() {
			for range 1000 {
				s.Create("transient")
			}
			ch := s.CreateConfig("config")
			Expect(ch).To(BeNumerically(">=", uint32(1<<24)))
			Expect(MustBeOk(s.Get(ch))).To(Equal("config"))
		})
	})

	Describe("Get", func() {
		It("Should retrieve a transient string by handle", func() {
			h := s.Create("world")
			Expect(MustBeOk(s.Get(h))).To(Equal("world"))
		})

		It("Should retrieve a config string by handle", func() {
			h := s.CreateConfig("persistent")
			Expect(MustBeOk(s.Get(h))).To(Equal("persistent"))
		})

		It("Should return false for an unknown handle", func() {
			_, ok := s.Get(999)
			Expect(ok).To(BeFalse())
		})

		It("Should return false for handle zero", func() {
			_, ok := s.Get(0)
			Expect(ok).To(BeFalse())
		})

		It("Should return false for max uint32 handle", func() {
			_, ok := s.Get(math.MaxUint32)
			Expect(ok).To(BeFalse())
		})

		It("Should check transient before config on Get", func() {
			s.Create("a")
			s.Create("b")
			s.CreateConfig("c")
			Expect(MustBeOk(s.Get(1))).To(Equal("a"))
			Expect(MustBeOk(s.Get(2))).To(Equal("b"))
		})

		It("Should fall back to config when transient handle not found", func() {
			ch := s.CreateConfig("fallback")
			Expect(MustBeOk(s.Get(ch))).To(Equal("fallback"))
		})

		It("Should return false for handle in gap between transient and config", func() {
			s.Create("t")
			s.CreateConfig("c")
			_, ok := s.Get(100)
			Expect(ok).To(BeFalse())
		})
	})

	Describe("Clear", func() {
		It("Should remove transient strings", func() {
			h := s.Create("ephemeral")
			s.Clear()
			_, ok := s.Get(h)
			Expect(ok).To(BeFalse())
		})

		It("Should preserve config strings", func() {
			ch := s.CreateConfig("persistent")
			s.Create("ephemeral")
			s.Clear()
			Expect(MustBeOk(s.Get(ch))).To(Equal("persistent"))
		})

		It("Should reset transient counter so handles restart from 1", func() {
			s.Create("a")
			s.Create("b")
			s.Clear()
			h := s.Create("c")
			Expect(h).To(Equal(uint32(1)))
		})

		It("Should allow creating new transient strings after clear", func() {
			s.Create("old")
			s.Clear()
			h := s.Create("new")
			Expect(MustBeOk(s.Get(h))).To(Equal("new"))
		})

		It("Should be safe to call on empty state", func() {
			Expect(func() { s.Clear() }).ToNot(Panic())
		})

		It("Should be safe to call multiple times", func() {
			s.Create("a")
			s.Clear()
			s.Clear()
			s.Clear()
			h := s.Create("b")
			Expect(h).To(Equal(uint32(1)))
		})

		It("Should preserve multiple config strings", func() {
			ch1 := s.CreateConfig("c1")
			ch2 := s.CreateConfig("c2")
			s.Create("t1")
			s.Create("t2")
			s.Clear()
			Expect(MustBeOk(s.Get(ch1))).To(Equal("c1"))
			Expect(MustBeOk(s.Get(ch2))).To(Equal("c2"))
		})

		It("Should not affect config counter", func() {
			s.CreateConfig("first")
			s.Clear()
			h2 := s.CreateConfig("second")
			Expect(h2).To(Equal(uint32(1<<24 + 1)))
		})
	})

	Describe("Reset", func() {
		It("Should remove both transient and config strings", func() {
			th := s.Create("transient")
			ch := s.CreateConfig("config")
			s.Reset()
			_, tok := s.Get(th)
			_, cok := s.Get(ch)
			Expect(tok).To(BeFalse())
			Expect(cok).To(BeFalse())
		})

		It("Should reset config counter so config handles restart", func() {
			h1 := s.CreateConfig("first")
			s.Reset()
			h2 := s.CreateConfig("second")
			Expect(h2).To(Equal(h1))
		})

		It("Should reset transient counter", func() {
			s.Create("a")
			s.Create("b")
			s.Reset()
			h := s.Create("c")
			Expect(h).To(Equal(uint32(1)))
		})

		It("Should allow full reuse after reset", func() {
			s.Create("a")
			s.CreateConfig("b")
			s.Reset()
			th := s.Create("c")
			ch := s.CreateConfig("d")
			Expect(MustBeOk(s.Get(th))).To(Equal("c"))
			Expect(MustBeOk(s.Get(ch))).To(Equal("d"))
		})

		It("Should be safe to call on empty state", func() {
			Expect(func() { s.Reset() }).ToNot(Panic())
		})

		It("Should be safe to call multiple times", func() {
			s.Create("a")
			s.CreateConfig("b")
			s.Reset()
			s.Reset()
			s.Reset()
			h := s.Create("c")
			Expect(h).To(Equal(uint32(1)))
		})
	})

	Describe("Lifecycle", func() {
		It("Should handle repeated clear-create cycles", func() {
			for cycle := range 10 {
				h := s.Create("cycle")
				Expect(h).To(Equal(uint32(1)))
				Expect(MustBeOk(s.Get(h))).To(Equal("cycle"))
				s.Clear()
				_ = cycle
			}
		})

		It("Should maintain config strings across clear cycles", func() {
			ch := s.CreateConfig("stable")
			for range 10 {
				s.Create("temp")
				s.Clear()
			}
			Expect(MustBeOk(s.Get(ch))).To(Equal("stable"))
		})
	})
})
