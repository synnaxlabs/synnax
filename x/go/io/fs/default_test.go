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
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Default", func() {
	Describe("List", func() {
		It("Should skip a dangling symlink and list the other entries", func() {
			dir := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(dir, "kept"), nil, 0o600)).To(Succeed())
			Expect(os.Symlink(
				filepath.Join(dir, "missing"),
				filepath.Join(dir, "dangling"),
			)).To(Succeed())
			infos := MustSucceed(xfs.Default.List(dir))
			Expect(infos).To(HaveLen(1))
			Expect(infos[0].Name()).To(Equal("kept"))
		})
	})
})
