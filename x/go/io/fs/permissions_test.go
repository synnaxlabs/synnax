// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fs_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/io/fs"
)

var _ = Describe("Permissions", func() {
	Describe("HasSufficientPermissions", func() {
		DescribeTable("should return the correct value",
			func(actual, expected os.FileMode, output bool) {
				Expect(fs.HasSufficientPermissions(actual, expected)).To(Equal(output))
			},
			Entry(
				"0o755 0o700",
				fs.UserRWX|fs.GroupRX|fs.OtherRX,
				fs.UserRWX,
				true,
			),
			Entry(
				"0o600 0o700",
				fs.UserRW,
				fs.UserRWX,
				false,
			),
			Entry(
				"0o650 0o750",
				fs.UserRW|fs.OtherRX,
				fs.UserRWX|fs.OtherRX,
				false,
			),
		)

	})
})
