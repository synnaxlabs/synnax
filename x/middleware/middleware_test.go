package middleware_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/middleware"
)

type request struct {
	value string
}

type response struct {
	value string
}

type myFirstMiddleware struct{}

func (m *myFirstMiddleware) Handle(
	ctx context.Context,
	req *request,
	next func() (*response, error),
) (*response, error) {
	req.value = "request"
	res, err := next()
	if err != nil {
		return res, err
	}
	res.value = "response"
	return res, nil
}

type myLastMiddleware struct{}

func (m *myLastMiddleware) Handle(
	ctx context.Context,
	req *request,
	next func() (*response, error),
) (*response, error) {
	return &response{value: "last"}, nil
}

var _ = Describe("ExecSequentially", func() {
	It("Should execute clientMiddleware in the correct order", func() {
		executor := &middleware.Executor[*request, *response]{
			Middleware: []middleware.Middleware[*request, *response]{
				&myFirstMiddleware{},
				&myFirstMiddleware{},
				&myLastMiddleware{},
			},
		}
		req := &request{}
		res, err := executor.Exec(context.TODO(), req)
		Expect(err).To(BeNil())
		Expect(res.value).To(Equal("response"))
		Expect(req.value).To(Equal("request"))
	})
	It("Should panic if the last middleware calls next", func() {
		req := &request{}
		Expect(func() {
			_, _ = middleware.ExecSequentially(
				context.TODO(),
				req,
				[]middleware.Middleware[*request, *response]{
					&myFirstMiddleware{},
					&myFirstMiddleware{},
				})
		}).To(Panic())
	})
})
