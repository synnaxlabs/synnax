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
	"io"
	"net"
	"os"
	"os/exec"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Start", func() {
	// startCore starts a Core with in-memory storage and the given extra flags. It
	// returns once the Core is running, with the writing end of the Core's stdin.
	startCore := func(flags ...string) (*gexec.Session, io.WriteCloser) {
		GinkgoHelper()
		l := MustSucceed(net.Listen("tcp", "127.0.0.1:0"))
		addr := l.Addr().String()
		Expect(l.Close()).To(Succeed())
		dir := GinkgoT().TempDir()
		cmd := exec.Command(
			MustSucceed(os.Executable()),
			append([]string{
				"start", "--mem", "--insecure", "--no-driver",
				"--listen", addr, "--data", dir,
			}, flags...)...,
		)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), coreEnv+"=1")
		stdin := MustSucceed(cmd.StdinPipe())
		session := MustSucceed(gexec.Start(cmd, GinkgoWriter, GinkgoWriter))
		DeferCleanup(func() { session.Kill().Wait() })
		Eventually(session.Out, 30*time.Second).Should(gbytes.Say("Synnax is running"))
		return session, stdin
	}

	DescribeTable("Should shut down cleanly on a stop signal",
		func(sig syscall.Signal) {
			session, _ := startCore()
			session.Signal(sig)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Synnax has shut down"))
		},
		Entry("SIGINT", syscall.SIGINT),
		Entry("SIGTERM", syscall.SIGTERM),
	)

	It("Should exit at once when a second signal arrives during shutdown", func() {
		session, _ := startCore()
		// Distinct signals, so the kernel cannot merge them into one.
		session.Signal(syscall.SIGTERM)
		session.Signal(syscall.SIGINT)
		Eventually(session, 30*time.Second).Should(gexec.Exit(1))
		Expect(session.Out).To(gbytes.Say("second stop signal"))
	})

	It(
		"Should shut down cleanly on the stop keyword followed by a closed stdin",
		func() {
			session, stdin := startCore("--stop-on-stdin-close")
			MustSucceed(io.WriteString(stdin, "stop\n"))
			Expect(stdin.Close()).To(Succeed())
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Synnax has shut down"))
		},
	)

	It("Should shut down cleanly on one signal after the stop keyword", func() {
		session, stdin := startCore()
		MustSucceed(io.WriteString(stdin, "stop\n"))
		Eventually(session.Out).Should(gbytes.Say("Synnax is shutting down"))
		session.Signal(syscall.SIGTERM)
		Eventually(session, 30*time.Second).Should(gexec.Exit(0))
		Expect(session.Out).To(gbytes.Say("Synnax has shut down"))
	})

	It("Should exit at once on two signals after the stop keyword", func() {
		session, stdin := startCore()
		MustSucceed(io.WriteString(stdin, "stop\n"))
		session.Signal(syscall.SIGTERM)
		session.Signal(syscall.SIGINT)
		Eventually(session, 30*time.Second).Should(gexec.Exit(1))
		Expect(session.Out).To(gbytes.Say("second stop signal"))
	})
})
