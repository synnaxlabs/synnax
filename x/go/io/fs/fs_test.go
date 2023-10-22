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
	"os"
)

var _ = Describe("Open", func() {
	Describe("Create test CREATE flag", func() {
		It("Should create a file in MemFS", func() {
			my_mem_fs := fs.NewMem()
			file, _ := my_mem_fs.Open("test_file2.txt", os.O_CREATE)
			file_stats, _ := file.Stat()
			Expect(file_stats.Name()).To(Equal("test_file2.txt"))
		})

	})
	Describe("Create test without create flag", func() {
		It("Should not create a file in MemFS", func() {
			my_mem_fs := fs.NewMem()
			_, err := my_mem_fs.Open("test_file2.txt", os.O_RDONLY)
			Expect(err).ToNot(BeNil())
		})
	})
	Describe("Create test with exclusive", func() {
		It("Should create a file in MemFS", func() {
			my_mem_fs := fs.NewMem()
			_, err := my_mem_fs.Open("test_file2.txt", os.O_CREATE|os.O_EXCL)
			Expect(err).ToNot(HaveOccurred())
		})
	})
	Describe("Read file test", func() {
		It("Should read a file in MemFS", func() {
			my_mem_fs := fs.NewMem()
			_, err := my_mem_fs.Open("test_file.txt", os.O_CREATE)
			Expect(err).ToNot(HaveOccurred())
			file, err := my_mem_fs.Open("test_file.txt", os.O_RDONLY)
			Expect(err).ToNot(HaveOccurred())
			info, _ := file.Stat()
			Expect(info.Name()).To(Equal("test_file.txt"))
		})
	})

	Describe("Read nonexistent file test", func() {
		It("Should break when no such file exists", func() {
			my_mem_fs := fs.NewMem()
			_, err := my_mem_fs.Open("test_file.txt", os.O_RDONLY)
			Expect(err).ToNot(BeNil())
		})
	})

	Describe("Reader/Writer test", func() {
		It("Should write & read the contents of file in MemFS", func() {
			my_mem_fs := fs.NewMem()
			_, err := my_mem_fs.Open("test_file.txt", os.O_CREATE)
			Expect(err).ToNot(HaveOccurred())

			file, err := my_mem_fs.Open("test_file.txt", os.O_RDWR)
			Expect(err).ToNot(HaveOccurred())
			n, err := file.Write([]byte("tacocat"))
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(7))

			file, err = my_mem_fs.Open("test_file.txt", os.O_RDONLY)
			r := make([]byte, 7)
			n, err = file.Read(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(7))
			Expect(r).To(Equal([]byte("tacocat")))
		})
	})

	Describe("Reader/Writer without permission", func() {
		It("Should read but not write to file in MemFS", func() {
			my_mem_fs := fs.NewMem()
			_, err := my_mem_fs.Open("test_file.txt", os.O_CREATE)
			Expect(err).ToNot(HaveOccurred())

			file, err := my_mem_fs.Open("test_file.txt", os.O_RDONLY)
			Expect(err).ToNot(HaveOccurred())
			n, err := file.Write([]byte("tacocat"))
			Expect(err).ToNot(BeNil())
			Expect(n).To(Equal(0))

			file, err = my_mem_fs.Open("test_file.txt", os.O_WRONLY)
			Expect(err).ToNot(HaveOccurred())
			n, err = file.Write([]byte("tacocat"))
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(7))

			file, err = my_mem_fs.Open("test_file.txt", os.O_RDONLY)
			r := make([]byte, 7)
			n, err = file.Read(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(7))
			Expect(r).To(Equal([]byte("tacocat")))
		})
	})

	Describe("ReaderAt/WriterAt test", func() {
		It("Should write & read the contents of file in MemFS", func() {
			my_mem_fs := fs.NewMem()
			_, err := my_mem_fs.Open("test_file.txt", os.O_CREATE)
			Expect(err).ToNot(HaveOccurred())

			// WRITE
			file, err := my_mem_fs.Open("test_file.txt", os.O_RDWR)
			Expect(err).ToNot(HaveOccurred())
			n, err := file.WriteAt([]byte("tacocat"), 3)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(7))

			// READ
			file, err = my_mem_fs.Open("test_file.txt", os.O_RDONLY)
			r := make([]byte, 10)
			n, err = file.Read(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(10))
			Expect(r).To(Equal([]byte{0, 0, 0, 't', 'a', 'c', 'o', 'c', 'a', 't'}))

			// WRITE
			file, err = my_mem_fs.Open("test_file.txt", os.O_RDWR)
			Expect(err).ToNot(HaveOccurred())
			n, err = file.WriteAt([]byte("ocataco"), 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(7))

			// READ
			file, err = my_mem_fs.Open("test_file.txt", os.O_RDONLY)
			r = make([]byte, 10)
			n, err = file.Read(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(10))
			Expect(r).To(Equal([]byte{0, 'o', 'c', 'a', 't', 'a', 'c', 'o', 'a', 't'}))
		})
	})
})

var _ = Describe("Sub", func() {
	It("Should make subdirectories", func() {
		my_mem_fs := fs.NewMem()
		_, err := my_mem_fs.Sub("sub1")
		Expect(err).ToNot(HaveOccurred())

		Expect(my_mem_fs.Exists("sub1")).To(BeTrue())
		Expect(my_mem_fs.Exists("sub2")).To(BeFalse())

		_, err = my_mem_fs.Sub("sub2")
		Expect(err).ToNot(HaveOccurred())

		Expect(my_mem_fs.Exists("sub1")).To(BeTrue())
		Expect(my_mem_fs.Exists("sub2")).To(BeTrue())
	})

	It("Should give FS of subdirectories", func() {
		my_mem_fs := fs.NewMem()
		sub_FS, err := my_mem_fs.Sub("sub1")
		Expect(err).ToNot(HaveOccurred())

		Expect(my_mem_fs.Exists("sub1")).To(BeTrue())
		Expect(my_mem_fs.Exists("sub2")).To(BeFalse())

		_, err = my_mem_fs.Sub("sub2")
		Expect(err).ToNot(HaveOccurred())

		Expect(my_mem_fs.Exists("sub1")).To(BeTrue())
		Expect(my_mem_fs.Exists("sub2")).To(BeTrue())

		_, err = sub_FS.Open("yum.txt", os.O_CREATE)
		Expect(err).ToNot(HaveOccurred())
		Expect(sub_FS.Exists("yum.txt")).To(BeTrue())
	})
})

var _ = Describe("Exists", func() {
	It("Should return false if a file does not exist and true if it does", func() {
		my_mem_fs := fs.NewMem()
		e, err := my_mem_fs.Exists("yum.txt")
		Expect(err).ToNot(HaveOccurred())
		Expect(e).To(BeFalse())

		_, err = my_mem_fs.Open("yum.txt", os.O_CREATE)
		Expect(err).ToNot(HaveOccurred())

		e, err = my_mem_fs.Exists("yum.txt")
		Expect(err).ToNot(HaveOccurred())
		Expect(e).To(BeTrue())
	})

	It("Should return false if a directory does not exist and true if it does", func() {
		my_mem_fs := fs.NewMem()
		e, err := my_mem_fs.Exists("yum")
		Expect(err).ToNot(HaveOccurred())
		Expect(e).To(BeFalse())

		_, err = my_mem_fs.Sub("yum")
		Expect(err).ToNot(HaveOccurred())

		e, err = my_mem_fs.Exists("yum")
		Expect(err).ToNot(HaveOccurred())
		Expect(e).To(BeTrue())
	})
})

var _ = Describe("List", func() {
	It("Should provide a list of all the files and directories", func() {
		my_mem_fs := fs.NewMem()
		_, err := my_mem_fs.Sub("sub1")
		Expect(err).ToNot(HaveOccurred())

		_, err = my_mem_fs.Sub("sub2")
		Expect(err).ToNot(HaveOccurred())

		_, err = my_mem_fs.Open("file1.json", os.O_CREATE)
		Expect(err).ToNot(HaveOccurred())

		_, err = my_mem_fs.Open("file2.json", os.O_CREATE)
		Expect(err).ToNot(HaveOccurred())

		l, err := my_mem_fs.List()
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
})
