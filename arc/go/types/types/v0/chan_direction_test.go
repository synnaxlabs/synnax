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
	v0 "github.com/synnaxlabs/arc/types/types/v0"
)

var _ = Describe("ChanDirection", func() {
	Describe("IsRead", func() {
		It("Should return true for ChanDirectionRead", func() {
			Expect(v0.ChanDirectionRead.IsRead()).To(BeTrue())
		})
		It("Should return false for ChanDirectionWrite", func() {
			Expect(v0.ChanDirectionWrite.IsRead()).To(BeFalse())
		})
		It("Should return false for ChanDirectionNone", func() {
			Expect(v0.ChanDirectionNone.IsRead()).To(BeFalse())
		})
		It("Should return true for combined read+write", func() {
			combined := v0.ChanDirectionRead | v0.ChanDirectionWrite
			Expect(combined.IsRead()).To(BeTrue())
		})
	})
	Describe("IsWrite", func() {
		It("Should return true for ChanDirectionWrite", func() {
			Expect(v0.ChanDirectionWrite.IsWrite()).To(BeTrue())
		})
		It("Should return false for ChanDirectionRead", func() {
			Expect(v0.ChanDirectionRead.IsWrite()).To(BeFalse())
		})
		It("Should return false for ChanDirectionNone", func() {
			Expect(v0.ChanDirectionNone.IsWrite()).To(BeFalse())
		})
		It("Should return true for combined read+write", func() {
			combined := v0.ChanDirectionRead | v0.ChanDirectionWrite
			Expect(combined.IsWrite()).To(BeTrue())
		})
	})
	Describe("IsSet", func() {
		It("Should return false for ChanDirectionNone", func() {
			Expect(v0.ChanDirectionNone.IsSet()).To(BeFalse())
		})
		It("Should return true for ChanDirectionRead", func() {
			Expect(v0.ChanDirectionRead.IsSet()).To(BeTrue())
		})
		It("Should return true for ChanDirectionWrite", func() {
			Expect(v0.ChanDirectionWrite.IsSet()).To(BeTrue())
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
	})
})
