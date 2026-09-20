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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/arc/types/versions/v0"
)

var _ = Describe("ChanDirection", func() {
	Describe("IsRead", func() {
		DescribeTable("Should report whether the direction includes read",
			func(d v0.ChanDirection, expected bool) {
				Expect(d.IsRead()).To(Equal(expected))
			},
			Entry("read", v0.ChanDirectionRead, true),
			Entry("write", v0.ChanDirectionWrite, false),
			Entry("none", v0.ChanDirectionNone, false),
			Entry("read_write", v0.ChanDirectionReadWrite, true),
			Entry("read|write", v0.ChanDirectionRead|v0.ChanDirectionWrite, true),
		)
	})
	Describe("IsWrite", func() {
		DescribeTable("Should report whether the direction includes write",
			func(d v0.ChanDirection, expected bool) {
				Expect(d.IsWrite()).To(Equal(expected))
			},
			Entry("write", v0.ChanDirectionWrite, true),
			Entry("read", v0.ChanDirectionRead, false),
			Entry("none", v0.ChanDirectionNone, false),
			Entry("read_write", v0.ChanDirectionReadWrite, true),
			Entry("read|write", v0.ChanDirectionRead|v0.ChanDirectionWrite, true),
		)
	})
	Describe("IsSet", func() {
		DescribeTable("Should report whether a direction has been specified",
			func(d v0.ChanDirection, expected bool) {
				Expect(d.IsSet()).To(Equal(expected))
			},
			Entry("none", v0.ChanDirectionNone, false),
			Entry("read", v0.ChanDirectionRead, true),
			Entry("write", v0.ChanDirectionWrite, true),
			Entry("read_write", v0.ChanDirectionReadWrite, true),
		)
	})
	Describe("ChanDirectionReadWrite", func() {
		It("Should equal the bitwise union of read and write", func() {
			Expect(v0.ChanDirectionReadWrite).
				To(Equal(v0.ChanDirectionRead | v0.ChanDirectionWrite))
		})
	})
	Describe("CheckCompatibility", func() {
		It("Should pass when write requires write", func() {
			Expect(
				v0.ChanDirectionWrite.CheckCompatibility(v0.ChanDirectionWrite),
			).To(Succeed())
		})
		It("Should fail when write requires write but got read", func() {
			Expect(
				v0.ChanDirectionWrite.CheckCompatibility(v0.ChanDirectionRead),
			).To(MatchError(ContainSubstring("write channel")))
		})
		It("Should pass when read requires read", func() {
			Expect(
				v0.ChanDirectionRead.CheckCompatibility(v0.ChanDirectionRead),
			).To(Succeed())
		})
		It("Should fail when read requires read but got write", func() {
			Expect(
				v0.ChanDirectionRead.CheckCompatibility(v0.ChanDirectionWrite),
			).To(MatchError(ContainSubstring("read channel")))
		})
		It("Should pass when required direction is unset", func() {
			Expect(
				v0.ChanDirectionNone.CheckCompatibility(v0.ChanDirectionWrite),
			).To(Succeed())
		})
		It("Should pass when actual direction is unset", func() {
			Expect(
				v0.ChanDirectionWrite.CheckCompatibility(v0.ChanDirectionNone),
			).To(Succeed())
		})
		It("Should pass when read_write satisfies either requirement", func() {
			Expect(
				v0.ChanDirectionRead.CheckCompatibility(v0.ChanDirectionReadWrite),
			).To(Succeed())
			Expect(
				v0.ChanDirectionWrite.CheckCompatibility(v0.ChanDirectionReadWrite),
			).To(Succeed())
		})
	})
})
