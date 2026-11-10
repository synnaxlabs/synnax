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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	servicetest "github.com/synnaxlabs/synnax/cmd/service/testutil"
	"github.com/synnaxlabs/synnax/cmd/update"
	"github.com/synnaxlabs/synnax/pkg/release"
	releasetest "github.com/synnaxlabs/synnax/pkg/release/testutil"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// No spec reaches a Core service on the computer that runs it.
var _ = BeforeEach(func() { servicetest.NewSystemctl() })

// fakeCore returns a fake Core binary of the given version.
func fakeCore(version string) []byte {
	return fmt.Appendf(nil, "#!/bin/sh\necho 'Synnax %s (commit: abc1234)'\n", version)
}

// writeCore writes a fake Core binary of the given version to path.
func writeCore(path, version string) {
	GinkgoHelper()
	Expect(os.WriteFile(path, fakeCore(version), 0o755)).To(Succeed())
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
		cfg                release.Config
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
		cmd := update.NewCmd(cfg)
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

		It("Should not update while an update of the data is pending", func() {
			writeCore(binary+".old", "0.59.0")
			Expect(os.WriteFile(
				filepath.Join(data, "UPDATING"),
				[]byte("0000000001-0.59.0\n0.60.0"),
				0o600,
			)).To(Succeed())
			Expect(run("--file", next)).To(MatchError(
				"An update of the data did not finish. Wait for the Core service to " +
					"finish it, and start the service if it stopped.",
			))
			Expect(systemctl.Calls()).To(BeEmpty())
			Expect(coreAt(binary + ".old")).To(ContainSubstring("0.59.0"))
		})

		It("Should do nothing when the Core is already installed", func() {
			writeCore(next, "0.60.0")
			Expect(run("--file", next)).To(Succeed())
			Expect(out.String()).To(Equal("Core 0.60.0 is already installed.\n"))
			Expect(systemctl.Calls()).To(BeEmpty())
			Expect(binary + ".old").ToNot(BeAnExistingFile())
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
				"Core 0.61.0 did not finish the update within 500ms. Check the Core log. " +
					"If the Core undoes the update, run `synnax update undo` to put back " +
					"Core 0.60.0.",
			))
			Expect(coreAt(binary)).To(ContainSubstring("0.61.0"))
		})
	})

	Describe("From a release", func() {
		var (
			files  string
			signer releasetest.Signer
		)
		BeforeEach(func() {
			files = GinkgoT().TempDir()
			signer = releasetest.NewSigner()
			server := httptest.NewServer(http.FileServer(http.Dir(files)))
			DeferCleanup(server.Close)
			cfg = release.DefaultConfig
			cfg.URL, cfg.PublicKey = server.URL, signer.PublicKey
		})

		// publish adds the Linux Core file and its signature to the release of version.
		publish := func(version string, core, signature []byte) {
			GinkgoHelper()
			path := filepath.Join(
				files,
				"core",
				"v"+version,
				"synnax-v"+version+"-linux",
			)
			Expect(os.MkdirAll(filepath.Dir(path), 0o755)).To(Succeed())
			Expect(os.WriteFile(path, core, 0o644)).To(Succeed())
			Expect(os.WriteFile(path+".sig", signature, 0o644)).To(Succeed())
		}

		It("Should download the signed Core and wait for the commit", func() {
			publish(
				"0.61.0",
				fakeCore("0.61.0"),
				signer.Sign("synnax-v0.61.0-linux", fakeCore("0.61.0")),
			)
			systemctl.OnStart(fmt.Sprintf(
				"printf 0.61.0 > %q",
				filepath.Join(data, "VERSION"),
			))
			Expect(run("0.61.0")).To(Succeed())
			Expect(out.String()).To(HavePrefix("Downloading Core 0.61.0.\n"))
			Expect(out.String()).To(HaveSuffix("Updated to Core 0.61.0.\n"))
			Expect(systemctl.Calls()).To(Equal([]string{"stop synnax", "start synnax"}))
			Expect(coreAt(binary)).To(ContainSubstring("0.61.0"))
			Expect(binary + ".new").ToNot(BeAnExistingFile())
		})

		It("Should keep the Core when the file fails its signature", func() {
			publish(
				"0.61.0",
				fakeCore("0.61.0"),
				signer.Sign("synnax-v0.61.0-linux", fakeCore("0.62.0")),
			)
			Expect(run("0.61.0")).To(MatchError(release.ErrSignature))
			Expect(systemctl.Calls()).To(BeEmpty())
			Expect(coreAt(binary)).To(ContainSubstring("0.60.0"))
			Expect(binary + ".new").ToNot(BeAnExistingFile())
		})

		It("Should keep the Core when the file holds another version", func() {
			publish(
				"0.61.0",
				fakeCore("0.62.0"),
				signer.Sign("synnax-v0.61.0-linux", fakeCore("0.62.0")),
			)
			Expect(run("0.61.0")).To(MatchError(
				"The release file synnax-v0.61.0-linux holds Core 0.62.0.",
			))
			Expect(systemctl.Calls()).To(BeEmpty())
			Expect(coreAt(binary)).To(ContainSubstring("0.60.0"))
		})

		It("Should keep the Core when the file does not run", func() {
			core := []byte("#!/bin/sh\nexit 1\n")
			publish("0.61.0", core, signer.Sign("synnax-v0.61.0-linux", core))
			Expect(run("0.61.0")).To(MatchError(ContainSubstring(
				"The release file synnax-v0.61.0-linux does not run on this computer",
			)))
			Expect(systemctl.Calls()).To(BeEmpty())
			Expect(coreAt(binary)).To(ContainSubstring("0.60.0"))
			Expect(binary + ".new").ToNot(BeAnExistingFile())
		})

		It("Should name the file that the release does not have", func() {
			Expect(run("0.61.0")).To(MatchError(
				"Core 0.61.0 has no signed release file synnax-v0.61.0-linux. Check " +
					"the version.",
			))
			Expect(systemctl.Calls()).To(BeEmpty())
		})

		It("Should not download a version that failed before", func() {
			Expect(os.WriteFile(
				filepath.Join(data, "UPDATE_FAILED"),
				[]byte("0.61.0"),
				0o600,
			)).To(Succeed())
			Expect(run("0.61.0")).To(MatchError(
				"The update to Core 0.61.0 failed before. Fix the cause, run `synnax " +
					"update retry`, then update again.",
			))
			Expect(out.String()).To(BeEmpty())
			Expect(systemctl.Calls()).To(BeEmpty())
		})

		It("Should not download an older Core", func() {
			Expect(run("0.59.0")).To(MatchError(
				"Core 0.59.0 is older than the installed Core 0.60.0. To go back, run " +
					"`synnax update undo`.",
			))
			Expect(out.String()).To(BeEmpty())
			Expect(systemctl.Calls()).To(BeEmpty())
		})

		It("Should not download the Core that is already installed", func() {
			Expect(run("0.60.0")).To(Succeed())
			Expect(out.String()).To(Equal("Core 0.60.0 is already installed.\n"))
			Expect(systemctl.Calls()).To(BeEmpty())
		})

		It("Should refuse a version that is not a Core version", func() {
			err := run("latest")
			Expect(err).To(MatchError("latest is not a Core version, such as 0.61.0."))
			Expect(errors.Is(err, validate.ErrValidation)).To(BeTrue())
		})

		It("Should refuse a version and a file together", func() {
			err := run("0.61.0", "--file", next)
			Expect(err).To(MatchError("Give a version or --file, not both."))
			Expect(errors.Is(err, validate.ErrValidation)).To(BeTrue())
			Expect(systemctl.Calls()).To(BeEmpty())
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

		It("Should put back an older binary for a backup with no version", func() {
			createBackup(data, "0000000001-unknown")
			Expect(run(
				"undo",
				"--log-file-path", filepath.Join(data, "logs", "synnax.log"),
			)).To(Succeed())
			Expect(out.String()).To(HaveSuffix(
				"Put back Core 0.60.0 at " + binary + ". Start the Core.\n",
			))
			Expect(coreAt(binary)).To(ContainSubstring("0.60.0"))
		})

		Describe("After the Core undid the update", func() {
			It("Should put back the binary", func() {
				createBackup(data, "0000000001-0.60.0")
				Expect(os.WriteFile(
					filepath.Join(data, "VERSION"),
					[]byte("0.60.0"),
					0o600,
				)).To(Succeed())
				Expect(run(
					"undo",
					"--log-file-path", filepath.Join(data, "logs", "synnax.log"),
				)).To(Succeed())
				Expect(out.String()).To(Equal(
					"The Core already undid the update.\n" +
						"Put back Core 0.60.0 at " + binary + ". Start the Core.\n",
				))
				Expect(coreAt(binary)).To(ContainSubstring("0.60.0"))
			})

			It("Should keep a replaced binary of another version", func() {
				createBackup(data, "0000000001-0.60.0")
				Expect(os.WriteFile(
					filepath.Join(data, "VERSION"),
					[]byte("0.60.0"),
					0o600,
				)).To(Succeed())
				writeCore(binary+".old", "0.59.0")
				Expect(run(
					"undo",
					"--log-file-path", filepath.Join(data, "logs", "synnax.log"),
				)).To(MatchError(HavePrefix("The last update is already undone.")))
				Expect(coreAt(binary)).To(ContainSubstring("0.61.0"))
			})

			Describe("With a backup that has no version", func() {
				BeforeEach(func() {
					createBackup(data, "0000000001-unknown")
					Expect(os.Remove(filepath.Join(data, "VERSION"))).To(Succeed())
				})

				It("Should put back an older binary", func() {
					Expect(run(
						"undo",
						"--log-file-path", filepath.Join(data, "logs", "synnax.log"),
					)).To(Succeed())
					Expect(out.String()).To(HaveSuffix(
						"Put back Core 0.60.0 at " + binary + ". Start the Core.\n",
					))
					Expect(coreAt(binary)).To(ContainSubstring("0.60.0"))
				})

				It("Should keep a newer binary", func() {
					writeCore(binary+".old", "0.61.0")
					writeCore(binary, "0.60.0")
					Expect(run(
						"undo",
						"--log-file-path", filepath.Join(data, "logs", "synnax.log"),
					)).To(MatchError(
						HavePrefix("The last update is already undone."),
					))
					Expect(coreAt(binary)).To(ContainSubstring("0.60.0"))
				})
			})
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

		It("Should not restore the backup when the service lookup fails", func() {
			createBackup(data, "0000000001-0.60.0")
			systemctl.Install(
				filepath.Dir(binary),
				binary,
				"start",
				"--config",
				filepath.Join(data, "missing.yaml"),
			)
			Expect(run(
				"undo",
				"--data", data,
				"--log-file-path", filepath.Join(data, "logs", "synnax.log"),
			)).To(MatchError(HavePrefix("failed to read service configuration")))
			Expect(MustSucceed(os.ReadFile(filepath.Join(data, "VERSION")))).
				To(BeEquivalentTo("0.61.0"))
		})

		It("Should name the Core to run when the put back fails", func() {
			createBackup(data, "0000000001-0.60.0")
			Expect(os.WriteFile(binary+".old", []byte("not a binary"), 0o755)).
				To(Succeed())
			Expect(run(
				"undo",
				"--log-file-path", filepath.Join(data, "logs", "synnax.log"),
			)).To(MatchError(
				HavePrefix("failed to put back the binary at " + binary + ": "),
			))
			Expect(out.String()).To(HaveSuffix("Run Core 0.60.0 on the data.\n"))
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
