package kfs_test

import (
	"context"
	"github.com/arya-analytics/x/kfs"
	"github.com/arya-analytics/x/signal"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("sync", func() {

	It("Should sync the contents of the file to the file system every interval", func() {
		ctx, cancel := signal.TODO()
		defer cancel()
		fs, err := kfs.New[int]("testdata", kfs.WithExtensionConfig(".test"), kfs.WithFS(kfs.NewMem()))
		Expect(err).ToNot(HaveOccurred())
		defer Expect(fs.RemoveAll()).To(Succeed())
		_, err = fs.Acquire(1)
		Expect(err).To(BeNil())
		_, err = fs.Acquire(2)
		Expect(err).To(BeNil())
		_, err = fs.Acquire(3)
		Expect(err).To(BeNil())
		fs.Release(1)
		fs.Release(2)
		fs.Release(3)
		time.Sleep(5 * time.Millisecond)
		Expect(fs.OpenFiles()[1].Age() > 5*time.Millisecond).To(BeTrue())
		sync := &kfs.Sync[int]{
			FS:       fs,
			Interval: 2 * time.Millisecond,
			MaxAge:   2 * time.Millisecond,
		}
		errs := sync.Start(ctx)
		go func() {
			defer GinkgoRecover()
			Expect(<-errs).ToNot(HaveOccurred())
		}()
		time.Sleep(6 * time.Millisecond)
		fOne := fs.OpenFiles()[1]
		Expect(fOne.Age() < 7*time.Millisecond).To(BeTrue())
	})

	It("Should sync the contents of all of the files on shutdown", func() {
		ctx, cancel := signal.TODO()
		fs, err := kfs.New[int]("testdata", kfs.WithExtensionConfig(".test"), kfs.WithFS(kfs.NewMem()))
		Expect(err).ToNot(HaveOccurred())
		defer Expect(fs.RemoveAll()).To(Succeed())
		_, err = fs.Acquire(1)
		Expect(err).To(BeNil())
		_, err = fs.Acquire(2)
		Expect(err).To(BeNil())
		_, err = fs.Acquire(3)
		Expect(err).To(BeNil())
		fs.Release(1)
		fs.Release(2)
		fs.Release(3)
		time.Sleep(5 * time.Millisecond)
		Expect(fs.OpenFiles()[1].Age() > 5*time.Millisecond).To(BeTrue())
		sync := &kfs.Sync[int]{
			FS:       fs,
			Interval: 5 * time.Millisecond,
			MaxAge:   2 * time.Millisecond,
		}
		errs := sync.Start(ctx)
		go func() {
			defer GinkgoRecover()
			Expect(<-errs).ToNot(HaveOccurred())
		}()
		time.Sleep(15 * time.Millisecond)
		cancel()
		Expect(errors.Is(ctx.Wait(), context.Canceled)).To(BeTrue())
		fOne := fs.OpenFiles()[1]
		Expect(fOne.Age() < 3*time.Millisecond).To(BeTrue())
	})

})
