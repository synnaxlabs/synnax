// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/arc/types/versions/v1"
)

var _ = Describe("ChanDirection", func() {
	Describe("IsRead", func() {
		DescribeTable("Should report whether the direction includes read",
			func(d v1.ChanDirection, expected bool) {
				Expect(d.IsRead()).To(Equal(expected))
			},
			Entry("read", v1.ChanDirectionRead, true),
			Entry("write", v1.ChanDirectionWrite, false),
			Entry("none", v1.ChanDirectionNone, false),
			Entry("read_write", v1.ChanDirectionReadWrite, true),
			Entry("read|write", v1.ChanDirectionRead|v1.ChanDirectionWrite, true),
		)
	})
	Describe("IsWrite", func() {
		DescribeTable("Should report whether the direction includes write",
			func(d v1.ChanDirection, expected bool) {
				Expect(d.IsWrite()).To(Equal(expected))
			},
			Entry("write", v1.ChanDirectionWrite, true),
			Entry("read", v1.ChanDirectionRead, false),
			Entry("none", v1.ChanDirectionNone, false),
			Entry("read_write", v1.ChanDirectionReadWrite, true),
			Entry("read|write", v1.ChanDirectionRead|v1.ChanDirectionWrite, true),
		)
	})
	Describe("IsSet", func() {
		DescribeTable("Should report whether a direction has been specified",
			func(d v1.ChanDirection, expected bool) {
				Expect(d.IsSet()).To(Equal(expected))
			},
			Entry("none", v1.ChanDirectionNone, false),
			Entry("read", v1.ChanDirectionRead, true),
			Entry("write", v1.ChanDirectionWrite, true),
			Entry("read_write", v1.ChanDirectionReadWrite, true),
		)
	})
	Describe("ChanDirectionReadWrite", func() {
		It("Should equal the bitwise union of read and write", func() {
			Expect(v1.ChanDirectionReadWrite).
				To(Equal(v1.ChanDirectionRead | v1.ChanDirectionWrite))
		})
	})
	Describe("CheckCompatibility", func() {
		It("Should pass when write requires write", func() {
			Expect(
				v1.ChanDirectionWrite.CheckCompatibility(v1.ChanDirectionWrite),
			).To(Succeed())
		})
		It("Should fail when write requires write but got read", func() {
			Expect(
				v1.ChanDirectionWrite.CheckCompatibility(v1.ChanDirectionRead),
			).To(MatchError(ContainSubstring("write channel")))
		})
		It("Should pass when read requires read", func() {
			Expect(
				v1.ChanDirectionRead.CheckCompatibility(v1.ChanDirectionRead),
			).To(Succeed())
		})
		It("Should fail when read requires read but got write", func() {
			Expect(
				v1.ChanDirectionRead.CheckCompatibility(v1.ChanDirectionWrite),
			).To(MatchError(ContainSubstring("read channel")))
		})
		It("Should pass when required direction is unset", func() {
			Expect(
				v1.ChanDirectionNone.CheckCompatibility(v1.ChanDirectionWrite),
			).To(Succeed())
		})
		It("Should pass when actual direction is unset", func() {
			Expect(
				v1.ChanDirectionWrite.CheckCompatibility(v1.ChanDirectionNone),
			).To(Succeed())
		})
		It("Should pass when read_write satisfies either requirement", func() {
			Expect(
				v1.ChanDirectionRead.CheckCompatibility(v1.ChanDirectionReadWrite),
			).To(Succeed())
			Expect(
				v1.ChanDirectionWrite.CheckCompatibility(v1.ChanDirectionReadWrite),
			).To(Succeed())
		})
	})
})
