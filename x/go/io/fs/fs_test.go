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

var _ = Describe("FS", func() {
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
		)

		Context("FS:"+fsName, Ordered, func() {
			BeforeEach(func() {
				fsRoot = makeFS()
				fs = MustSucceed(fsRoot.Sub("test-spec"))
			})
			AfterEach(func() {
				Expect(fsRoot.Remove("test-spec")).To(Succeed())
			})
			AfterAll(func() { Expect(xfs.Default.Remove("testData")).To(Succeed()) })
			Describe("Open", func() {
				Describe("Create test CREATE flag", func() {
					It("Should create a file", func() {
						file, err := fs.Open("test_file.txt", os.O_CREATE)
						Expect(err).ToNot(HaveOccurred())
						fileStats, _ := file.Stat()
						Expect(fileStats.Name()).To(Equal("test_file.txt"))
						Expect(file.Close()).To(Succeed())
					})
				})
				Describe("Create test without create flag", func() {
					It("Should not create a file", func() {
						_, err := fs.Open("test_file.txt", os.O_RDONLY)
						Expect(err).To(MatchError(os.ErrNotExist))
					})
					It("Should not create a file under write mode", func() {
						_, err := fs.Open("test_file.txt", os.O_WRONLY)
						Expect(err).To(MatchError(os.ErrNotExist))
						Expect(MustSucceed(fs.Exists("test_file.txt"))).To(BeFalse())
					})
					It("Should not create a file under read write mode", func() {
						_, err := fs.Open("test_file.txt", os.O_RDWR)
						Expect(err).To(MatchError(os.ErrNotExist))
						Expect(MustSucceed(fs.Exists("test_file.txt"))).To(BeFalse())
					})
				})
				Describe("Create test with exclusive", func() {
					It("Should create a file in MemFS", func() {
						f, err := fs.Open("test_file.txt", os.O_CREATE|os.O_EXCL)
						Expect(err).ToNot(HaveOccurred())
						Expect(f.Close()).To(Succeed())
					})
				})
				Describe("Read file test", func() {
					It("Should read a file in MemFS", func() {
						f, err := fs.Open("test_file.txt", os.O_CREATE)
						Expect(err).ToNot(HaveOccurred())
						Expect(f.Close()).To(Succeed())
						file, err := fs.Open("test_file.txt", os.O_RDONLY)
						Expect(err).ToNot(HaveOccurred())
						info, _ := file.Stat()
						Expect(info.Name()).To(Equal("test_file.txt"))
						Expect(file.Close()).To(Succeed())
					})

					It("Should read a file even when passed in with create", func() {
						f, err := fs.Open("test_file.txt", os.O_CREATE|os.O_RDWR)
						Expect(err).ToNot(HaveOccurred())
						Expect(MustSucceed(f.Write([]byte{1, 2, 3, 4}))).To(Equal(4))
						Expect(f.Close()).To(Succeed())
						f, err = fs.Open("test_file.txt", os.O_CREATE|os.O_RDWR)
						Expect(MustSucceed(f.Stat()).Size()).To(Equal(int64(4)))
						var buf = make([]byte, 4)
						Expect(MustSucceed(f.Read(buf))).To(Equal(4))
						Expect(buf).To(Equal([]byte{1, 2, 3, 4}))
						Expect(f.Close()).To(Succeed())
					})
				})

				Describe("Read nonexistent file test", func() {
					It("Should break when no such file exists", func() {
						_, err := fs.Open("test_file.txt", os.O_RDONLY)
						Expect(err).ToNot(BeNil())
					})
				})

				Describe("Reader/Writer test", func() {
					It("Should write & read the contents of file in MemFS", func() {
						f, err := fs.Open("test_file.txt", os.O_CREATE)
						Expect(err).ToNot(HaveOccurred())
						Expect(f.Close()).To(Succeed())

						file, err := fs.Open("test_file.txt", os.O_RDWR)
						Expect(err).ToNot(HaveOccurred())
						n, err := file.Write([]byte("tacocat"))
						Expect(err).ToNot(HaveOccurred())
						Expect(n).To(Equal(7))
						Expect(file.Close()).To(Succeed())

						file, err = fs.Open("test_file.txt", os.O_RDONLY)
						r := make([]byte, 7)
						n, err = file.Read(r)
						Expect(err).ToNot(HaveOccurred())
						Expect(n).To(Equal(7))
						Expect(r).To(Equal([]byte("tacocat")))
						Expect(file.Close()).To(Succeed())
					})
				})

				Describe("Reader/Writer without permission", func() {
					It("Should read but not write to file in MemFS", func() {
						f, err := fs.Open("test_file.txt", os.O_CREATE)
						Expect(err).ToNot(HaveOccurred())
						Expect(f.Close()).To(Succeed())

						file, err := fs.Open("test_file.txt", os.O_RDONLY)
						Expect(err).ToNot(HaveOccurred())
						n, err := file.Write([]byte("tacocat"))
						Expect(err).ToNot(BeNil())
						Expect(n).To(Equal(0))
						Expect(f.Close()).To(Succeed())

						file, err = fs.Open("test_file.txt", os.O_WRONLY)
						Expect(err).ToNot(HaveOccurred())
						n, err = file.Write([]byte("tacocat"))
						Expect(err).ToNot(HaveOccurred())
						Expect(n).To(Equal(7))
						Expect(file.Close()).To(Succeed())

						file, err = fs.Open("test_file.txt", os.O_RDONLY)
						r := make([]byte, 7)
						n, err = file.Read(r)
						Expect(err).ToNot(HaveOccurred())
						Expect(n).To(Equal(7))
						Expect(r).To(Equal([]byte("tacocat")))
						Expect(file.Close()).To(Succeed())
					})
				})

				Describe("ReaderAt/WriterAt test", func() {
					It("Should write & read the contents of file in MemFS", func() {
						f, err := fs.Open("test_file.txt", os.O_CREATE)
						Expect(err).ToNot(HaveOccurred())
						Expect(f.Close()).To(Succeed())

						// WRITE
						file, err := fs.Open("test_file.txt", os.O_RDWR)
						Expect(err).ToNot(HaveOccurred())
						n, err := file.WriteAt([]byte("tacocat"), 3)
						Expect(err).ToNot(HaveOccurred())
						Expect(n).To(Equal(7))
						Expect(file.Close()).To(Succeed())

						// READ
						file, err = fs.Open("test_file.txt", os.O_RDONLY)
						r := make([]byte, 10)
						n, err = file.Read(r)
						Expect(err).ToNot(HaveOccurred())
						Expect(n).To(Equal(10))
						Expect(r).To(Equal([]byte{0, 0, 0, 't', 'a', 'c', 'o', 'c', 'a', 't'}))
						Expect(file.Close()).To(Succeed())

						// WRITE
						file, err = fs.Open("test_file.txt", os.O_RDWR)
						Expect(err).ToNot(HaveOccurred())
						n, err = file.WriteAt([]byte("ocataco"), 1)
						Expect(err).ToNot(HaveOccurred())
						Expect(n).To(Equal(7))
						Expect(file.Close()).To(Succeed())

						// READ
						file, err = fs.Open("test_file.txt", os.O_RDONLY)
						r = make([]byte, 10)
						n, err = file.Read(r)
						Expect(err).ToNot(HaveOccurred())
						Expect(n).To(Equal(10))
						Expect(r).To(Equal([]byte{0, 'o', 'c', 'a', 't', 'a', 'c', 'o', 'a', 't'}))
						Expect(file.Close()).To(Succeed())
					})
				})

				// Added test to assert a fix to the behaviour that memFS's open does not
				// open with the os.O_APPEND flag.
				Describe("Append mode", func() {
					It("Should append to the end of a file", func() {
						f := MustSucceed(fs.Open("test_file.txt", os.O_CREATE|os.O_WRONLY))
						_, err := f.Write([]byte("oldoldold"))
						Expect(err).ToNot(HaveOccurred())
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						var buf = make([]byte, 9)
						Expect(MustSucceed(fs.Stat("test_file.txt")).Size()).To(Equal(int64(9)))
						Expect(MustSucceed(f.Read(buf))).To(Equal(9))
						Expect(buf).To(Equal([]byte("oldoldold")))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDWR|os.O_APPEND))
						_, err = f.Write([]byte("newnew"))
						Expect(err).ToNot(HaveOccurred())
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						buf = make([]byte, 15)
						Expect(MustSucceed(fs.Stat("test_file.txt")).Size()).To(Equal(int64(15)))
						Expect(MustSucceed(f.Read(buf))).To(Equal(15))
						Expect(buf).To(Equal([]byte("oldoldoldnewnew")))
						Expect(f.Close()).To(Succeed())

						Expect(xfs.Default.Remove("testData")).To(Succeed())
					})
					It("Should overwrite when there is no Append flag", func() {
						f := MustSucceed(fs.Open("test_file.txt", os.O_CREATE|os.O_WRONLY))
						_, err := f.Write([]byte("oldoldold"))
						Expect(err).ToNot(HaveOccurred())
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						var buf = make([]byte, 9)
						Expect(MustSucceed(fs.Stat("test_file.txt")).Size()).To(Equal(int64(9)))
						Expect(MustSucceed(f.Read(buf))).To(Equal(9))
						Expect(buf).To(Equal([]byte("oldoldold")))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDWR))
						_, err = f.Write([]byte("newnew"))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						buf = make([]byte, 9)
						Expect(MustSucceed(fs.Stat("test_file.txt")).Size()).To(Equal(int64(9)))
						Expect(MustSucceed(f.Read(buf))).To(Equal(9))
						Expect(buf).To(Equal([]byte("newnewold")))
						Expect(f.Close()).To(Succeed())
					})
					It("Should work for a combination of APPEND and not APPEND", func() {
						f := MustSucceed(fs.Open("test_file.txt", os.O_CREATE|os.O_WRONLY))
						_, err := f.Write([]byte("oldoldold"))
						Expect(err).ToNot(HaveOccurred())
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						var buf = make([]byte, 9)
						Expect(MustSucceed(fs.Stat("test_file.txt")).Size()).To(Equal(int64(9)))
						Expect(MustSucceed(f.Read(buf))).To(Equal(9))
						Expect(buf).To(Equal([]byte("oldoldold")))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_WRONLY))
						_, err = f.Write([]byte("newnew"))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						buf = make([]byte, 9)
						Expect(MustSucceed(fs.Stat("test_file.txt")).Size()).To(Equal(int64(9)))
						Expect(MustSucceed(f.Read(buf))).To(Equal(9))
						Expect(buf).To(Equal([]byte("newnewold")))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_WRONLY|os.O_APPEND))
						_, err = f.Write([]byte("brandnew"))
						_, err = f.Write([]byte("haha"))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						buf = make([]byte, 21)
						Expect(MustSucceed(fs.Stat("test_file.txt")).Size()).To(Equal(int64(21)))
						Expect(MustSucceed(f.Read(buf))).To(Equal(21))
						Expect(buf).To(Equal([]byte("newnewoldbrandnewhaha")))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_WRONLY))
						_, err = f.Write([]byte("ipromise"))
						_, err = f.Write([]byte("n"))
						_, err = f.Write([]byte("e"))
						_, err = f.Write([]byte("w"))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						buf = make([]byte, 21)
						Expect(MustSucceed(fs.Stat("test_file.txt")).Size()).To(Equal(int64(21)))
						Expect(MustSucceed(f.Read(buf))).To(Equal(21))
						Expect(buf).To(Equal([]byte("ipromisenewandnewhaha")))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_WRONLY|os.O_APPEND))
						_, err = f.Write([]byte("t"))
						_, err = f.Write([]byte("e"))
						_, err = f.Write([]byte("a"))
						Expect(f.Close()).To(Succeed())

						f = MustSucceed(fs.Open("test_file.txt", os.O_RDONLY))
						buf = make([]byte, 24)
						Expect(MustSucceed(fs.Stat("test_file.txt")).Size()).To(Equal(int64(24)))
						Expect(MustSucceed(f.Read(buf))).To(Equal(24))
						Expect(buf).To(Equal([]byte("ipromisenewandnewhahatea")))
						Expect(f.Close()).To(Succeed())
					})
				})
			})

			Describe("Sub", func() {
				It("Should make subdirectories", func() {
					_, err := fs.Sub("sub1")
					Expect(err).ToNot(HaveOccurred())

					Expect(fs.Exists("sub1")).To(BeTrue())
					Expect(fs.Exists("sub2")).To(BeFalse())

					_, err = fs.Sub("sub2")
					Expect(err).ToNot(HaveOccurred())

					Expect(fs.Exists("sub1")).To(BeTrue())
					Expect(fs.Exists("sub2")).To(BeTrue())
				})

				It("Should give FS of subdirectories", func() {
					sub_FS, err := fs.Sub("sub1")
					Expect(err).ToNot(HaveOccurred())

					Expect(fs.Exists("sub1")).To(BeTrue())
					Expect(fs.Exists("sub2")).To(BeFalse())

					_, err = fs.Sub("sub2")
					Expect(err).ToNot(HaveOccurred())

					Expect(fs.Exists("sub1")).To(BeTrue())
					Expect(fs.Exists("sub2")).To(BeTrue())

					f, err := sub_FS.Open("yum.txt", os.O_CREATE)
					Expect(err).ToNot(HaveOccurred())
					Expect(f.Close()).To(Succeed())
					Expect(sub_FS.Exists("yum.txt")).To(BeTrue())
				})

				It("Should correctly interpret relative paths", func() {
					MustSucceed(fs.Sub("./sub1"))
					l, err := fs.List("")
					Expect(err).ToNot(HaveOccurred())
					Expect(l).To(HaveLen(1))
					Expect(l[0].Name()).To(Equal("sub1"))
				})
			})

			var _ = Describe("Exists", func() {
				It("Should return false if a file does not exist and true if it does", func() {
					e, err := fs.Exists("yum.txt")
					Expect(err).ToNot(HaveOccurred())
					Expect(e).To(BeFalse())

					f, err := fs.Open("yum.txt", os.O_CREATE)
					Expect(err).ToNot(HaveOccurred())
					Expect(f.Close()).To(Succeed())

					e, err = fs.Exists("yum.txt")
					Expect(err).ToNot(HaveOccurred())
					Expect(e).To(BeTrue())
				})

				It("Should return false if a directory does not exist and true if it does", func() {
					e, err := fs.Exists("yum")
					Expect(err).ToNot(HaveOccurred())
					Expect(e).To(BeFalse())

					_, err = fs.Sub("yum")
					Expect(err).ToNot(HaveOccurred())

					e, err = fs.Exists("yum")
					Expect(err).ToNot(HaveOccurred())
					Expect(e).To(BeTrue())
				})
			})

			Describe("List", func() {
				It("Should provide a list of all the files and directories", func() {
					_, err := fs.Sub("sub1")
					Expect(err).ToNot(HaveOccurred())

					_, err = fs.Sub("sub2")
					Expect(err).ToNot(HaveOccurred())

					f, err := fs.Open("file1.json", os.O_CREATE)
					Expect(err).ToNot(HaveOccurred())
					Expect(f.Close()).To(Succeed())

					f, err = fs.Open("file2.json", os.O_CREATE)
					Expect(err).ToNot(HaveOccurred())
					Expect(f.Close()).To(Succeed())

					l, err := fs.List("")
					Expect(err).ToNot(HaveOccurred())

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
					subFS, err := fs.Sub("sub1")
					Expect(err).ToNot(HaveOccurred())
					subFS2, err := subFS.Sub("sub2")
					Expect(err).ToNot(HaveOccurred())
					f := MustSucceed(subFS2.Open("file1.json", os.O_CREATE))
					l := MustSucceed(subFS2.List(""))
					Expect(l).To(HaveLen(1))
					Expect(l[0].Name()).To(Equal("file1.json"))
					Expect(f.Close()).To(Succeed())
				})
			})

			Describe("Rename", func() {
				It("Should rename a file for Mem FS", func() {
					f, err := fs.Open("a.json", os.O_CREATE)
					Expect(err).ToNot(HaveOccurred())
					Expect(f.Close()).To(Succeed())
					err = fs.Rename("a.json", "b.json")
					Expect(err).ToNot(HaveOccurred())
					Expect(fs.Exists("a.json")).To(BeFalse())
					Expect(fs.Exists("b.json")).To(BeTrue())
				})

				It("Should rename a directory for Mem FS", func() {
					_, err := fs.Sub("a")
					Expect(err).ToNot(HaveOccurred())
					err = fs.Rename("a", "b")
					Expect(err).ToNot(HaveOccurred())
					Expect(fs.Exists("a")).To(BeFalse())
					Expect(fs.Exists("b")).To(BeTrue())
				})
			})

			Describe("Truncate", func() {
				It("Should not truncate a file without write perms", func() {
					f, err := fs.Open("a.txt", os.O_CREATE|os.O_RDONLY)
					Expect(err).ToNot(HaveOccurred())
					err = f.Truncate(5)
					if s := MustSucceed(fs.Stat("")); s.Sys() == nil {
						Expect(err).To(MatchError(ContainSubstring("fs: file was not created for writing (truncate requires write fd)")))
					} else {
						Expect(err).To(MatchError(ContainSubstring("invalid argument")))
					}
					Expect(f.Close()).To(Succeed())
				})
				It("Should truncate a file when the size is smaller than original", func() {
					f, err := fs.Open("b.txt", os.O_CREATE|os.O_RDWR)
					Expect(err).ToNot(HaveOccurred())
					_, err = f.Write([]byte("hello world"))
					Expect(err).ToNot(HaveOccurred())
					err = f.Truncate(5)
					Expect(err).ToNot(HaveOccurred())
					Expect(MustSucceed(f.Stat()).Size()).To(Equal(int64(5)))
					Expect(f.Close()).To(Succeed())

					buf := make([]byte, 5)
					f, err = fs.Open("b.txt", os.O_RDONLY)
					_, err = f.Read(buf)
					Expect(err).ToNot(HaveOccurred())
					Expect(f.Close()).To(Succeed())
					Expect(buf).To(Equal([]byte("hello")))
				})
				It("Should not reset the file handle after down-truncate", func() {
					f, err := fs.Open("b.txt", os.O_CREATE|os.O_RDWR)
					Expect(err).ToNot(HaveOccurred())
					_, err = f.Write([]byte("helloworld"))
					Expect(err).ToNot(HaveOccurred())
					err = f.Truncate(5)
					Expect(err).ToNot(HaveOccurred())
					buf := []byte("hihihi")
					_, err = f.Write(buf)
					Expect(err).ToNot(HaveOccurred())
					Expect(f.Close()).To(Succeed())

					buf = make([]byte, 16)
					f, err = fs.Open("b.txt", os.O_RDONLY)
					_, err = f.Read(buf)
					Expect(err).ToNot(HaveOccurred())
					Expect(f.Close()).To(Succeed())
					Expect(buf).To(Equal([]byte("hello\x00\x00\x00\x00\x00hihihi")))
				})
				It("Should not reset the file handle after up-truncate", func() {
					f, err := fs.Open("c.txt", os.O_CREATE|os.O_WRONLY)
					Expect(err).ToNot(HaveOccurred())
					buf := []byte("hehehe")
					_, err = f.Write(buf)
					Expect(err).ToNot(HaveOccurred())

					Expect(f.Truncate(15)).To(Succeed())

					buf = []byte("haha")
					_, err = f.Write(buf)
					Expect(err).ToNot(HaveOccurred())
					Expect(f.Close()).To(Succeed())

					buf = make([]byte, 15)
					f, err = fs.Open("c.txt", os.O_RDONLY)
					_, err = f.Read(buf)
					Expect(err).ToNot(HaveOccurred())
					Expect(f.Close()).To(Succeed())
					Expect(buf).To(Equal([]byte("hehehehaha\x00\x00\x00\x00\x00")))
				})
				It("Should extend a file when the size is bigger than original", func() {
					f, err := fs.Open("d.txt", os.O_CREATE|os.O_RDWR)
					Expect(err).ToNot(HaveOccurred())
					_, err = f.Write([]byte("hello"))
					Expect(err).ToNot(HaveOccurred())
					err = f.Truncate(10)
					Expect(err).ToNot(HaveOccurred())
					Expect(MustSucceed(f.Stat()).Size()).To(Equal(int64(10)))
					Expect(f.Close()).To(Succeed())

					f, err = fs.Open("d.txt", os.O_RDONLY)
					Expect(err).ToNot(HaveOccurred())
					buf := make([]byte, 10)
					_, err = f.Read(buf)
					Expect(f.Close()).To(Succeed())
					Expect(buf).To(Equal([]byte("hello\x00\x00\x00\x00\x00")))
				})
			})
		})
	}
})
