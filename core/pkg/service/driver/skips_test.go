// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

var _ = Describe("Skips", func() {
	var (
		s     *driver.Skips
		start time.Time
	)
	BeforeEach(func() {
		s = driver.NewSkips("record")
		start = time.Unix(1_000, 0)
	})

	// read counts n items read.
	read := func(n int) {
		for range n {
			s.Read()
		}
	}
	// expectDeadline expects the deadline of s to be at, or none for a zero at.
	expectDeadline := func(at time.Time) {
		GinkgoHelper()
		deadline, ok := s.Deadline()
		Expect(ok).To(Equal(!at.IsZero()))
		Expect(deadline).To(Equal(at))
	}

	It("Should give no warning before a skip", func() {
		read(3)
		Expect(s.Warning(start)).To(BeNil())
		expectDeadline(time.Time{})
	})

	It("Should warn with the counts and the last reason", func() {
		read(12_000)
		s.Skip(start, "first")
		s.Skip(start, "second")
		Expect(s.Warning(start)).To(Equal(&driver.Warning{
			Message:     "Skipped 2 of 12,000 records",
			Description: "second",
			Data: msgpack.EncodedJSON{
				"stats": map[string]any{"read": uint64(12_000), "skipped": uint64(2)},
			},
		}))
	})

	It("Should name one item in the singular", func() {
		read(1)
		s.Skip(start, "bad")
		Expect(s.Warning(start).Message).To(Equal("Skipped 1 of 1 record"))
	})

	It("Should keep the warning until a second passes", func() {
		read(2)
		s.Skip(start, "first")
		first := s.Warning(start)
		s.Skip(start.Add(500*time.Millisecond), "second")
		Expect(s.Warning(start.Add(500 * time.Millisecond))).To(BeIdenticalTo(first))
		expectDeadline(start.Add(time.Second))
		Expect(s.Warning(start.Add(time.Second)).Message).
			To(Equal("Skipped 2 of 2 records"))
	})

	It("Should keep the warning while no item is skipped", func() {
		read(1)
		s.Skip(start, "bad")
		first := s.Warning(start)
		read(5)
		Expect(s.Warning(start.Add(2 * time.Second))).To(BeIdenticalTo(first))
	})

	It("Should clear the warning a hold after the last skip", func() {
		read(2)
		s.Skip(start, "first")
		s.Skip(start.Add(time.Second), "second")
		Expect(s.Warning(start.Add(time.Second))).NotTo(BeNil())
		last := start.Add(time.Second)
		expectDeadline(last.Add(driver.SkipHold))
		Expect(s.Warning(last.Add(driver.SkipHold - time.Millisecond))).NotTo(BeNil())
		Expect(s.Warning(last.Add(driver.SkipHold))).To(BeNil())
		expectDeadline(time.Time{})
	})
})
