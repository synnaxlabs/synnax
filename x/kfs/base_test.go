package kfs_test

import (
	"github.com/arya-analytics/x/kfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Os", func() {
	var (
		baseFS kfs.BaseFS
	)
	BeforeEach(func() {
		baseFS = kfs.NewOS()
	})
	It("Should open a file for reading and writing", func() {
		f, err := baseFS.Open("testdata/test.txt")
		Expect(err).To(BeNil())
		Expect(f).NotTo(BeNil())
		Expect(f.Close()).To(BeNil())
	})
	It("Should create and delete a file", func() {
		f, err := baseFS.Create("testdata/ntest.txt")
		Expect(err).To(BeNil())
		Expect(f).NotTo(BeNil())
		Expect(f.Close()).To(BeNil())
		Expect(baseFS.Remove("testdata/ntest.txt")).To(BeNil())
	})
})
