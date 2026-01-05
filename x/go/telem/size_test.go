// Copyright 2026 Synnax Labs, Inc.
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
			Expect(s.String()).To(Equal("0 B"))
		})

		It("Should format bytes correctly", func() {
			s := telem.Size(42)
			Expect(s.String()).To(Equal("42 B"))
		})

		It("Should format kilobytes correctly", func() {
			s := telem.Kilobyte + 24*telem.Byte
			Expect(s.String()).To(Equal("1.024 kB"))
		})

		It("Should format megabytes correctly", func() {
			s := telem.Megabyte + 24*telem.Byte
			Expect(s.String()).To(Equal("1.000024 MB"))
		})

		It("Should format gigabytes correctly", func() {
			s := telem.Gigabyte + 24*telem.Megabyte
			Expect(s.String()).To(Equal("1.024 GB"))
		})

		It("Should format terabytes correctly", func() {
			s := telem.Terabyte + 24*telem.Gigabyte
			Expect(s.String()).To(Equal("1.024 TB"))
		})

		It("Should format mixed sizes correctly", func() {
			s := telem.Gigabyte + telem.Megabyte + telem.Kilobyte + 42*telem.Byte
			Expect(s.String()).To(Equal("1.001001042 GB"))
		})

		It("Should format large sizes correctly", func() {
			s := telem.Terabyte + telem.Gigabyte + telem.Megabyte + telem.Kilobyte
			Expect(s.String()).To(Equal("1.001001001 TB"))
		})
	})
})
