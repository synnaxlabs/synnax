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
	"crypto/tls"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/mock"
	"github.com/synnaxlabs/synnax/pkg/server"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("HttpRedirect", func() {
	It("Should redirect http requests to https", func() {
		fs := xfs.NewMem()
		mock.GenerateCerts(fs)
		prov := MustSucceed(security.NewProvider(security.ProviderConfig{
			LoaderConfig: cert.LoaderConfig{FS: fs},
			KeySize:      mock.SmallKeySize,
			Insecure:     new(false),
		}))
		received := false
		b := MustSucceed(server.Serve(server.Config{
			ListenAddress: "localhost:26260",
			Security: server.SecurityConfig{
				Insecure: new(false),
				TLS:      prov.TLS(),
			},
			Branches: []server.Branch{
				server.NewHTTPRedirectBranch(),
				server.NewSimpleHTTPBranch(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					received = true
					w.WriteHeader(http.StatusOK)
				}), server.RoutingPolicyServeAlwaysPreferSecure),
			},
		}))

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}
		resp, err := client.Get("http://localhost:26260")
		Expect(err).To(Succeed())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(received).To(BeTrue())
		Expect(b.Close()).To(Succeed())
	})

})
