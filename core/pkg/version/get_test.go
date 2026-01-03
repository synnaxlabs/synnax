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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/synnax/pkg/version"
	"strings"
	"time"
)

var _ = Describe("Version", func() {
	Describe("Get", func() {
		It("Should return the version name", func() {
			v := version.Prod()
			cv := strings.TrimSpace(strings.ReplaceAll(v, "\n", ""))
			Expect(strings.Count(cv, ".")).To(Equal(2))
		})
	})

	Describe("Commit", func() {
		It("Should return unknown when not set via ldflags", func() {
			// When built without ldflags, should return unknown
			Expect(version.Commit()).To(Equal("unknown"))
		})
	})

	Describe("Date", func() {
		It("Should return unknown when not set via ldflags", func() {
			// When built without ldflags, should return unknown
			Expect(version.Date()).To(Equal("unknown"))
		})
	})

	Describe("Time", func() {
		It("Should return zero time when not set via ldflags", func() {
			// When built without ldflags, should return zero time
			Expect(version.Time()).To(Equal(time.Time{}))
		})
	})

	Describe("Full", func() {
		It("Should return just version when commit and date are unknown", func() {
			full := version.Full()
			// Should just be the version number since commit/date aren't set
			Expect(strings.Count(full, ".")).To(Equal(2))
			Expect(full).NotTo(ContainSubstring("commit:"))
			Expect(full).NotTo(ContainSubstring("built:"))
		})
	})
})
