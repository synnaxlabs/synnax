// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build linux

// Package testutil provides a fake systemctl for the specs of the Core service.
package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/testutil"
)

const script = `#!/bin/sh
state=%q
case "$1" in
is-system-running) if [ -f "$state/offline" ]; then echo offline; exit 1; fi
   echo running ;;
show) if [ -f "$state/show" ]; then cat "$state/show"; else echo LoadState=not-found; fi ;;
*) echo "$*" >> "$state/calls"
   if [ "$1" = start ] && [ -f "$state/on-start" ]; then
     mv "$state/on-start" "$state/started"
     sh "$state/started"
   fi ;;
esac
`

// busctl prints the ExecStart property of the service that Install wrote.
const busctl = `#!/bin/sh
cat %q
`

// Systemctl is a fake systemctl first on the PATH of the test process.
type Systemctl struct{ dir string }

// NewSystemctl puts a fake systemctl and busctl first on the PATH until the spec ends.
// The fake reports no service until Install.
func NewSystemctl() *Systemctl {
	ginkgo.GinkgoHelper()
	dir := ginkgo.GinkgoT().TempDir()
	gomega.Expect(os.WriteFile(
		filepath.Join(dir, "systemctl"),
		fmt.Appendf(nil, script, dir),
		0o755,
	)).To(gomega.Succeed())
	gomega.Expect(os.WriteFile(
		filepath.Join(dir, "busctl"),
		fmt.Appendf(nil, busctl, filepath.Join(dir, "exec-start")),
		0o755,
	)).To(gomega.Succeed())
	ginkgo.GinkgoT().Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return &Systemctl{dir: dir}
}

// Install makes the fake report a service that runs `<binary> <args>`.
func (s *Systemctl) Install(workDir, binary string, args ...string) {
	ginkgo.GinkgoHelper()
	gomega.Expect(os.WriteFile(
		filepath.Join(s.dir, "show"),
		fmt.Appendf(nil, "LoadState=loaded\nWorkingDirectory=%s\n", workDir),
		0o644,
	)).To(gomega.Succeed())
	argv := append([]string{binary}, args...)
	property := fmt.Appendf(nil, "a(sasbttttuii) 1 %q %d", binary, len(argv))
	for _, arg := range argv {
		property = fmt.Appendf(property, " %q", arg)
	}
	gomega.Expect(os.WriteFile(
		filepath.Join(s.dir, "exec-start"),
		append(property, " false 0 0 0 0 0 0 0\n"...),
		0o644,
	)).To(gomega.Succeed())
}

// Offline makes the fake report that systemd does not run on the host.
func (s *Systemctl) Offline() {
	ginkgo.GinkgoHelper()
	gomega.Expect(os.WriteFile(filepath.Join(s.dir, "offline"), nil, 0o644)).
		To(gomega.Succeed())
}

// OnStart runs the shell script at the next start of the service only.
func (s *Systemctl) OnStart(script string) {
	ginkgo.GinkgoHelper()
	gomega.Expect(os.WriteFile(filepath.Join(s.dir, "on-start"), []byte(script), 0o644)).
		To(gomega.Succeed())
}

// Calls returns the commands other than show that the fake received.
func (s *Systemctl) Calls() []string {
	ginkgo.GinkgoHelper()
	data, err := os.ReadFile(filepath.Join(s.dir, "calls"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return strings.Split(
		strings.TrimSpace(string(testutil.MustSucceed(data, err))),
		"\n",
	)
}
