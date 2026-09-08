// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

// matchingTempDirs returns the basenames of every entry in os.TempDir that
// starts with the given prefix. Used to observe whether OpenOS leaves a
// directory behind after its enclosing scope exits.
func matchingTempDirs(prefix string) []string {
	entries := MustSucceed(os.ReadDir(os.TempDir()))
	var out []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) {
			out = append(out, e.Name())
		}
	}
	return out
}

var _ = Describe("FS Testutil", func() {
	Describe("CopyFS", func() {
		for fsName, openFS := range FileSystems {
			Context("FS: "+fsName, func() {
				It("Should copy one fs entirely from one place to another", func() {
					fs := openFS()
					sub1 := MustSucceed(fs.Sub("sub1"))
					sub2 := MustSucceed(fs.Sub("sub2"))

					By("Creating various files and directories in sub1", func() {
						// sub1
						//   - subsub1
						//       - subsubsub1
						//          - file3
						//       - file2
						//   - subsub2
						//   - file1
						subsub1 := MustSucceed(sub1.Sub("subsub1"))
						MustSucceed(sub1.Sub("subsub2"))
						f1 := MustSucceed(sub1.Open("file1", os.O_CREATE))
						Expect(f1.Close()).To(Succeed())
						subsubsub1 := MustSucceed(subsub1.Sub("subsubsub1"))
						f2 := MustSucceed(subsub1.Open("file2", os.O_CREATE))
						Expect(f2.Close()).To(Succeed())
						f3 := MustSucceed(subsubsub1.Open("file3", os.O_CREATE))
						Expect(f3.Close()).To(Succeed())
					})

					By("Copying the FS")
					Expect(CopyFS(sub1, sub2)).To(Succeed())

					By("Asserting it was an exact replica")
					infoToName := func(i os.FileInfo, _ int) string { return i.Name() }
					sub1list := lo.Map(MustSucceed(sub1.List("")), infoToName)
					sub2list := lo.Map(MustSucceed(sub2.List("")), infoToName)
					Expect(sub1list).To(Equal(sub2list))

					subsub1list := lo.Map(MustSucceed(sub1.List("subsub1")), infoToName)
					subsub2list := lo.Map(MustSucceed(sub2.List("subsub1")), infoToName)
					Expect(subsub1list).To(Equal(subsub2list))

					subsubsub1list := lo.Map(
						MustSucceed(sub1.List("subsub1/subsubsub1")),
						infoToName,
					)
					subsubsub2list := lo.Map(
						MustSucceed(sub2.List("subsub1/subsubsub1")),
						infoToName,
					)
					Expect(subsubsub1list).To(Equal(subsubsub2list))
				})
			})
		}
	})

	Describe("OpenMem", func() {
		It("Should return a usable in-memory FS", func() {
			fs := OpenMem()
			Expect(fs).ToNot(BeNil())
			f := MustSucceed(fs.Open("a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(f)
			MustSucceed(f.Write([]byte("hello")))
			Expect(MustSucceed(fs.Exists("a.bin"))).To(BeTrue())
		})

		It(
			"Should return a fresh FS on each call so callers cannot leak state across tests",
			func() {
				a := OpenMem()
				b := OpenMem()
				fa := MustSucceed(a.Open("only-in-a.bin", os.O_CREATE|os.O_RDWR))
				DeferClose(fa)
				Expect(MustSucceed(b.Exists("only-in-a.bin"))).To(BeFalse())
			},
		)

		It("Should leave nothing on disk", func() {
			before := matchingTempDirs(TempDirPrefix())
			OpenMem()
			Expect(matchingTempDirs(TempDirPrefix())).To(Equal(before))
		})
	})

	Describe("OpenOS", func() {
		It("Should return a usable on-disk FS", func() {
			fs := OpenOS()
			Expect(fs).ToNot(BeNil())
			f := MustSucceed(fs.Open("a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(f)
			MustSucceed(f.Write([]byte("hello")))
			Expect(MustSucceed(fs.Exists("a.bin"))).To(BeTrue())
		})

		It("Should return a fresh FS rooted at its own tempdir on each call", func() {
			a := OpenOS()
			b := OpenOS()
			fa := MustSucceed(a.Open("only-in-a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(fa)
			Expect(MustSucceed(b.Exists("only-in-a.bin"))).To(BeFalse())
		})

		It("Should back the FS with a real directory under os.TempDir", func() {
			before := matchingTempDirs(TempDirPrefix())
			OpenOS()
			after := matchingTempDirs(TempDirPrefix())
			Expect(after).To(HaveLen(len(before) + 1))
		})

		Describe("Cleanup", Ordered, func() {
			var (
				priorDirs []string
				createdAt string
			)
			BeforeAll(func() {
				ShouldNotLeakGoroutines()
				priorDirs = matchingTempDirs(TempDirPrefix())
			})
			It("Creates the tempdir while the spec is running", func() {
				OpenOS()
				current := matchingTempDirs(TempDirPrefix())
				Expect(current).To(HaveLen(len(priorDirs) + 1))
				for _, name := range current {
					if !slices.Contains(priorDirs, name) {
						createdAt = filepath.Join(os.TempDir(), name)
						break
					}
				}
				Expect(createdAt).ToNot(BeEmpty())
			})
			It("Removes the tempdir before the next spec runs", func() {
				_, err := os.Stat(createdAt)
				Expect(os.IsNotExist(err)).To(BeTrue())
				Expect(matchingTempDirs(TempDirPrefix())).To(Equal(priorDirs))
			})
		})
	})

	Describe("FileSystems", func() {
		It("Should expose memFS and osFS factories", func() {
			Expect(FileSystems).To(HaveKey("memFS"))
			Expect(FileSystems).To(HaveKey("osFS"))
		})

		It("Should produce a working FS for every backend", func() {
			for fsName, openFS := range FileSystems {
				By("backend: " + fsName)
				fs := openFS()
				f := MustSucceed(fs.Open("a.bin", os.O_CREATE|os.O_RDWR))
				DeferClose(f)
				MustSucceed(f.Write([]byte("hello")))
				Expect(MustSucceed(fs.Exists("a.bin"))).To(BeTrue())
			}
		})

		It("Should bind memFS to OpenMem and osFS to OpenOS", func() {
			before := matchingTempDirs(TempDirPrefix())
			FileSystems["memFS"]()
			Expect(matchingTempDirs(TempDirPrefix())).To(Equal(before))

			FileSystems["osFS"]()
			Expect(matchingTempDirs(TempDirPrefix())).To(HaveLen(len(before) + 1))
		})
	})

	Describe("Factory type", func() {
		It("Should be assignable from any FileSystems value", func() {
			var f Factory = OpenMem
			Expect(f()).ToNot(BeNil())
			f = OpenOS
			Expect(f()).ToNot(BeNil())
			f = FileSystems["memFS"]
			Expect(f()).ToNot(BeNil())
		})
	})
})
