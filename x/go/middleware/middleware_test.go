// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package middleware_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/middleware"
)

type request struct {
	value string
}

type response struct{}

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

var _ = Describe("Middleware", func() {
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
})
