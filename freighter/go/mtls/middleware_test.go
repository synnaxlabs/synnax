// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mtls_test

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/mtls"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Middleware", func() {
	var (
		m         freighter.Middleware
		md        freighter.MD
		collector freighter.MiddlewareCollector
	)
	BeforeEach(func() {
		m = mtls.GateMiddleware("foo")
		md = freighter.MD{}
		collector = freighter.MiddlewareCollector{}
		collector.Use(m)
	})
	Describe("GateMiddleware", func() {
		It("Should allow a certificate with a valid CN", func() {
			md.Sec.TLS.Used = true
			md.Sec.TLS.VerifiedChains = [][]*x509.Certificate{
				{
					{
						Subject: pkix.Name{
							CommonName: "foo",
						},
					},
				},
			}
			_ = MustSucceed(collector.Exec(context.TODO(), md, freighter.NopFinalizer))
		})
		It("Should return a SecurityError if no certificate is provided", func() {
			md.Sec.TLS.Used = true
			_, err := collector.Exec(context.TODO(), md, freighter.NopFinalizer)
			Expect(err).To(HaveOccurredAs(mtls.AuthError))
		})
		It("Should return a SecurityError if the CN is not correct", func() {
			md.Sec.TLS.Used = true
			md.Sec.TLS.VerifiedChains = [][]*x509.Certificate{
				{
					{
						Subject: pkix.Name{
							CommonName: "bar",
						},
					},
				},
			}
			_, err := collector.Exec(context.TODO(), md, freighter.NopFinalizer)
			Expect(err).To(HaveOccurredAs(mtls.AuthError))
		})
	})
})
