package persist_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/operation"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/kfs"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
)

type BasicOperation struct {
	executed bool
}

func (b *BasicOperation) Context() context.Context {
	return context.Background()
}

func (b *BasicOperation) FileKey() int {
	return 1
}

func (b *BasicOperation) Exec(f kfs.File[int]) {
	b.executed = true
	if _, err := f.Write([]byte("hello")); err != nil {
		panic(err)
	}
}

func (b *BasicOperation) WriteError(err error) {
	panic(err)
}

var _ = Describe("Persist", func() {
	var (
		p  *Persist[int, operation.Operation[int]]
		fs kfs.FS[int]
	)
	BeforeEach(func() {
		var err error
		fs, err = kfs.New[int]("testdata", kfs.WithFS(kfs.NewMem()))
		Expect(err).ToNot(HaveOccurred())
		p = MustSucceed(New[int, operation.Operation[int]](fs, Config{
			NumWorkers: 50,
		}))
	})
	Describe("QExec", func() {
		It("Should execute an operation correctly", func() {
			b := &BasicOperation{}
			ops := confluence.NewStream[[]operation.Operation[int]](1)
			ctx, cancel := signal.TODO()
			defer cancel()
			p.InFrom(ops)
			p.Flow(ctx)
			ops.Inlet() <- []operation.Operation[int]{b}
			ops.Close()
			Expect(ctx.Wait()).To(Succeed())
			// GoRead the file.
			f, err := fs.Acquire(1)
			Expect(err).ToNot(HaveOccurred())
			fs.Release(1)
			buf := make([]byte, 5)
			_, err = f.Seek(0, 0)
			Expect(err).ToNot(HaveOccurred())
			if _, err := f.Read(buf); err != nil {
				panic(err)
			}
			Expect(string(buf)).To(Equal("hello"))
		})
	})
	Describe("Shutdown", func() {
		It("Should execute all operations before shutting down", func() {
			b := &BasicOperation{}
			ops := confluence.NewStream[[]operation.Operation[int]](1)
			p.InFrom(ops)
			ctx, cancel := signal.TODO()
			defer cancel()
			p.Flow(ctx)
			ops.Inlet() <- []operation.Operation[int]{b}
			ops.Close()
			Expect(ctx.Wait()).To(Succeed())
			Expect(b.executed).To(BeTrue())
		})
	})
})
