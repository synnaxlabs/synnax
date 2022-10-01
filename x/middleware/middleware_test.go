package middleware_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/middleware"
	. "github.com/synnaxlabs/x/testutil"
)

type request struct {
	value string
}

type response struct {
	value string
}

type myFirstMiddleware struct{}

func (m *myFirstMiddleware) Exec(
	ctx context.Context,
	req *request,
	next func(context.Context, *request) error,
) error {
	req.value = "request"
	err := next(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

type myFinalizer struct{}

func (m *myFinalizer) Finalize(ctx context.Context, req *request) error {
	return nil
}

var _ = Describe("ExecSequentially", func() {
	It("Should execute middleware in the correct order", func() {
		chain := &middleware.Chain[*request]{
			&myFirstMiddleware{},
			&myFirstMiddleware{},
		}
		req := &request{}
		err := chain.Exec(context.TODO(), req, &myFinalizer{})
		Expect(err).To(BeNil())
		Expect(req.value).To(Equal("request"))
	})
	It("Should not execute middleware if the context is canceled", func() {
		collector := &middleware.Collector[*request]{}
		collector.Use(
			&myFirstMiddleware{},
			&myFirstMiddleware{},
		)
		req := &request{}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := collector.Exec(ctx, req, &myFinalizer{})
		Expect(err).To(HaveOccurredAs(context.Canceled))
		Expect(req.value).To(Equal(""))
	})
})
