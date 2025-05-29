// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/synnax/pkg/server"
	"github.com/synnaxlabs/x/config"
	. "github.com/synnaxlabs/x/testutil"
	"net/http"
	"sync"
)

type integerServer struct {
}

func (b integerServer) BindTo(router *fhttp.Router) {
	g := fhttp.UnaryServer[int, int](router, false, "/basic")
	g.BindHandler(func(ctx context.Context, req int) (int, error) {
		req++
		return req, nil
	})
}

var _ = Describe("HTTP", func() {
	It("Should serve http requests", func() {
		r := fhttp.NewRouter()
		integerServer{}.BindTo(r)
		b := MustSucceed(server.New(server.Config{
			ListenAddress: "localhost:26260",
			Security: server.SecurityConfig{
				Insecure: config.Bool(true),
			},
			Debug: config.Bool(true),
			Branches: []server.Branch{
				&server.SecureHTTPBranch{
					Transports: []fhttp.BindableTransport{r},
				},
			},
		}))
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			Expect(b.Serve()).To(Succeed())
			wg.Done()
		}()
		_, err := http.Get("http://localhost:26260/basic")
		Expect(err).To(Succeed())
		b.Stop()
		wg.Wait()
	})
})
