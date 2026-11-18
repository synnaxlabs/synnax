// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package start_test

import (
	"io"
	"net"
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/synnaxlabs/synnax/cmd"
	. "github.com/synnaxlabs/x/testutil"
)

// coreEnv makes the test binary run as the Core instead of running the specs.
const coreEnv = "SYNNAX_START_TEST_CORE"

// TestMain runs the Core when coreEnv is set, so specs can start a real Core process
// from this binary without building one.
func TestMain(m *testing.M) {
	if os.Getenv(coreEnv) != "" {
		cmd.RunMain()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// launchCore launches a Core with in-memory storage and the given extra flags. It
// returns the Core's session, the writing end of its stdin, and its address.
func launchCore(flags ...string) (*gexec.Session, io.WriteCloser, string) {
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
	return session, stdin, addr
}

// startCore launches a Core as launchCore does and returns once the Core is running.
func startCore(flags ...string) (*gexec.Session, io.WriteCloser, string) {
	GinkgoHelper()
	session, stdin, addr := launchCore(flags...)
	Eventually(session.Out, 30*time.Second).Should(gbytes.Say("Synnax is running"))
	return session, stdin, addr
}

func TestStart(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Start Suite", SuiteConfig())
}
