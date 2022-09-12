package kfs_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/kfs"
)

var fsx = []func() kfs.BaseFS{
	func() kfs.BaseFS {
		fs := kfs.NewMem()
		_, err := fs.Create("./testdata/test.txt")
		Expect(err).ToNot(HaveOccurred())
		return fs
	},
	kfs.NewOS,
}

var _ = Describe("Os", func() {
	// iterate through each fs in fs
	for _, fsFunc := range fsx {
		fsFunc := fsFunc
		var (
			fs kfs.BaseFS
		)
		BeforeEach(func() { fs = fsFunc() })
		It("Should open a file for reading and writing", func() {
			f, err := fs.Open("./testdata/test.txt")
			Expect(err).To(BeNil())
			Expect(f).NotTo(BeNil())
			Expect(f.Close()).To(BeNil())
		})
		It("Should create and remove a file", func() {
			f, err := fs.Create("./testdata/ctest.txt")
			Expect(err).To(BeNil())
			Expect(f).NotTo(BeNil())
			Expect(f.Close()).To(BeNil())
			Expect(fs.Remove("./testdata/ctest.txt")).To(BeNil())
		})
		It("Should return stat information for a file", func() {
			fi, err := fs.Stat("./testdata/test.txt")
			Expect(err).To(BeNil())
			Expect(fi).NotTo(BeNil())
		})
	}

})
