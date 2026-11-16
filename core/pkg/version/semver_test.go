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
)

var _ = Describe("IsDev", func() {
	DescribeTable(
		"Should report whether a version belongs to a dev build",
		func(v string, dev bool) { Expect(version.IsDev(v)).To(Equal(dev)) },
		Entry("unstamped", "0.0.0", true),
		Entry("stamped by CI", "0.0.0-dev.0.60.1.abc1234", true),
		Entry("numeric commit", "0.0.0-dev.0123456", true),
		Entry("release", "0.60.1", false),
		Entry("candidate", "0.60.0-rc.1", false),
	)
})

var _ = Describe("Valid", func() {
	DescribeTable(
		"Should report whether a version is valid semver",
		func(v string, valid bool) { Expect(version.Valid(v)).To(Equal(valid)) },
		Entry("release", "0.60.1", true),
		Entry("candidate", "0.60.0-rc.1", true),
		Entry("v prefix", "v0.60.1", false),
		Entry("short", "0.61", false),
		Entry("build metadata", "0.61.0+build", false),
		Entry("empty", "", false),
		Entry("garbage", "garbage", false),
	)
})

var _ = Describe("Compare", func() {
	DescribeTable(
		"Should order versions",
		func(a, b string, order int) { Expect(version.Compare(a, b)).To(Equal(order)) },
		Entry("equal", "0.60.0", "0.60.0", 0),
		Entry("older patch", "0.60.0", "0.60.1", -1),
		Entry("newer minor", "0.61.0", "0.60.5", 1),
		Entry("candidate before its release", "0.60.0-rc.1", "0.60.0", -1),
		Entry("numeric candidates", "0.60.0-rc.10", "0.60.0-rc.9", 1),
		Entry("invalid before valid", "garbage", "0.60.0", -1),
	)
})
