package kfs_test

import (
	"github.com/arya-analytics/x/kfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var fsx = []func() kfs.BaseFS{
	kfs.NewMem,
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
			f, err := fs.Create("testdata/test.txt")
			Expect(err).ToNot(HaveOccurred())
			Expect(f.Close()).ToNot(HaveOccurred())
			f, err = fs.Open("testdata/test.txt")
			Expect(err).To(BeNil())
			Expect(f).NotTo(BeNil())
			_, err = f.Write([]byte("Hello World"))
			Expect(err).To(BeNil())
			Expect(f.Close()).To(BeNil())
			Expect(fs.Remove("testdata/test.txt")).To(BeNil())
		})
		It("Should return stat information for a file", func() {
			f, err := fs.Create("testdata/test.txt")
			Expect(err).To(BeNil())
			Expect(f).NotTo(BeNil())
			Expect(f.Close()).To(BeNil())
			fi, err := fs.Stat("testdata/test.txt")
			Expect(err).To(BeNil())
			Expect(fi).NotTo(BeNil())
			Expect(fs.Remove("testdata/test.txt")).To(BeNil())
		})
	}

})
