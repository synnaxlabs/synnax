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
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/cert/file"
	"github.com/synnaxlabs/synnax/pkg/security/testutil"
	"github.com/synnaxlabs/synnax/pkg/server"
	"github.com/synnaxlabs/x/address"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("HttpRedirect", func() {
	It("Should redirect HTTP requests to HTTPS", func() {
		fs := xfs.NewMem()
		testutil.GenerateCerts(fs)
		prov := MustSucceed(security.NewProvider(security.ProviderConfig{
			FS:       fs,
			Insecure: new(false),
		}))
		src := MustSucceed(file.NewSource(fs,
			"/usr/local/synnax/certs/node.crt",
			"/usr/local/synnax/certs/node.key",
		))
		received := false
		b := MustSucceed(server.Serve(server.Config{
			Listeners: []server.Listener{{
				Address: "localhost:0",
				TLS:     prov.TLSConfigFor(src),
			}},
			Security: server.SecurityConfig{Insecure: new(false)},
			Branches: []server.Branch{
				server.NewHTTPRedirectBranch(),
				server.NewSimpleHTTPBranch(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						received = true
						w.WriteHeader(http.StatusOK)
					}),
					server.RoutingPolicyServeAlwaysPreferSecure,
				),
			},
		}))

		l := MustSucceed(cert.NewLoader(cert.LoaderConfig{FS: fs}))
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: certPool(MustSucceed(l.TrustAnchorsPEM())),
			},
		}
		client := &http.Client{Transport: tr}
		resp := MustSucceed(client.Get("http://" + b.Addresses()[0].String()))
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(received).To(BeTrue())
		Expect(b.Close()).To(Succeed())
	})

	Describe("Redirect target", func() {
		var b *server.Server
		BeforeEach(func() {
			fs := xfs.NewMem()
			testutil.GenerateCerts(fs)
			prov := MustSucceed(security.NewProvider(security.ProviderConfig{
				FS:       fs,
				Insecure: new(false),
			}))
			src := MustSucceed(file.NewSource(fs,
				"/usr/local/synnax/certs/node.crt",
				"/usr/local/synnax/certs/node.key",
			))
			b = MustOpen(server.Serve(server.Config{
				Listeners: []server.Listener{
					{Address: "localhost:0", TLS: prov.TLSConfigFor(src)},
					{Address: "127.0.0.1:0", TLS: prov.TLSConfigFor(src)},
				},
				Security: server.SecurityConfig{Insecure: new(false)},
				Branches: []server.Branch{server.NewHTTPRedirectBranch()},
			}))
		})
		listenerTarget := func(addr address.Address) string {
			return "https://" + addr.String() + "/a/b?c=d"
		}
		localhostTarget := func(addr address.Address) string {
			return "https://localhost:" + addr.PortString()[1:] + "/a/b?c=d"
		}
		requestTarget := func(address.Address) string {
			return "https://localhost:9443/a/b?c=d"
		}
		DescribeTable(
			"Should keep the request host and port when the certificate covers the host",
			func(listener int, host string, target func(address.Address) string) {
				addr := b.Addresses()[listener]
				conn := MustOpen(net.Dial("tcp", addr.String()))
				MustSucceed(fmt.Fprintf(
					conn, "GET /a/b?c=d HTTP/1.1\r\nHost: %s\r\n\r\n", host,
				))
				res := MustSucceed(http.ReadResponse(bufio.NewReader(conn), nil))
				Expect(res.Body.Close()).To(Succeed())
				Expect(res.StatusCode).To(Equal(http.StatusMovedPermanently))
				Expect(res.Header.Get("Location")).To(Equal(target(addr)))
			},
			Entry("forged host, first listener", 0, "evil.example", listenerTarget),
			Entry("forged host, second listener", 1, "evil.example", listenerTarget),
			Entry(
				"forged host and port, first listener",
				0,
				"evil.example:9443",
				listenerTarget,
			),
			Entry(
				"forged host and port, second listener",
				1,
				"evil.example:9443",
				listenerTarget,
			),
			Entry("covered host, first listener", 0, "localhost", localhostTarget),
			Entry("covered host, second listener", 1, "localhost", localhostTarget),
			Entry(
				"covered host and port, first listener",
				0,
				"localhost:9443",
				requestTarget,
			),
			Entry(
				"covered host and port, second listener",
				1,
				"localhost:9443",
				requestTarget,
			),
		)
		It("Should redirect an absolute-form request to its path and query", func() {
			conn := MustOpen(net.Dial("tcp", b.Addresses()[0].String()))
			MustSucceed(fmt.Fprint(
				conn,
				"GET http://localhost:9443/a/b?c=d HTTP/1.1\r\n"+
					"Host: localhost:9443\r\n\r\n",
			))
			res := MustSucceed(http.ReadResponse(bufio.NewReader(conn), nil))
			Expect(res.Body.Close()).To(Succeed())
			Expect(res.StatusCode).To(Equal(http.StatusMovedPermanently))
			Expect(res.Header.Get("Location")).
				To(Equal("https://localhost:9443/a/b?c=d"))
		})
	})
})
