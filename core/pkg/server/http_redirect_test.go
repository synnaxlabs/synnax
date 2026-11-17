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
					{Address: "synnax.test:0", TLS: prov.TLSConfigFor(src)},
				},
				Security: server.SecurityConfig{Insecure: new(false)},
				Branches: []server.Branch{server.NewHTTPRedirectBranch()},
			}))
		})
		const (
			loopback = 0
			named    = 1
		)
		redirect := func(listener int, host string) string {
			GinkgoHelper()
			// These Listeners bind all interfaces, so loopback reaches them.
			addr := b.Addresses()[listener]
			conn := MustOpen(net.Dial("tcp", "127.0.0.1"+addr.PortString()))
			MustSucceed(fmt.Fprintf(
				conn, "GET /a/b?c=d HTTP/1.1\r\nHost: %s\r\n\r\n", host,
			))
			res := MustSucceed(http.ReadResponse(bufio.NewReader(conn), nil))
			Expect(res.Body.Close()).To(Succeed())
			Expect(res.StatusCode).To(Equal(http.StatusMovedPermanently))
			return res.Header.Get("Location")
		}
		target := func(listener int, host string) string {
			return "https://" + host + b.Addresses()[listener].PortString() + "/a/b?c=d"
		}
		DescribeTable(
			"Should redirect to the configured host of a named Listener when the "+
				"certificate does not cover the request host",
			func(host string) {
				Expect(redirect(named, host)).To(Equal(target(named, "synnax.test")))
			},
			Entry("forged host", "evil.example"),
			Entry("forged host and port", "evil.example:9443"),
		)
		DescribeTable(
			"Should keep the request host and port when the certificate covers it",
			func(listener int) {
				Expect(redirect(listener, "localhost")).
					To(Equal(target(listener, "localhost")))
				Expect(redirect(listener, "localhost:9443")).
					To(Equal("https://localhost:9443/a/b?c=d"))
			},
			Entry("loopback Listener", loopback),
			Entry("named Listener", named),
		)
		It("Should keep the request host when the Listener host is loopback", func() {
			Expect(redirect(loopback, "192.168.1.5")).
				To(Equal(target(loopback, "192.168.1.5")))
			Expect(redirect(loopback, "192.168.1.5:9090")).
				To(Equal("https://192.168.1.5:9090/a/b?c=d"))
		})
		It("Should redirect an absolute-form request to its path and query", func() {
			addr := "127.0.0.1" + b.Addresses()[named].PortString()
			conn := MustOpen(net.Dial("tcp", addr))
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
