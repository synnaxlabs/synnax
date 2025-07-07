// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fmtls_test

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fmtls"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Middleware", func() {
	var (
		m         freighter.Middleware
		ctx       freighter.Context
		collector freighter.MiddlewareCollector
	)
	BeforeEach(func() {
		m = fmtls.GateMiddleware("foo")
		ctx = freighter.Context{Context: context.Background()}
		collector = freighter.MiddlewareCollector{}
		collector.Use(m)
	})
	Describe("GateMiddleware", func() {
		It("Should allow a certificate with a valid CN", func() {
			ctx.SecurityInfo.TLS.Used = true
			ctx.SecurityInfo.TLS.VerifiedChains = [][]*x509.Certificate{
				{
					{
						Subject: pkix.Name{
							CommonName: "foo",
						},
					},
				},
			}
			_ = MustSucceed(collector.Exec(ctx, freighter.NoopFinalizer))
		})
		It("Should return a SecurityError if no certificate is provided", func() {
			ctx.SecurityInfo.TLS.Used = true
			_, err := collector.Exec(ctx, freighter.NoopFinalizer)
			Expect(err).To(HaveOccurredAs(fmtls.ErrAuth))
		})
		It("Should return a SecurityError if the CN is not correct", func() {
			ctx.SecurityInfo.TLS.Used = true
			ctx.SecurityInfo.TLS.VerifiedChains = [][]*x509.Certificate{
				{
					{
						Subject: pkix.Name{
							CommonName: "bar",
						},
					},
				},
			}
			_, err := collector.Exec(ctx, freighter.NoopFinalizer)
			Expect(err).To(HaveOccurredAs(fmtls.ErrAuth))
		})
	})
})
