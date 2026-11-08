// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build linux

package update_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	servicetest "github.com/synnaxlabs/synnax/cmd/service/testutil"
	"github.com/synnaxlabs/synnax/cmd/update"
	. "github.com/synnaxlabs/x/testutil"
)

// No spec reaches a Core service on the computer that runs it.
var _ = BeforeEach(func() { servicetest.NewSystemctl() })

// writeCore writes a fake Core binary of the given version to path.
func writeCore(path, version string) {
	GinkgoHelper()
	Expect(os.WriteFile(
		path,
		fmt.Appendf(nil, "#!/bin/sh\necho 'Synnax %s (commit: abc1234)'\n", version),
		0o755,
	)).To(Succeed())
}

func coreAt(path string) string {
	GinkgoHelper()
	return string(MustSucceed(os.ReadFile(path)))
}

var _ = Describe("Update with a Core service", func() {
	var (
		systemctl          *servicetest.Systemctl
		data, binary, next string
		out                *bytes.Buffer
	)
	BeforeEach(func() {
		out = new(bytes.Buffer)
		systemctl = servicetest.NewSystemctl()
		dir := GinkgoT().TempDir()
		data = filepath.Join(dir, "data")
		binary = filepath.Join(dir, "synnax")
		next = filepath.Join(dir, "synnax-next")
		writeCore(binary, "0.60.0")
		writeCore(next, "0.61.0")
		Expect(os.MkdirAll(data, 0o700)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(data, "VERSION"), []byte("0.60.0"), 0o600)).
			To(Succeed())
		systemctl.Install(dir, binary, "start", "--data="+data)
	})

	run := func(args ...string) error {
		cmd := update.NewCmd()
		cmd.SetOut(out)
		cmd.SetErr(io.Discard)
		cmd.SetArgs(args)
		return cmd.Execute()
	}

	Describe("From a file", func() {
		It("Should replace the Core and wait for the commit", func() {
			systemctl.OnStart(fmt.Sprintf(
				"printf 0.61.0 > %q",
				filepath.Join(data, "VERSION"),
			))
			Expect(run("--file", next)).To(Succeed())
			Expect(out.String()).
				To(HaveSuffix("Updated to Core 0.61.0.\n"))
			Expect(systemctl.Calls()).To(Equal([]string{"stop synnax", "start synnax"}))
			Expect(coreAt(binary)).To(ContainSubstring("0.61.0"))
			Expect(coreAt(binary + ".old")).To(ContainSubstring("0.60.0"))
			Expect(binary + ".new").ToNot(BeAnExistingFile())
		})

		It("Should start the old Core when the new one undoes the update", func() {
			systemctl.OnStart(fmt.Sprintf(
				"printf 0.61.0 > %q",
				filepath.Join(data, "UPDATE_FAILED"),
			))
			Expect(run("--file", next)).To(MatchError(
				"Update to Core 0.61.0 failed. Core 0.60.0 runs again. Find the cause " +
					"in the Core log, fix it, and run `synnax update retry` before you " +
					"update again.",
			))
			Expect(systemctl.Calls()).To(Equal([]string{
				"stop synnax",
				"start synnax",
				"stop synnax",
				"start synnax",
			}))
			Expect(coreAt(binary)).To(ContainSubstring("0.60.0"))
		})

		It("Should not update to a version that failed before", func() {
			Expect(os.WriteFile(
				filepath.Join(data, "UPDATE_FAILED"),
				[]byte("0.61.0"),
				0o600,
			)).To(Succeed())
			Expect(run("--file", next)).To(MatchError(
				"The update to Core 0.61.0 failed before. Fix the cause, run `synnax " +
					"update retry`, then update again.",
			))
			Expect(systemctl.Calls()).To(BeEmpty())
			Expect(coreAt(binary)).To(ContainSubstring("0.60.0"))
			Expect(binary + ".new").ToNot(BeAnExistingFile())
		})

		It("Should not update to an older Core", func() {
			writeCore(next, "0.59.0")
			Expect(run("--file", next)).To(MatchError(
				"Core 0.59.0 is older than the installed Core 0.60.0. To go back, run " +
					"`synnax update undo`.",
			))
			Expect(systemctl.Calls()).To(BeEmpty())
		})

		It("Should not update to a file that is not a Core", func() {
			Expect(os.WriteFile(next, []byte("#!/bin/sh\necho hello\n"), 0o644)).
				To(Succeed())
			Expect(run("--file", next)).To(MatchError(ContainSubstring(
				next + " is not a Core binary for this computer",
			)))
			Expect(systemctl.Calls()).To(BeEmpty())
		})

		It("Should stop waiting at the timeout", func() {
			Expect(run("--file", next, "--timeout", "500ms")).To(MatchError(
				"Core 0.61.0 did not finish the update within 500ms. Check the Core log.",
			))
			Expect(coreAt(binary)).To(ContainSubstring("0.61.0"))
		})
	})

	It("Should use the data directory of the service without --data", func() {
		Expect(os.WriteFile(
			filepath.Join(data, "UPDATE_FAILED"),
			[]byte("0.61.0"),
			0o600,
		)).To(Succeed())
		Expect(run(
			"retry",
			"--log-file-path", filepath.Join(data, "logs", "synnax.log"),
		)).To(Succeed())
		Expect(out.String()).
			To(Equal("Core 0.61.0 updates the data again at its next start.\n"))
	})

	Describe("Undo", func() {
		BeforeEach(func() {
			writeCore(binary+".old", "0.60.0")
			writeCore(binary, "0.61.0")
			Expect(os.WriteFile(
				filepath.Join(data, "VERSION"),
				[]byte("0.61.0"),
				0o600,
			)).To(Succeed())
		})

		It("Should put back the binary that the update replaced", func() {
			createBackup(data, "0000000001-0.60.0")
			Expect(run(
				"undo",
				"--log-file-path", filepath.Join(data, "logs", "synnax.log"),
			)).To(Succeed())
			Expect(out.String()).To(HaveSuffix(
				"Put back Core 0.60.0 at " + binary + ". Start the Core.\n",
			))
			Expect(coreAt(binary)).To(ContainSubstring("0.60.0"))
			Expect(coreAt(binary + ".old")).To(ContainSubstring("0.61.0"))
		})

		It("Should keep a replaced binary of another version", func() {
			createBackup(data, "0000000001-0.59.0")
			Expect(run(
				"undo",
				"--data", data,
				"--log-file-path", filepath.Join(data, "logs", "synnax.log"),
			)).To(Succeed())
			Expect(out.String()).
				To(HaveSuffix("Run Core 0.59.0 on the data.\n"))
			Expect(coreAt(binary)).To(ContainSubstring("0.61.0"))
		})

		It("Should keep the service binary when it undoes other data", func() {
			other := GinkgoT().TempDir()
			createBackup(other, "0000000001-0.60.0")
			Expect(run(
				"undo",
				"--data", other,
				"--log-file-path", filepath.Join(other, "logs", "synnax.log"),
			)).To(Succeed())
			Expect(out.String()).
				To(HaveSuffix("Run Core 0.60.0 on the data.\n"))
			Expect(coreAt(binary)).To(ContainSubstring("0.61.0"))
		})
	})
})
