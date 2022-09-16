package kfs_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/alamos"
	"github.com/synnaxlabs/x/kfs"
)

var _ = Describe("Metrics", func() {
	var (
		baseFS kfs.BaseFS
		fs     kfs.FS[int]
		exp    alamos.Experiment
	)
	BeforeEach(func() {
		baseFS = kfs.NewMem()
		exp = alamos.New("metrics")
		var err error
		fs, err = kfs.New[int]("testdata", kfs.WithExtensionConfig(".metrics"), kfs.WithFS(baseFS), kfs.WithExperiment(exp))
		Expect(err).ToNot(HaveOccurred())
	})
	Describe("TryLock", func() {
		It("Should record the count and average time", func() {
			_, err := fs.Acquire(1)
			Expect(err).ToNot(HaveOccurred())
			fs.Release(1)
			m := fs.Metrics().Acquire
			Expect(m.Count()).To(Equal(1))
			Expect(m.Values()[0]).ToNot(BeZero())
		})
	})
	Describe("Unlock", func() {
		It("Should record the count and average time", func() {
			_, err := fs.Acquire(1)
			Expect(err).ToNot(HaveOccurred())
			fs.Release(1)
			m := fs.Metrics().Release
			Expect(m.Count()).To(Equal(1))
			Expect(m.Values()[0]).ToNot(BeZero())
		})
	})
})
