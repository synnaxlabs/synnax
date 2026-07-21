// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/x/telem/types/v0"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Size", func() {
	Describe("String", func() {
		It("Should format zero size correctly", func() {
			s := v0.Size(0)
			Expect(s.String()).To(Equal("0 B"))
		})

		It("Should format bytes correctly", func() {
			s := v0.Size(42)
			Expect(s.String()).To(Equal("42 B"))
		})

		It("Should format kilobytes correctly", func() {
			s := v0.Kilobyte + 24*v0.Byte
			Expect(s.String()).To(Equal("1.024 kB"))
		})

		It("Should format megabytes correctly", func() {
			s := v0.Megabyte + 24*v0.Byte
			Expect(s.String()).To(Equal("1.000024 MB"))
		})

		It("Should format gigabytes correctly", func() {
			s := v0.Gigabyte + 24*v0.Megabyte
			Expect(s.String()).To(Equal("1.024 GB"))
		})

		It("Should format terabytes correctly", func() {
			s := v0.Terabyte + 24*v0.Gigabyte
			Expect(s.String()).To(Equal("1.024 TB"))
		})
		It("Should format petabytes correctly", func() {
			s := v0.Petabyte + 24*v0.Terabyte
			Expect(s.String()).To(Equal("1.024 PB"))
		})
		It("Should format exabytes correctly", func() {
			s := v0.Exabyte + 24*v0.Petabyte
			Expect(s.String()).To(Equal("1.024 EB"))
		})

		It("Should format mixed sizes correctly", func() {
			s := v0.Gigabyte + v0.Megabyte + v0.Kilobyte + 42*v0.Byte
			Expect(s.String()).To(Equal("1.001001042 GB"))
		})

		It("Should format large sizes correctly", func() {
			s := v0.Terabyte + v0.Gigabyte + v0.Megabyte + v0.Kilobyte
			Expect(s.String()).To(Equal("1.001001001 TB"))
		})
	})
	Describe("Kilobytes", func() {
		It("Should return the correct number of kilobytes", func() {
			s := v0.Gigabyte + v0.Megabyte + v0.Kilobyte + 42*v0.Byte
			Expect(s.Kilobytes()).To(Equal(1001001.042))
		})
	})
	Describe("Megabytes", func() {
		It("Should return the correct number of megabytes", func() {
			s := v0.Gigabyte + v0.Megabyte + v0.Kilobyte + 42*v0.Byte
			Expect(s.Megabytes()).To(Equal(1001.001042))
		})
	})
	Describe("Gigabytes", func() {
		It("Should return the correct number of gigabytes", func() {
			s := v0.Gigabyte + v0.Megabyte + v0.Kilobyte + 42*v0.Byte
			Expect(s.Gigabytes()).To(Equal(1.001001042))
		})
	})
	Describe("Terabytes", func() {
		It("Should return the correct number of terabytes", func() {
			s := v0.Terabyte + v0.Gigabyte + v0.Megabyte
			Expect(s.Terabytes()).To(Equal(1.001001))
		})
	})
	Describe("Petabytes", func() {
		It("Should return the correct number of petabytes", func() {
			s := v0.Petabyte + v0.Terabyte + v0.Gigabyte
			Expect(s.Petabytes()).To(Equal(1.001001))
		})
	})
	Describe("Exabytes", func() {
		It("Should return the correct number of exabytes", func() {
			s := v0.Exabyte + v0.Petabyte + v0.Terabyte
			Expect(s.Exabytes()).To(Equal(1.001001))
		})
	})
	Describe("MarshalJSON", func() {
		It("Should marshal the size into a string", func() {
			b := MustSucceed(json.Marshal(v0.Kilobyte))
			Expect(string(b)).To(Equal(`"1000"`))
		})
	})
	Describe("UnmarshalJSON", func() {
		It("Should unmarshal a size from a number", func() {
			var s v0.Size
			Expect(json.Unmarshal([]byte("1000"), &s)).To(Succeed())
			Expect(s).To(Equal(v0.Kilobyte))
		})
		It("Should unmarshal a size from a string", func() {
			var s v0.Size
			Expect(json.Unmarshal([]byte(`"1000"`), &s)).To(Succeed())
			Expect(s).To(Equal(v0.Kilobyte))
		})
		It("Should unmarshal a value past the float64 safe-integer range", func() {
			big := v0.Size(1)<<62 + 7
			b := MustSucceed(json.Marshal(big))
			var s v0.Size
			Expect(json.Unmarshal(b, &s)).To(Succeed())
			Expect(s).To(Equal(big))
		})
		It("Should return an error on an invalid size", func() {
			var s v0.Size
			Expect(json.Unmarshal([]byte(`"not-a-number"`), &s)).To(
				MatchError(ContainSubstring("invalid syntax")),
			)
		})
	})
})
