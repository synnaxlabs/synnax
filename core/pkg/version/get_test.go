// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package version_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/version"
)

var _ = Describe("Get", func() {
	It("Should return the dev version when not set via ldflags", func() {
		Expect(version.Get()).To(Equal("0.0.0"))
	})
})

var _ = Describe("Commit", func() {
	It("Should return unknown when not set via ldflags", func() {
		Expect(version.Commit()).To(Equal("unknown"))
	})
})

var _ = Describe("Date", func() {
	It("Should return unknown when not set via ldflags", func() {
		Expect(version.Date()).To(Equal("unknown"))
	})
})

var _ = Describe("Time", func() {
	It("Should return zero time when not set via ldflags", func() {
		Expect(version.Time()).To(Equal(time.Time{}))
	})
})

var _ = Describe("Full", func() {
	It("Should return just version when commit and date are unknown", func() {
		full := version.Full()
		Expect(full).To(Equal("0.0.0"))
		Expect(full).NotTo(ContainSubstring("commit:"))
		Expect(full).NotTo(ContainSubstring("built:"))
	})
})
