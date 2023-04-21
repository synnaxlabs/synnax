// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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

var _ middleware.Middleware[*request, *response] = &myFirstMiddleware{}

func (m *myFirstMiddleware) Exec(
	req *request,
	next func(*request) (*response, error),
) (*response, error) {
	req.value = "request"
	return next(req)
}

type myFinalizer struct{}

var _ middleware.Finalizer[*request, *response] = &myFinalizer{}

func (m *myFinalizer) Finalize(req *request) (*response, error) {
	return nil, nil
}

var _ = Describe("ExecSequentially", func() {
	It("Should execute middleware in the correct order", func() {
		chain := &middleware.Chain[*request, *response]{
			&myFirstMiddleware{},
			&myFirstMiddleware{},
		}
		req := &request{}
		_, err := chain.Exec(req, &myFinalizer{})
		Expect(err).To(BeNil())
		Expect(req.value).To(Equal("request"))
	})
	It("Should not execute middleware if the context is canceled", func() {
		collector := &middleware.Collector[*request, *response]{}
		collector.Use(
			&myFirstMiddleware{},
			&myFirstMiddleware{},
		)
		req := &request{}
		_, err := collector.Exec(req, &myFinalizer{})
		Expect(err).To(HaveOccurredAs(context.Canceled))
		Expect(req.value).To(Equal(""))
	})
})
