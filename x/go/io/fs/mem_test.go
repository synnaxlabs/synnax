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
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Mem", func() {
	Describe("SyncDir", func() {
		It("Should take a current directory element as the directory itself", func() {
			fs := fs.NewMem()
			MustSucceed(fs.Sub("a"))
			Expect(fs.SyncDir(".")).To(Succeed())
			Expect(fs.SyncDir("./a/.")).To(Succeed())
		})
		It("Should name the call and the path of the caller in its error", func() {
			fs := fs.NewMem()
			MustSucceed(fs.Sub("a/b"))
			Expect(fs.SyncDir("a/b/missing")).To(MatchError(
				&os.PathError{Op: "sync", Path: "a/b/missing", Err: os.ErrNotExist},
			))
		})
	})

	Describe("Current directory elements", func() {
		// Each backend is a root FS with no subFS, so no path.Join cleans a name. The
		// string is the prefix that puts a relative name under the root.
		backends := map[string]func() (fs.FS, string){
			"memFS": func() (fs.FS, string) { return fs.NewMem(), "" },
			"osFS": func() (fs.FS, string) {
				return fs.Default, GinkgoT().TempDir() + "/"
			},
		}
		for name, open := range backends {
			Context(name, func() {
				var (
					fsys fs.FS
					root string
				)
				BeforeEach(func() {
					fsys, root = open()
					MustSucceed(fsys.Sub(root + "a"))
				})
				It(
					"Should stat a directory named with a trailing . like SyncDir",
					func() {
						Expect(fsys.SyncDir(root + "a/.")).To(Succeed())
						info := MustSucceed(fsys.Stat(root + "a/."))
						Expect(info.IsDir()).To(BeTrue())
						Expect(fsys.Exists(root + "a/.")).To(BeTrue())
					},
				)
				It("Should not create a file named . in a directory", func() {
					Expect(fsys.Open(root+"a/.", os.O_CREATE|os.O_WRONLY)).Error().
						To(HaveOccurred())
					Expect(fsys.List(root + "a")).To(BeEmpty())
				})
				It(
					"Should keep a file reachable after a rename onto a trailing .",
					func() {
						sub := MustSucceed(fsys.Sub(root + "x"))
						f := MustSucceed(sub.Open("f.txt", os.O_CREATE|os.O_WRONLY))
						Expect(f.Close()).To(Succeed())
						renamed := fsys.Rename(root+"x", root+"a/.") == nil
						at := root + "x/f.txt"
						if renamed {
							at = root + "a/./f.txt"
						}
						Expect(fsys.Exists(at)).To(BeTrue(), "renamed: %v", renamed)
					},
				)
				It("Should return ENOTDIR from SyncDir for a file under a .", func() {
					f := MustSucceed(fsys.Open(root+"a/./f.txt", os.O_CREATE))
					Expect(f.Close()).To(Succeed())
					Expect(fsys.SyncDir(root + "a/./f.txt")).
						To(MatchError(syscall.ENOTDIR))
					Expect(fsys.Exists(root + "a/f.txt")).To(BeTrue())
				})
			})
		}
	})

	It("Should advance the modification time on every write", func() {
		fs := fs.NewMem()
		var prev time.Time
		for range 100 {
			f := MustSucceed(fs.Open("f.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC))
			MustSucceed(f.Write([]byte("tacocat")))
			Expect(f.Close()).To(Succeed())
			mod := MustSucceed(fs.Stat("f.txt")).ModTime()
			Expect(mod).To(BeTemporally(">", prev))
			prev = mod
		}
	})
})
