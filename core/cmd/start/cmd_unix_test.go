// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !windows

package start_test

import (
	"bufio"
	"io"
	"net"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Start", func() {
	// holdShutdown opens a request to the Core at addr and leaves its body unsent. The
	// Core waits on the request when it shuts down, until its 5s read timeout from the
	// request ends it, so stop signals sent in that window reach a shutting-down Core.
	holdShutdown := func(addr string) {
		GinkgoHelper()
		conn := MustSucceed(net.Dial("tcp", addr))
		DeferCleanup(conn.Close)
		MustSucceed(io.WriteString(
			conn,
			"POST / HTTP/1.1\r\nHost: core\r\nContent-Length: 1\r\n"+
				"Expect: 100-continue\r\n\r\n",
		))
		Expect(conn.SetReadDeadline(time.Now().Add(5 * time.Second))).To(Succeed())
		Expect(bufio.NewReader(conn).ReadString('\n')).
			To(Equal("HTTP/1.1 100 Continue\r\n"))
	}

	DescribeTable("Should shut down cleanly on a stop signal",
		func(sig syscall.Signal) {
			session, _, _ := startCore()
			session.Signal(sig)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Synnax has shut down"))
		},
		Entry("SIGINT", syscall.SIGINT),
		Entry("SIGTERM", syscall.SIGTERM),
	)

	It("Should exit at once when a second signal arrives during shutdown", func() {
		session, _, addr := startCore()
		holdShutdown(addr)
		// Distinct signals, so the kernel cannot merge them into one.
		session.Signal(syscall.SIGTERM)
		session.Signal(syscall.SIGINT)
		Eventually(session, 30*time.Second).Should(gexec.Exit(1))
		Expect(session.Out).To(gbytes.Say("second stop signal"))
	})

	It(
		"Should shut down cleanly on the stop keyword followed by a closed stdin",
		func() {
			session, stdin, _ := startCore("--stop-on-stdin-close")
			MustSucceed(io.WriteString(stdin, "stop\n"))
			Expect(stdin.Close()).To(Succeed())
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Synnax has shut down"))
		},
	)

	It("Should shut down cleanly on one signal after the stop keyword", func() {
		session, stdin, _ := startCore()
		MustSucceed(io.WriteString(stdin, "stop\n"))
		Eventually(session.Out).Should(gbytes.Say("Synnax is shutting down"))
		session.Signal(syscall.SIGTERM)
		Eventually(session, 30*time.Second).Should(gexec.Exit(0))
		Expect(session.Out).To(gbytes.Say("Synnax has shut down"))
	})

	It("Should exit at once on two signals after the stop keyword", func() {
		session, stdin, addr := startCore()
		holdShutdown(addr)
		MustSucceed(io.WriteString(stdin, "stop\n"))
		session.Signal(syscall.SIGTERM)
		session.Signal(syscall.SIGINT)
		Eventually(session, 30*time.Second).Should(gexec.Exit(1))
		Expect(session.Out).To(gbytes.Say("second stop signal"))
	})
})
