// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Discover", func() {
	Describe("Handshake", func() {
		noop := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

		// handshake dials server and runs a handshake with cfg on the connection.
		handshake := func(
			ctx context.Context,
			server *httptest.Server,
			cfg *tls.Config,
		) task.TestStep {
			GinkgoHelper()
			addr := server.Listener.Addr().String()
			conn := MustSucceed(net.Dial("tcp", addr))
			DeferCleanup(conn.Close)
			return driver.Handshake(ctx, conn, addr, cfg)
		}

		// trusting returns a TLS config that trusts the certificate of server.
		trusting := func(server *httptest.Server) *tls.Config {
			roots := x509.NewCertPool()
			roots.AddCert(server.Certificate())
			return &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots}
		}

		var server *httptest.Server

		BeforeEach(func() {
			server = httptest.NewTLSServer(noop)
			DeferCleanup(server.Close)
		})

		It("Should secure a connection to a server that the roots trust",
			func(ctx SpecContext) {
				step := handshake(ctx, server, trusting(server))
				Expect(step.Stage).To(Equal(driver.StageTLS))
				Expect(step.Address).To(Equal(server.Listener.Addr().String()))
				Expect(step.Variant).To(Equal(status.VariantSuccess))
				Expect(step.Message).To(Equal("Secured connection"))
			},
		)

		It("Should not trust a certificate that the roots do not sign",
			func(ctx SpecContext) {
				step := handshake(ctx, server, nil)
				Expect(step.Variant).To(Equal(status.VariantError))
				Expect(step.Message).To(Equal("Certificate not trusted"))
			},
		)

		It("Should reject a certificate that does not name the host",
			func(ctx SpecContext) {
				cfg := trusting(server)
				cfg.ServerName = "other.test"
				step := handshake(ctx, server, cfg)
				Expect(step.Variant).To(Equal(status.VariantError))
				Expect(step.Message).To(Equal("Certificate does not match the address"))
			},
		)

		It("Should report a server that does not use TLS", func(ctx SpecContext) {
			plain := httptest.NewServer(noop)
			DeferCleanup(plain.Close)
			step := handshake(ctx, plain, nil)
			Expect(step.Variant).To(Equal(status.VariantError))
			Expect(step.Message).To(Equal("Server does not use TLS"))
		})
	})

	Describe("AnswerTest", func() {
		step := func(stage string, variant status.Variant, message string) task.TestStep {
			return task.TestStep{
				Stage:       stage,
				Variant:     variant,
				Message:     message,
				Description: message + ".",
			}
		}

		It("Should answer with no error when every step passed", func() {
			data := MustSucceed(driver.AnswerTest([]task.TestStep{
				step("dial", status.VariantSuccess, "Reached broker"),
				step("broker", status.VariantWarning, "Slow broker"),
			}))
			Expect(data).To(HaveKey("steps"))
		})

		It("Should answer with the first step that failed", func() {
			data, err := driver.AnswerTest([]task.TestStep{
				step("dial", status.VariantSuccess, "Reached broker"),
				step("broker", status.VariantError, "Failed to reach broker 1"),
				step("broker", status.VariantSuccess, "Reached broker 2"),
			})
			Expect(err).To(MatchError(
				"Failed to reach broker 1: Failed to reach broker 1.",
			))
			Expect(data["steps"]).To(HaveLen(3))
		})
	})

	DescribeTable("FormatOf",
		func(payload []byte, format driver.Format) {
			Expect(driver.FormatOf(payload)).To(Equal(format))
		},
		Entry("a JSON object", []byte(`{"v":1}`), driver.FormatJSON),
		Entry("a JSON number", []byte("12.5"), driver.FormatJSON),
		Entry("text that is not JSON", []byte("ON"), driver.FormatText),
		Entry("bytes that are not text", []byte{0xff, 0xfe, 0x01}, driver.FormatBinary),
	)

	DescribeTable(
		"Display",
		func(payload []byte, format driver.Format, limit int, shown string) {
			Expect(driver.Display(payload, format, limit)).To(Equal(shown))
		},
		Entry("JSON as text", []byte(`{"v":1}`), driver.FormatJSON, 16, `{"v":1}`),
		Entry("the start of text", []byte("abcdef"), driver.FormatText, 4, "abcd"),
		Entry("text cut at a rune boundary", []byte("ééé"), driver.FormatText, 5, "éé"),
		Entry(
			"binary as hex",
			[]byte{0xff, 0xfe, 0x01},
			driver.FormatBinary,
			16,
			"fffe01",
		),
		Entry(
			"the start of binary",
			[]byte{0xff, 0xfe, 0x01},
			driver.FormatBinary,
			2,
			"fffe",
		),
	)
})
