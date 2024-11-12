// Copyright 2023 Synnax Labs, Inc.
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
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
	"os"
)

// Most of the functionalities of a File are well-tested in the tests for a file system in fs_test.go.
var _ = Describe("File", func() {
	fileSystems := map[string]func() xfs.FS{
		"memFS": func() xfs.FS {
			return xfs.NewMem()
		},
		"osFS": func() xfs.FS {
			return MustSucceed(xfs.Default.Sub("./testData"))
		},
	}

	for fsName, makeFS := range fileSystems {
		fsName, makeFS := fsName, makeFS

		var (
			fsRoot, fs xfs.FS
			f          xfs.File
		)

		Context("FS:"+fsName, Ordered, func() {
			BeforeEach(func() {
				fsRoot = makeFS()
				fs = MustSucceed(fsRoot.Sub("test-spec"))
				f = MustSucceed(fs.Open("file.fi", os.O_CREATE|os.O_RDWR))
				MustSucceed(f.Write([]byte("tacocat")))
			})
			AfterEach(func() {
				Expect(f.Close()).ToNot(HaveOccurred())
				Expect(fsRoot.Remove("test-spec")).To(Succeed())
			})
			AfterAll(func() { Expect(xfs.Default.Remove("testData")).To(Succeed()) })

			Describe("Stat", func() {
				It("Should give the size of the file correctly", func() {
					s, err := f.Stat()
					Expect(err).ToNot(HaveOccurred())
					Expect(s.Size()).To(Equal(int64(7)))
				})
			})
		})
	}
})
