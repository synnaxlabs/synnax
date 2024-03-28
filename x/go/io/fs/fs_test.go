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
	"github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
	"os"
)

var _ = Describe("FS", func() {
	fileSystems := []func() fs.FS{
		func() fs.FS {
			return fs.NewMem()
		},
		func() fs.FS {
			return MustSucceed(fs.Default.Sub("./testData"))
		},
	}

	for _, fsBuilder := range fileSystems {
		var myMemFS fs.FS
		BeforeEach(func() {
			myMemFS = fsBuilder()
		})
		AfterEach(func() {
			Expect(os.RemoveAll("./testData")).To(Succeed())
		})
		Describe("Open", func() {
			Describe("Create test CREATE flag", func() {
				It("Should create a file in MemFS", func() {
					file, _ := myMemFS.Open("test_file2.txt", os.O_CREATE)
					fileStats, _ := file.Stat()
					Expect(fileStats.Name()).To(Equal("test_file2.txt"))
				})

			})
			Describe("Create test without create flag", func() {
				It("Should not create a file in MemFS", func() {
					_, err := myMemFS.Open("test_file2.txt", os.O_RDONLY)
					Expect(err).ToNot(BeNil())
				})
			})
			Describe("Create test with exclusive", func() {
				It("Should create a file in MemFS", func() {
					_, err := myMemFS.Open("test_file2.txt", os.O_CREATE|os.O_EXCL)
					Expect(err).ToNot(HaveOccurred())
				})
			})
			Describe("Read file test", func() {
				It("Should read a file in MemFS", func() {
					_, err := myMemFS.Open("test_file.txt", os.O_CREATE)
					Expect(err).ToNot(HaveOccurred())
					file, err := myMemFS.Open("test_file.txt", os.O_RDONLY)
					Expect(err).ToNot(HaveOccurred())
					info, _ := file.Stat()
					Expect(info.Name()).To(Equal("test_file.txt"))
				})
			})

			Describe("Read nonexistent file test", func() {
				It("Should break when no such file exists", func() {
					_, err := myMemFS.Open("test_file.txt", os.O_RDONLY)
					Expect(err).ToNot(BeNil())
				})
			})

			Describe("Reader/Writer test", func() {
				It("Should write & read the contents of file in MemFS", func() {
					_, err := myMemFS.Open("test_file.txt", os.O_CREATE)
					Expect(err).ToNot(HaveOccurred())

					file, err := myMemFS.Open("test_file.txt", os.O_RDWR)
					Expect(err).ToNot(HaveOccurred())
					n, err := file.Write([]byte("tacocat"))
					Expect(err).ToNot(HaveOccurred())
					Expect(n).To(Equal(7))

					file, err = myMemFS.Open("test_file.txt", os.O_RDONLY)
					r := make([]byte, 7)
					n, err = file.Read(r)
					Expect(err).ToNot(HaveOccurred())
					Expect(n).To(Equal(7))
					Expect(r).To(Equal([]byte("tacocat")))
				})
			})

			Describe("Reader/Writer without permission", func() {
				It("Should read but not write to file in MemFS", func() {
					_, err := myMemFS.Open("test_file.txt", os.O_CREATE)
					Expect(err).ToNot(HaveOccurred())

					file, err := myMemFS.Open("test_file.txt", os.O_RDONLY)
					Expect(err).ToNot(HaveOccurred())
					n, err := file.Write([]byte("tacocat"))
					Expect(err).ToNot(BeNil())
					Expect(n).To(Equal(0))

					file, err = myMemFS.Open("test_file.txt", os.O_WRONLY)
					Expect(err).ToNot(HaveOccurred())
					n, err = file.Write([]byte("tacocat"))
					Expect(err).ToNot(HaveOccurred())
					Expect(n).To(Equal(7))

					file, err = myMemFS.Open("test_file.txt", os.O_RDONLY)
					r := make([]byte, 7)
					n, err = file.Read(r)
					Expect(err).ToNot(HaveOccurred())
					Expect(n).To(Equal(7))
					Expect(r).To(Equal([]byte("tacocat")))
				})
			})

			Describe("ReaderAt/WriterAt test", func() {
				It("Should write & read the contents of file in MemFS", func() {
					_, err := myMemFS.Open("test_file.txt", os.O_CREATE)
					Expect(err).ToNot(HaveOccurred())

					// WRITE
					file, err := myMemFS.Open("test_file.txt", os.O_RDWR)
					Expect(err).ToNot(HaveOccurred())
					n, err := file.WriteAt([]byte("tacocat"), 3)
					Expect(err).ToNot(HaveOccurred())
					Expect(n).To(Equal(7))

					// READ
					file, err = myMemFS.Open("test_file.txt", os.O_RDONLY)
					r := make([]byte, 10)
					n, err = file.Read(r)
					Expect(err).ToNot(HaveOccurred())
					Expect(n).To(Equal(10))
					Expect(r).To(Equal([]byte{0, 0, 0, 't', 'a', 'c', 'o', 'c', 'a', 't'}))

					// WRITE
					file, err = myMemFS.Open("test_file.txt", os.O_RDWR)
					Expect(err).ToNot(HaveOccurred())
					n, err = file.WriteAt([]byte("ocataco"), 1)
					Expect(err).ToNot(HaveOccurred())
					Expect(n).To(Equal(7))

					// READ
					file, err = myMemFS.Open("test_file.txt", os.O_RDONLY)
					r = make([]byte, 10)
					n, err = file.Read(r)
					Expect(err).ToNot(HaveOccurred())
					Expect(n).To(Equal(10))
					Expect(r).To(Equal([]byte{0, 'o', 'c', 'a', 't', 'a', 'c', 'o', 'a', 't'}))
				})
			})
		})

		Describe("Sub", func() {
			It("Should make subdirectories", func() {
				_, err := myMemFS.Sub("sub1")
				Expect(err).ToNot(HaveOccurred())

				Expect(myMemFS.Exists("sub1")).To(BeTrue())
				Expect(myMemFS.Exists("sub2")).To(BeFalse())

				_, err = myMemFS.Sub("sub2")
				Expect(err).ToNot(HaveOccurred())

				Expect(myMemFS.Exists("sub1")).To(BeTrue())
				Expect(myMemFS.Exists("sub2")).To(BeTrue())
			})

			It("Should give FS of subdirectories", func() {
				sub_FS, err := myMemFS.Sub("sub1")
				Expect(err).ToNot(HaveOccurred())

				Expect(myMemFS.Exists("sub1")).To(BeTrue())
				Expect(myMemFS.Exists("sub2")).To(BeFalse())

				_, err = myMemFS.Sub("sub2")
				Expect(err).ToNot(HaveOccurred())

				Expect(myMemFS.Exists("sub1")).To(BeTrue())
				Expect(myMemFS.Exists("sub2")).To(BeTrue())

				_, err = sub_FS.Open("yum.txt", os.O_CREATE)
				Expect(err).ToNot(HaveOccurred())
				Expect(sub_FS.Exists("yum.txt")).To(BeTrue())
			})

			It("Should correctly interpret relative paths", func() {
				MustSucceed(myMemFS.Sub("./sub1"))
				l, err := myMemFS.List("")
				Expect(err).ToNot(HaveOccurred())
				Expect(l).To(HaveLen(1))
				Expect(l[0].Name()).To(Equal("sub1"))
			})
		})

		var _ = Describe("Exists", func() {
			It("Should return false if a file does not exist and true if it does", func() {
				e, err := myMemFS.Exists("yum.txt")
				Expect(err).ToNot(HaveOccurred())
				Expect(e).To(BeFalse())

				_, err = myMemFS.Open("yum.txt", os.O_CREATE)
				Expect(err).ToNot(HaveOccurred())

				e, err = myMemFS.Exists("yum.txt")
				Expect(err).ToNot(HaveOccurred())
				Expect(e).To(BeTrue())
			})

			It("Should return false if a directory does not exist and true if it does", func() {
				e, err := myMemFS.Exists("yum")
				Expect(err).ToNot(HaveOccurred())
				Expect(e).To(BeFalse())

				_, err = myMemFS.Sub("yum")
				Expect(err).ToNot(HaveOccurred())

				e, err = myMemFS.Exists("yum")
				Expect(err).ToNot(HaveOccurred())
				Expect(e).To(BeTrue())
			})
		})

		Describe("List", func() {
			It("Should provide a list of all the files and directories", func() {
				_, err := myMemFS.Sub("sub1")
				Expect(err).ToNot(HaveOccurred())

				_, err = myMemFS.Sub("sub2")
				Expect(err).ToNot(HaveOccurred())

				_, err = myMemFS.Open("file1.json", os.O_CREATE)
				Expect(err).ToNot(HaveOccurred())

				_, err = myMemFS.Open("file2.json", os.O_CREATE)
				Expect(err).ToNot(HaveOccurred())

				l, err := myMemFS.List("")
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
				subFS, err := myMemFS.Sub("sub1")
				Expect(err).ToNot(HaveOccurred())
				subFS2, err := subFS.Sub("sub2")
				Expect(err).ToNot(HaveOccurred())
				MustSucceed(subFS2.Open("file1.json", os.O_CREATE))
				l := MustSucceed(subFS2.List(""))
				Expect(l).To(HaveLen(1))
				Expect(l[0].Name()).To(Equal("file1.json"))
			})

		})

		Describe("Rename", func() {
			It("Should rename a file for Mem FS", func() {
				_, err := myMemFS.Open("a.json", os.O_CREATE)
				Expect(err).To(BeNil())
				err = myMemFS.Rename("a.json", "b.json")
				Expect(err).To(BeNil())
				Expect(myMemFS.Exists("a.json")).To(BeFalse())
				Expect(myMemFS.Exists("b.json")).To(BeTrue())
			})

			It("Should rename a directory for Mem FS", func() {
				_, err := myMemFS.Sub("a")
				Expect(err).To(BeNil())
				err = myMemFS.Rename("a", "b")
				Expect(err).To(BeNil())
				Expect(myMemFS.Exists("a")).To(BeFalse())
				Expect(myMemFS.Exists("b")).To(BeTrue())
			})
		})
	}
})
