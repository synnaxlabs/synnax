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
	"runtime"

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
	Describe("NewDefault", func() {
		It("Should create directories and files with the given modes", func() {
			if runtime.GOOS == "windows" {
				Skip("Windows does not use Unix permission bits")
			}
			fs := xfs.NewDefault(xfs.UserRWX, xfs.UserRW)
			dir := filepath.Join(GinkgoT().TempDir(), "sub")
			sub := MustSucceed(fs.Sub(dir))
			f := MustSucceed(sub.Open("file", os.O_CREATE|os.O_WRONLY))
			Expect(f.Close()).To(Succeed())
			Expect(MustSucceed(os.Stat(dir)).Mode().Perm()).To(Equal(xfs.UserRWX))
			Expect(
				MustSucceed(os.Stat(filepath.Join(dir, "file"))).Mode().Perm(),
			).To(Equal(xfs.UserRW))
		})
	})
})
