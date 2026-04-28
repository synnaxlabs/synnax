// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/synnax/pkg/server"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/net"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("HTTP", func() {
	It("Should serve http requests", Focus, func(ctx context.Context) {
		r := MustSucceed(fhttp.NewRouter())
		s := fhttp.NewUnaryServer[int, int](r, "/basic")
		s.BindHandler(func(_ context.Context, req int) (int, error) {
			req++
			return req, nil
		})
		port := MustSucceed(net.FindOpenPort())
		addr := address.Newf("localhost:%d", port)
		b := MustSucceed(server.Serve(server.Config{
			ListenAddress: addr,
			Security:      server.SecurityConfig{Insecure: new(true)},
			Debug:         new(true),
			Branches: []server.Branch{
				&server.SecureHTTPBranch{Transports: []fhttp.BindableTransport{r}},
			},
		}))
		defer func() { Expect(b.Close()).To(Succeed()) }()
		url := "http://" + addr.String() + "/basic"
		body := MustSucceed(json.Marshal(1))
		req := MustSucceed(http.NewRequestWithContext(
			ctx, http.MethodPost, url, bytes.NewReader(body),
		))
		req.Header.Set(fiber.HeaderContentType, "application/json")
		res := MustSucceed((&http.Client{}).Do(req))
		defer func() { Expect(res.Body.Close()).To(Succeed()) }()
		Expect(res.StatusCode).To(Equal(http.StatusOK))
		respBody := MustSucceed(io.ReadAll(res.Body))
		var got int
		Expect(json.Unmarshal(respBody, &got)).To(Succeed())
		Expect(got).To(Equal(2))
	})
})
