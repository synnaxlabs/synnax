// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Size", func() {
	Describe("String", func() {
		It("Should format zero size correctly", func() {
			s := telem.Size(0)
			Expect(s.String()).To(Equal("0B"))
		})

		It("Should format bytes correctly", func() {
			s := telem.Size(42)
			Expect(s.String()).To(Equal("42B"))
		})

		It("Should format kilobytes correctly", func() {
			s := telem.Size(1024)
			Expect(s.String()).To(Equal("1KB 24B"))
		})

		It("Should format megabytes correctly", func() {
			s := telem.Size(1024 * 1024)
			Expect(s.String()).To(Equal("1MB 24KB"))
		})

		It("Should format gigabytes correctly", func() {
			s := telem.Size(1024 * 1024 * 1024)
			Expect(s.String()).To(Equal("1GB 24MB"))
		})

		It("Should format terabytes correctly", func() {
			s := telem.Size(1024 * 1024 * 1024 * 1024)
			Expect(s.String()).To(Equal("1TB 24GB"))
		})

		It("Should format mixed sizes correctly", func() {
			s := telem.Size(1024*1024*1024 + 1024*1024 + 1024 + 42)
			Expect(s.String()).To(Equal("1GB 1MB 1KB 42B"))
		})

		It("Should format large sizes correctly", func() {
			s := telem.Size(1024*1024*1024*1024 + 1024*1024*1024 + 1024*1024 + 1024 + 42)
			Expect(s.String()).To(Equal("1TB 1GB 1MB 1KB 42B"))
		})
	})

	Describe("isZero", func() {
		It("Should return true for zero size", func() {
			s := telem.Size(0)
			Expect(s.isZero()).To(BeTrue())
		})

		It("Should return false for non-zero size", func() {
			s := telem.Size(42)
			Expect(s.isZero()).To(BeFalse())
		})
	})

	Describe("truncate", func() {
		It("Should truncate to the nearest multiple", func() {
			s := telem.Size(1024 + 512)
			Expect(s.truncate(telem.Kilobyte)).To(Equal(telem.Size(1024)))
		})
	})

	Describe("sub", func() {
		It("Should subtract sizes correctly", func() {
			s1 := telem.Size(1024)
			s2 := telem.Size(512)
			Expect(s1.sub(s2)).To(Equal(telem.Size(512)))
		})
	})
})
