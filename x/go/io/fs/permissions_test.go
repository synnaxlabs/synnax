// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fs_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/io/fs"
	"os"
)

var _ = Describe("Permissions", func() {
	Describe("CheckSufficientPermissions", func() {
		DescribeTable("should return the correct value",
			func(actual, expected os.FileMode, output bool) {
				Expect(fs.CheckSufficientPermissions(actual, expected)).To(Equal(output))
			},
			Entry("0755 700", os.FileMode(0755), fs.OS_USER_RWX, true),
			Entry("600 700", os.FileMode(600), fs.OS_USER_RWX, false),
		)

	})
})
