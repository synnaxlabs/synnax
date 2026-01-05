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
	xfs "github.com/synnaxlabs/x/io/fs"
	invariants "github.com/synnaxlabs/x/io/fs/internal/invariants"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("FS", func() {
	fileSystems := map[string]func() xfs.FS{
		"memFS": func() xfs.FS { return xfs.NewMem() },
		"osFS":  func() xfs.FS { return MustSucceed(xfs.Default.Sub("./testData")) },
	}

	for fsName, makeFS := range fileSystems {
		var fsRoot, fs xfs.FS
		Context("FS:"+fsName, Ordered, func() {
			BeforeEach(func() {
				fsRoot = makeFS()
				fs = MustSucceed(fsRoot.Sub("test-spec"))
			})
			AfterEach(func() {
				Expect(fsRoot.Remove("test-spec")).To(Succeed())
			})
			AfterAll(func() { Expect(xfs.Default.Remove("testData")).To(Succeed()) })
			Describe("Stat", func() {
				It("Should return the correct file info", func() {
					file := MustSucceed(fs.Open("test_file.txt", os.O_CREATE|os.O_RDWR))
					Expect(file.Write([]byte("tacocat"))).To(Equal(7))
					stats := MustSucceed(file.Stat())
					Expect(stats.Size()).To(Equal(int64(7)))
					Expect(file.Close()).To(Succeed())
					Expect(fs.Remove("test_file.txt")).To(Succeed())
				})
			})
			Describe("Open", func() {
				Describe("Create test CREATE flag", func() {
					It("Should create a file", func() {
						file := MustSucceed(fs.Open("test_file.txt", os.O_CREATE))
						fileStats := MustSucceed(file.Stat())
						Expect(fileStats.Name()).To(Equal("test_file.txt"))
						Expect(file.Close()).To(Succeed())
					})
				})
				Describe("Create test without create flag", func() {
					It("Should not create a file", func() {
						Expect(fs.Open("test_file.txt", os.O_RDONLY)).Error().
							To(MatchError(os.ErrNotExist))
					})
					It("Should not create a file under write mode", func() {
						Expect(fs.Open("test_file.txt", os.O_WRONLY)).Error().
							To(MatchError(os.ErrNotExist))
						Expect(MustSucceed(fs.Exists("test_file.txt"))).To(BeFalse())
					})
					It("Should not create a file under read write mode", func() {
						Expect(fs.Open("test_file.txt", os.O_RDWR)).Error().
							To(MatchError(os.ErrNotExist))
						Expect(MustSucceed(fs.Exists("test_file.txt"))).To(BeFalse())
					})
				})
				Describe("Create test with exclusive", func() {
					It("Should create a file in MemFS", func() {
						f := MustSucceed(
							fs.Open("test_file.txt", os.O_CREATE|os.O_EXCL),
						)
						Expect(f.Close()).To(Succeed())
					})
				})
				Describe("Read file test", func() {
					It("Should read a file in MemFS", func() {
						f := MustSucceed(fs.Open("test_file.txt", os.O_CREATE))
						Expect(f.Close()).To(Succeed())

						file := MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						info := MustSucceed(file.Stat())
						Expect(info.Name()).To(Equal("test_file.txt"))
						Expect(file.Close()).To(Succeed())
					})
					It("Should read a file even when passed in with create", func() {
						f := MustSucceed(
							fs.Open("test_file.txt", os.O_CREATE|os.O_RDWR),
						)
						Expect(f.Write([]byte{1, 2, 3, 4})).To(Equal(4))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_CREATE|os.O_RDWR))
						stats := MustSucceed(f.Stat())
						Expect(stats.Size()).To(BeEquivalentTo(4))
						buf := make([]byte, 4)
						Expect(f.Read(buf)).To(Equal(4))
						Expect(buf).To(Equal([]byte{1, 2, 3, 4}))
						Expect(f.Close()).To(Succeed())
					})
				})
				Describe("Read nonexistent file test", func() {
					It("Should break when no such file exists", func() {
						Expect(fs.Open("test_file.txt", os.O_RDONLY)).Error().
							To(MatchError(os.ErrNotExist))
					})
				})
				Describe("Reader/Writer test", func() {
					It("Should write & read the contents of file in MemFS", func() {
						f := MustSucceed(fs.Open("test_file.txt", os.O_CREATE))
						Expect(f.Close()).To(Succeed())

						file := MustSucceed(fs.Open("test_file.txt", os.O_RDWR))
						Expect(file.Write([]byte("tacocat"))).To(Equal(7))
						Expect(file.Close()).To(Succeed())

						file = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						r := make([]byte, 7)
						Expect(file.Read(r)).To(Equal(7))
						Expect(r).To(Equal([]byte("tacocat")))
						Expect(file.Close()).To(Succeed())
					})
				})
				Describe("Reader/Writer without permission", func() {
					It("Should read but not write to file", func() {
						f := MustSucceed(fs.Open("test_file.txt", os.O_CREATE))
						Expect(f.Close()).To(Succeed())

						file := MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						Expect(file.Write([]byte("tacocat"))).Error().
							To(MatchError(invariants.ErrAccessDenied))
						Expect(file.Close()).To(Succeed())

						file = MustSucceed(fs.Open("test_file.txt", os.O_WRONLY))
						Expect(file.Write([]byte("tacocat"))).To(Equal(7))
						Expect(file.Close()).To(Succeed())

						file = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						r := make([]byte, 7)
						Expect(file.Read(r)).To(Equal(7))
						Expect(r).To(Equal([]byte("tacocat")))
						Expect(file.Close()).To(Succeed())
					})
				})
				Describe("ReaderAt/WriterAt test", func() {
					It("Should write & read the contents of file in MemFS", func() {
						f := MustSucceed(fs.Open("test_file.txt", os.O_CREATE))
						Expect(f.Close()).To(Succeed())

						file := MustSucceed(fs.Open("test_file.txt", os.O_RDWR))
						Expect(file.WriteAt([]byte("tacocat"), 3)).To(Equal(7))
						Expect(file.Close()).To(Succeed())

						file = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						r := make([]byte, 10)
						Expect(file.Read(r)).To(Equal(10))
						Expect(r).To(Equal([]byte("\x00\x00\x00tacocat")))
						Expect(file.Close()).To(Succeed())

						file = MustSucceed(fs.Open("test_file.txt", os.O_RDWR))
						Expect(file.WriteAt([]byte("ocataco"), 1)).To(Equal(7))
						Expect(file.Close()).To(Succeed())

						file = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						r = make([]byte, 10)
						Expect(file.Read(r)).To(Equal(10))
						Expect(r).To(Equal([]byte("\x00ocatacoat")))
						Expect(file.Close()).To(Succeed())
					})
				})

				// Added test to assert a fix to the behavior that memFS's open does not
				// open with the os.O_APPEND flag.
				Describe("Append mode", func() {
					It("Should append to the end of a file", func() {
						f := MustSucceed(
							fs.Open("test_file.txt", os.O_CREATE|os.O_WRONLY),
						)
						Expect(f.Write([]byte("oldoldold"))).To(Equal(9))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						stats := MustSucceed(fs.Stat("test_file.txt"))
						Expect(stats.Size()).To(BeEquivalentTo(9))
						buf := make([]byte, 9)
						Expect(f.Read(buf)).To(Equal(9))
						Expect(buf).To(Equal([]byte("oldoldold")))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDWR|os.O_APPEND))
						Expect(f.Write([]byte("newnew"))).To(Equal(6))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						stats = MustSucceed(fs.Stat("test_file.txt"))
						Expect(stats.Size()).To(BeEquivalentTo(15))
						buf = make([]byte, 15)
						Expect(f.Read(buf)).To(Equal(15))
						Expect(buf).To(Equal([]byte("oldoldoldnewnew")))
						Expect(f.Close()).To(Succeed())

						Expect(xfs.Default.Remove("testData")).To(Succeed())
					})
					It("Should overwrite when there is no Append flag", func() {
						f := MustSucceed(
							fs.Open("test_file.txt", os.O_CREATE|os.O_WRONLY),
						)
						Expect(f.Write([]byte("oldoldold"))).To(Equal(9))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						stats := MustSucceed(fs.Stat("test_file.txt"))
						Expect(stats.Size()).To(BeEquivalentTo(9))
						buf := make([]byte, 9)
						Expect(f.Read(buf)).To(Equal(9))
						Expect(buf).To(Equal([]byte("oldoldold")))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDWR))
						Expect(f.Write([]byte("newnew"))).To(Equal(6))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						stats = MustSucceed(fs.Stat("test_file.txt"))
						Expect(stats.Size()).To(BeEquivalentTo(9))
						buf = make([]byte, 9)
						Expect(f.Read(buf)).To(Equal(9))
						Expect(buf).To(Equal([]byte("newnewold")))
						Expect(f.Close()).To(Succeed())
					})
					It("Should work for a combination of APPEND and not APPEND", func() {
						f := MustSucceed(
							fs.Open("test_file.txt", os.O_CREATE|os.O_WRONLY),
						)
						Expect(f.Write([]byte("oldoldold"))).To(Equal(9))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						stats := MustSucceed(fs.Stat("test_file.txt"))
						Expect(stats.Size()).To(BeEquivalentTo(9))
						buf := make([]byte, 9)
						Expect(f.Read(buf)).To(Equal(9))
						Expect(buf).To(Equal([]byte("oldoldold")))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_WRONLY))
						Expect(f.Write([]byte("newnew"))).To(Equal(6))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						stats = MustSucceed(fs.Stat("test_file.txt"))
						Expect(stats.Size()).To(BeEquivalentTo(9))
						Expect(f.Read(buf)).To(Equal(9))
						Expect(buf).To(Equal([]byte("newnewold")))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(
							fs.Open("test_file.txt", os.O_WRONLY|os.O_APPEND),
						)
						Expect(f.Write([]byte("brandnew"))).To(Equal(8))
						Expect(f.Write([]byte("haha"))).To(Equal(4))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						stats = MustSucceed(fs.Stat("test_file.txt"))
						Expect(stats.Size()).To(BeEquivalentTo(21))
						buf = make([]byte, 21)
						Expect(f.Read(buf)).To(Equal(21))
						Expect(buf).To(Equal([]byte("newnewoldbrandnewhaha")))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_WRONLY))
						Expect(f.Write([]byte("ipromise"))).To(Equal(8))
						Expect(f.Write([]byte("n"))).To(Equal(1))
						Expect(f.Write([]byte("e"))).To(Equal(1))
						Expect(f.Write([]byte("w"))).To(Equal(1))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						stats = MustSucceed(fs.Stat("test_file.txt"))
						Expect(stats.Size()).To(BeEquivalentTo(21))
						buf = make([]byte, 21)
						Expect(f.Read(buf)).To(Equal(21))
						Expect(buf).To(Equal([]byte("ipromisenewandnewhaha")))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(
							fs.Open("test_file.txt", os.O_WRONLY|os.O_APPEND),
						)
						Expect(f.Write([]byte("t"))).To(Equal(1))
						Expect(f.Write([]byte("e"))).To(Equal(1))
						Expect(f.Write([]byte("a"))).To(Equal(1))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						stats = MustSucceed(fs.Stat("test_file.txt"))
						Expect(stats.Size()).To(BeEquivalentTo(24))
						buf = make([]byte, 24)
						Expect(f.Read(buf)).To(Equal(24))
						Expect(buf).To(Equal([]byte("ipromisenewandnewhahatea")))
						Expect(f.Close()).To(Succeed())
					})
				})
				Describe("Truncate mode", func() {
					It("Should truncate an existing file to zero length with O_TRUNC", func() {
						f := MustSucceed(
							fs.Open("test_file.txt", os.O_CREATE|os.O_WRONLY),
						)
						Expect(f.Write([]byte("original content"))).To(Equal(16))
						Expect(f.Close()).To(Succeed())

						stats := MustSucceed(fs.Stat("test_file.txt"))
						Expect(stats.Size()).To(BeEquivalentTo(16))

						f = MustSucceed(
							fs.Open("test_file.txt", os.O_WRONLY|os.O_TRUNC),
						)
						Expect(f.Close()).To(Succeed())

						stats = MustSucceed(fs.Stat("test_file.txt"))
						Expect(stats.Size()).To(BeZero())

						f = MustSucceed(fs.Open("test_file.txt", os.O_WRONLY))
						Expect(f.Write([]byte("new content"))).To(Equal(11))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						buf := make([]byte, 11)
						Expect(f.Read(buf)).To(Equal(11))
						Expect(buf).To(Equal([]byte("new content")))
						Expect(f.Close()).To(Succeed())
					})

					It("Should truncate and write in a single operation", func() {
						f := MustSucceed(
							fs.Open("test_file.txt", os.O_CREATE|os.O_WRONLY),
						)
						Expect(f.Write([]byte("old data that should be gone"))).
							To(Equal(28))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(
							fs.Open("test_file.txt", os.O_WRONLY|os.O_TRUNC),
						)
						Expect(f.Write([]byte("new data"))).To(Equal(8))
						Expect(f.Close()).To(Succeed())

						stats := MustSucceed(fs.Stat("test_file.txt"))
						Expect(stats.Size()).To(BeEquivalentTo(8))
						f = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						buf := make([]byte, 8)
						Expect(f.Read(buf)).To(Equal(8))
						Expect(buf).To(Equal([]byte("new data")))
						Expect(f.Close()).To(Succeed())
					})
				})
			})
			Describe("Sub", func() {
				It("Should make subdirectories", func() {
					Expect(fs.Sub("sub1")).To(Not(BeNil()))
					Expect(fs.Exists("sub1")).To(BeTrue())
					Expect(fs.Exists("sub2")).To(BeFalse())

					Expect(fs.Sub("sub2")).To(Not(BeNil()))
					Expect(fs.Exists("sub1")).To(BeTrue())
					Expect(fs.Exists("sub2")).To(BeTrue())
				})
				It("Should give FS of subdirectories", func() {
					Expect(fs.Sub("sub1")).To(Not(BeNil()))
					Expect(fs.Exists("sub1")).To(BeTrue())
					Expect(fs.Exists("sub2")).To(BeFalse())

					sub_FS := MustSucceed(fs.Sub("sub2"))
					Expect(fs.Exists("sub1")).To(BeTrue())
					Expect(fs.Exists("sub2")).To(BeTrue())

					f := MustSucceed(sub_FS.Open("yum.txt", os.O_CREATE))
					Expect(f.Close()).To(Succeed())
					Expect(sub_FS.Exists("yum.txt")).To(BeTrue())
				})
				It("Should correctly interpret relative paths", func() {
					Expect(fs.Sub("./sub1")).To(Not(BeNil()))
					l := MustSucceed(fs.List(""))
					Expect(l).To(HaveLen(1))
					Expect(l[0].Name()).To(Equal("sub1"))
				})
			})
			var _ = Describe("Exists", func() {
				It("Should return false if a file does not exist and true if it does", func() {
					Expect(fs.Exists("yum.txt")).To(BeFalse())
					f := MustSucceed(fs.Open("yum.txt", os.O_CREATE))
					Expect(f.Close()).To(Succeed())
					Expect(fs.Exists("yum.txt")).To(BeTrue())
				})
				It("Should return false if a directory does not exist and true if it does", func() {
					Expect(fs.Exists("yum")).To(BeFalse())
					Expect(fs.Sub("yum")).To(Not(BeNil()))
					Expect(fs.Exists("yum")).To(BeTrue())
				})
			})
			Describe("List", func() {
				It("Should provide a list of all the files and directories", func() {
					Expect(fs.Sub("sub1")).To(Not(BeNil()))
					Expect(fs.Sub("sub2")).To(Not(BeNil()))
					f := MustSucceed(fs.Open("file1.json", os.O_CREATE))
					Expect(f.Close()).To(Succeed())

					f = MustSucceed(fs.Open("file2.json", os.O_CREATE))
					Expect(f.Close()).To(Succeed())

					l := MustSucceed(fs.List(""))

					Expect(l[0].Name()).To(Equal("file1.json"))
					Expect(l[0].IsDir()).To(BeFalse())

					Expect(l[1].Name()).To(Equal("file2.json"))
					Expect(l[1].IsDir()).To(BeFalse())

					Expect(l[2].Name()).To(Equal("sub1"))
					Expect(l[2].IsDir()).To(BeTrue())

					Expect(l[3].Name()).To(Equal("sub2"))
					Expect(l[3].IsDir()).To(BeTrue())
				})

				It("Should correctly list files in sub-filesystems", func() {
					subFS := MustSucceed(fs.Sub("sub1"))
					subFS2 := MustSucceed(subFS.Sub("sub2"))
					f := MustSucceed(subFS2.Open("file1.json", os.O_CREATE))
					l := MustSucceed(subFS2.List(""))
					Expect(l).To(HaveLen(1))
					Expect(l[0].Name()).To(Equal("file1.json"))
					Expect(f.Close()).To(Succeed())
				})
				Context("Windows Regression", func() {
					It("Should list file sizes in the directory correctly", func() {
						// Windows has a different implementation of DirEntry.Info()
						// than other OS's - one of its problems is that it has trouble
						// returning the size of a file that still has handles open to
						// it. We avoided it by calling os.Stat() instead in our
						// implementation of List().
						subFS := MustSucceed(fs.Sub("sub1"))
						f := MustSucceed(subFS.Open("file1.txt", os.O_CREATE|os.O_RDWR))
						Expect(f.Write([]byte("tacocat"))).To(Equal(7))
						l := MustSucceed(subFS.List(""))
						s := MustSucceed(f.Stat())
						Expect(s.Size()).To(BeEquivalentTo(7))
						Expect(l[0].Size()).To(BeEquivalentTo(7))
						Expect(f.Close()).To(Succeed())
					})
				})
			})
			Describe("Rename", func() {
				It("Should rename a file for Mem FS", func() {
					f := MustSucceed(fs.Open("a.json", os.O_CREATE))
					Expect(f.Close()).To(Succeed())
					Expect(fs.Rename("a.json", "b.json")).To(Succeed())
					Expect(fs.Exists("a.json")).To(BeFalse())
					Expect(fs.Exists("b.json")).To(BeTrue())
				})
				It("Should rename a directory for Mem FS", func() {
					Expect(fs.Sub("a")).To(Not(BeNil()))
					Expect(fs.Rename("a", "b")).To(Succeed())
					Expect(fs.Exists("a")).To(BeFalse())
					Expect(fs.Exists("b")).To(BeTrue())
				})
			})
			Describe("Truncate", func() {
				It("Should truncate a file when the size is smaller than original", func() {
					f := MustSucceed(fs.Open("b.txt", os.O_CREATE|os.O_RDWR))
					Expect(f.Write([]byte("hello world"))).To(Equal(11))
					Expect(f.Truncate(5)).To(Succeed())
					stats := MustSucceed(f.Stat())
					Expect(stats.Size()).To(BeEquivalentTo(5))
					Expect(f.Close()).To(Succeed())

					f = MustSucceed(fs.Open("b.txt", os.O_RDONLY))
					buf := make([]byte, 5)
					Expect(f.Read(buf)).To(Equal(5))
					Expect(f.Close()).To(Succeed())
					Expect(buf).To(Equal([]byte("hello")))
				})
				It("Should not reset the file handle after down-truncate", func() {
					f := MustSucceed(fs.Open("b.txt", os.O_CREATE|os.O_RDWR))
					Expect(f.Write([]byte("helloworld"))).To(Equal(10))
					Expect(f.Truncate(5)).To(Succeed())
					buf := []byte("hihihi")
					Expect(f.Write(buf)).To(Equal(6))
					Expect(f.Close()).To(Succeed())

					f = MustSucceed(fs.Open("b.txt", os.O_RDONLY))
					buf = make([]byte, 16)
					Expect(f.Read(buf)).To(Equal(16))
					Expect(f.Close()).To(Succeed())
					Expect(buf).To(Equal([]byte("hello\x00\x00\x00\x00\x00hihihi")))
				})
				It("Should not reset the file handle after up-truncate", func() {
					f := MustSucceed(fs.Open("c.txt", os.O_CREATE|os.O_WRONLY))
					buf := []byte("hehehe")
					Expect(f.Write(buf)).To(Equal(6))
					Expect(f.Truncate(15)).To(Succeed())
					buf = []byte("haha")
					Expect(f.Write(buf)).To(Equal(4))
					Expect(f.Close()).To(Succeed())

					buf = make([]byte, 15)
					f = MustSucceed(fs.Open("c.txt", os.O_RDONLY))
					Expect(f.Read(buf)).To(Equal(15))
					Expect(f.Close()).To(Succeed())
					Expect(buf).To(Equal([]byte("hehehehaha\x00\x00\x00\x00\x00")))
				})
				It("Should extend a file when the size is bigger than original", func() {
					f := MustSucceed(fs.Open("d.txt", os.O_CREATE|os.O_RDWR))
					Expect(f.Write([]byte("hello"))).To(Equal(5))
					Expect(f.Truncate(10)).To(Succeed())
					stats := MustSucceed(f.Stat())
					Expect(stats.Size()).To(BeEquivalentTo(10))
					Expect(f.Close()).To(Succeed())
					f = MustSucceed(fs.Open("d.txt", os.O_RDONLY))
					buf := make([]byte, 10)
					Expect(f.Read(buf)).To(Equal(10))
					Expect(f.Close()).To(Succeed())
					Expect(buf).To(Equal([]byte("hello\x00\x00\x00\x00\x00")))
				})
			})
		})
	}
})
