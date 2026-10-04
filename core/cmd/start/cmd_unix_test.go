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
	// startCore starts a Core with in-memory storage and returns once it is running.
	startCore := func() *gexec.Session {
		GinkgoHelper()
		l := MustSucceed(net.Listen("tcp", "127.0.0.1:0"))
		addr := l.Addr().String()
		Expect(l.Close()).To(Succeed())
		dir := GinkgoT().TempDir()
		cmd := exec.Command(
			MustSucceed(os.Executable()),
			"start", "--mem", "--insecure", "--no-driver",
			"--listen", addr, "--data", dir,
		)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), coreEnv+"=1")
		session := MustSucceed(gexec.Start(cmd, GinkgoWriter, GinkgoWriter))
		DeferCleanup(func() { session.Kill().Wait() })
		Eventually(session.Out, 30*time.Second).Should(gbytes.Say("Synnax is running"))
		return session
	}

	DescribeTable("Should shut down cleanly on a stop signal",
		func(sig syscall.Signal) {
			session := startCore()
			session.Signal(sig)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Synnax has shut down"))
		},
		Entry("SIGINT", syscall.SIGINT),
		Entry("SIGTERM", syscall.SIGTERM),
	)

	It("Should shut down cleanly when a second signal arrives during shutdown", func() {
		session := startCore()
		session.Signal(syscall.SIGTERM)
		Eventually(session.Out).Should(gbytes.Say("Synnax is shutting down"))
		session.Signal(syscall.SIGINT)
		Eventually(session, 30*time.Second).Should(gexec.Exit(0))
		Expect(session.Out).To(gbytes.Say("Synnax has shut down"))
	})
})
