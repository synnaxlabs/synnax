// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package strings_test

import (
	"math"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	stlstrings "github.com/synnaxlabs/arc/stl/strings"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ProgramState", func() {
	var s *stlstrings.ProgramState

	BeforeEach(func() {
		s = stlstrings.NewProgramState()
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

		It("Should return handle 0 for empty strings", func() {
			Expect(s.Create("")).To(Equal(uint32(0)))
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

	Describe("CreateLiteral", func() {
		It("Should return a handle in the literal range", func() {
			h := s.CreateLiteral("literal_value")
			Expect(h).To(BeNumerically(">=", uint32(1<<24)))
		})

		It("Should return unique handles for successive literal creates", func() {
			h1 := s.CreateLiteral("a")
			h2 := s.CreateLiteral("b")
			Expect(h1).ToNot(Equal(h2))
		})

		It("Should not collide with transient handles", func() {
			th := s.Create("transient")
			ch := s.CreateLiteral("literal")
			Expect(th).ToNot(Equal(ch))
		})

		It("Should return handle 0 for empty literal strings", func() {
			Expect(s.CreateLiteral("")).To(Equal(uint32(0)))
		})

		It("Should handle UTF-8 literal strings", func() {
			h := s.CreateLiteral("配置")
			Expect(MustBeOk(s.Get(h))).To(Equal("配置"))
		})

		It("Should not collide even after many transient creates", func() {
			for range 1000 {
				s.Create("transient")
			}
			ch := s.CreateLiteral("literal")
			Expect(ch).To(BeNumerically(">=", uint32(1<<24)))
			Expect(MustBeOk(s.Get(ch))).To(Equal("literal"))
		})
	})

	Describe("Get", func() {
		It("Should retrieve a transient string by handle", func() {
			h := s.Create("world")
			Expect(MustBeOk(s.Get(h))).To(Equal("world"))
		})

		It("Should retrieve a literal string by handle", func() {
			h := s.CreateLiteral("persistent")
			Expect(MustBeOk(s.Get(h))).To(Equal("persistent"))
		})

		It("Should return false for an unknown handle", func() {
			_, ok := s.Get(999)
			Expect(ok).To(BeFalse())
		})

		It("Should return empty string with ok=true for handle zero", func() {
			Expect(MustBeOk(s.Get(0))).To(Equal(""))
		})

		It("Should return false for max uint32 handle", func() {
			_, ok := s.Get(math.MaxUint32)
			Expect(ok).To(BeFalse())
		})

		It("Should check transient before literal on Get", func() {
			s.Create("a")
			s.Create("b")
			s.CreateLiteral("c")
			Expect(MustBeOk(s.Get(1))).To(Equal("a"))
			Expect(MustBeOk(s.Get(2))).To(Equal("b"))
		})

		It("Should fall back to literal when transient handle not found", func() {
			ch := s.CreateLiteral("fallback")
			Expect(MustBeOk(s.Get(ch))).To(Equal("fallback"))
		})

		It("Should return false for handle in gap between transient and literal", func() {
			s.Create("t")
			s.CreateLiteral("c")
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

		It("Should preserve literal strings", func() {
			ch := s.CreateLiteral("persistent")
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

		It("Should preserve multiple literal strings", func() {
			ch1 := s.CreateLiteral("c1")
			ch2 := s.CreateLiteral("c2")
			s.Create("t1")
			s.Create("t2")
			s.Clear()
			Expect(MustBeOk(s.Get(ch1))).To(Equal("c1"))
			Expect(MustBeOk(s.Get(ch2))).To(Equal("c2"))
		})

		It("Should not affect literal counter", func() {
			s.CreateLiteral("first")
			s.Clear()
			h2 := s.CreateLiteral("second")
			Expect(h2).To(Equal(uint32(1<<24 + 1)))
		})
	})

	Describe("Reset", func() {
		It("Should remove both transient and literal strings", func() {
			th := s.Create("transient")
			ch := s.CreateLiteral("literal")
			s.Reset()
			_, tok := s.Get(th)
			_, cok := s.Get(ch)
			Expect(tok).To(BeFalse())
			Expect(cok).To(BeFalse())
		})

		It("Should reset literal counter so literal handles restart", func() {
			h1 := s.CreateLiteral("first")
			s.Reset()
			h2 := s.CreateLiteral("second")
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
			s.CreateLiteral("b")
			s.Reset()
			th := s.Create("c")
			ch := s.CreateLiteral("d")
			Expect(MustBeOk(s.Get(th))).To(Equal("c"))
			Expect(MustBeOk(s.Get(ch))).To(Equal("d"))
		})

		It("Should be safe to call on empty state", func() {
			Expect(func() { s.Reset() }).ToNot(Panic())
		})

		It("Should be safe to call multiple times", func() {
			s.Create("a")
			s.CreateLiteral("b")
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

		It("Should maintain literal strings across clear cycles", func() {
			ch := s.CreateLiteral("stable")
			for range 10 {
				s.Create("temp")
				s.Clear()
			}
			Expect(MustBeOk(s.Get(ch))).To(Equal("stable"))
		})
	})
})
