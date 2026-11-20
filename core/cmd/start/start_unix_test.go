// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !windows

package start_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/cmd/start"
	"golang.org/x/sys/unix"
)

var _ = Describe("DisablePermissionBits", func() {
	readMask := func() int {
		mask := unix.Umask(0)
		unix.Umask(mask)
		return mask
	}

	BeforeEach(func() {
		mask := readMask()
		DeferCleanup(func() { unix.Umask(mask) })
	})

	DescribeTable("Should remove every permission of other users",
		func(before, after int) {
			unix.Umask(before)
			start.DisablePermissionBits()
			Expect(readMask()).To(Equal(after))
		},
		Entry("from the usual mask", 0o022, 0o027),
		Entry("from a mask that keeps nothing for other users", 0o007, 0o007),
		Entry("while it keeps the mask of the group", 0o077, 0o077),
	)
})
