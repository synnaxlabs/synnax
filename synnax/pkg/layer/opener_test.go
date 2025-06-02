package layer_test

import (
	"context"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"

	"github.com/synnaxlabs/synnax/pkg/layer"
	xio "github.com/synnaxlabs/x/io"
)

var _ = Describe("Opener", Ordered, func() {
	var (
		err              error
		closer           xio.MultiCloser
		cleanup          func()
		ok               func(c io.Closer) bool
		multiCloserCalls int
		cancel           context.CancelFunc
	)
	BeforeEach(func() {
		err = nil
		var cancelCtx context.Context
		cancelCtx, cancel = context.WithCancel(ctx)
		multiCloserCalls = 0
		closer = xio.MultiCloser{
			xio.CloserFunc(func() error {
				multiCloserCalls++
				return nil
			}),
		}
		cleanup, ok = layer.NewOpener(cancelCtx, &err, &closer)
	})
	It("Should correctly open a set of services that return without an error", func() {
		open := func(ctx context.Context) error {
			defer cleanup()
			if err = func() error {
				return nil
			}(); ok(nil) {
				return err
			}
			return nil
		}
		Expect(open(ctx)).To(Succeed())
		Expect(multiCloserCalls).To(Equal(0))
	})
	It("Should call the closer if an error occurs", func() {
		open := func(ctx context.Context) error {
			defer cleanup()
			if err = func() error {
				return errors.New("cat")
			}(); !ok(nil) {
				return err
			}
			return nil
		}
		Expect(open(ctx)).To(HaveOccurred())
		Expect(multiCloserCalls).To(Equal(1))
	})
	It("Should call the closer if the context is cancelled", func() {
		open := func(ctx context.Context) error {
			defer cleanup()
			cancel()
			return err
		}
		Expect(open(ctx)).To(Succeed())
		Expect(multiCloserCalls).To(Equal(1))
	})

	It("Should add a new closer to the list of closers", func() {
		secondaryCloserCalls := 0
		open := func(ctx context.Context) error {
			defer cleanup()
			if err = func() error {
				return nil
			}(); ok(xio.CloserFunc(func() error {
				secondaryCloserCalls++
				return nil
			})) {
				return err
			}
			return nil
		}
		Expect(open(ctx)).To(Succeed())
		Expect(closer).To(HaveLen(2))
		Expect(multiCloserCalls).To(Equal(0))
		Expect(secondaryCloserCalls).To(Equal(0))
	})

	It("Should call added closers if an error occurs", func() {
		secondaryCloserCalls := 0
		tertiaryCloserCalls := 0
		open := func(ctx context.Context) error {
			defer cleanup()
			if err = func() error {
				return nil
			}(); !ok(xio.CloserFunc(func() error {
				secondaryCloserCalls++
				return nil
			})) {
				return err
			}
			if err = func() error {
				return errors.New("cat")
			}(); !ok(xio.CloserFunc(func() error {
				tertiaryCloserCalls++
				return nil
			})) {
				return err
			}
			return nil
		}
		Expect(open(ctx)).To(HaveOccurred())
		Expect(multiCloserCalls).To(Equal(1))
		Expect(secondaryCloserCalls).To(Equal(1))
		Expect(tertiaryCloserCalls).To(Equal(0))
	})
})
