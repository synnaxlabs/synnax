// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package update_test

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/cmd/update"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Update", func() {
	var dir string
	BeforeEach(func() {
		dir = MustSucceed(os.MkdirTemp("", "synnax-update"))
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })
	})

	createBackup := func(name string) {
		GinkgoHelper()
		backups := filepath.Join(dir, "backups")
		Expect(os.MkdirAll(backups, 0o700)).To(Succeed())
		f := MustSucceed(os.Create(filepath.Join(backups, name+".tar")))
		w := tar.NewWriter(f)
		for _, part := range []string{"kv/", "cesium/"} {
			Expect(w.WriteHeader(&tar.Header{
				Typeflag: tar.TypeDir,
				Name:     part,
				Mode:     0o700,
			})).To(Succeed())
		}
		Expect(w.Close()).To(Succeed())
		Expect(f.Close()).To(Succeed())
	}

	run := func(args ...string) (string, error) {
		var out bytes.Buffer
		update.Cmd.SetOut(&out)
		update.Cmd.SetErr(io.Discard)
		update.Cmd.SetArgs(append(
			args,
			"--data", dir,
			"--log-file-path", filepath.Join(dir, "logs", "synnax.log"),
		))
		err := update.Cmd.Execute()
		return out.String(), err
	}

	Describe("History", func() {
		It("Should say when there are no backups", func() {
			Expect(MustSucceed(run("history"))).To(Equal("There are no backups.\n"))
		})

		It("Should list the backups", func() {
			createBackup("0000000001-0.59.0")
			createBackup("0000000002-0.60.0")
			out := MustSucceed(run("history"))
			Expect(out).To(ContainSubstring("BACKUP"))
			Expect(out).To(MatchRegexp(`0000000001-0\.59\.0 .* 0\.59\.0\n`))
			Expect(out).To(MatchRegexp(`0000000002-0\.60\.0 .* 0\.60\.0\n`))
			Expect(out).To(ContainSubstring("synnax update undo <backup>"))
		})
	})

	Describe("Undo", func() {
		BeforeEach(func() {
			Expect(
				os.WriteFile(filepath.Join(dir, "VERSION"), []byte("0.60.0"), 0o600),
			).
				To(Succeed())
		})

		It("Should restore the newest backup and name the Core to run", func() {
			createBackup("0000000001-0.58.0")
			createBackup("0000000002-0.59.0")
			Expect(MustSucceed(run("undo"))).To(Equal(
				"Restored backup 0000000002-0.59.0.\nRun Core 0.59.0 on the data.\n",
			))
			Expect(os.ReadFile(filepath.Join(dir, "VERSION"))).
				To(BeEquivalentTo("0.59.0"))
		})

		It("Should restore the named backup", func() {
			createBackup("0000000001-0.58.0")
			createBackup("0000000002-0.59.0")
			Expect(MustSucceed(run("undo", "0000000001-0.58.0"))).
				To(HavePrefix("Restored backup 0000000001-0.58.0.\n"))
			Expect(os.ReadFile(filepath.Join(dir, "VERSION"))).
				To(BeEquivalentTo("0.58.0"))
		})

		It("Should name where it moved channels created after the backup", func() {
			createBackup("0000000001-0.59.0")
			Expect(os.MkdirAll(filepath.Join(dir, "cesium", "7"), 0o700)).To(Succeed())
			Expect(MustSucceed(run("undo"))).To(ContainSubstring(
				"Moved the telemetry of channels created after the backup to " +
					filepath.Join(dir, "backups", "orphaned-"),
			))
		})

		It("Should say when the backup does not record its version", func() {
			createBackup("0000000001-unknown")
			Expect(MustSucceed(run("undo"))).
				To(ContainSubstring("The backup does not record its Core version."))
		})

		It("Should not undo the same update twice", func() {
			createBackup("0000000001-0.59.0")
			MustSucceed(run("undo"))
			Expect(run("undo")).Error().To(MatchError(ContainSubstring(
				"run `synnax update undo 0000000001-0.59.0`",
			)))
		})

		It("Should return an error when there is no backup", func() {
			Expect(run("undo")).Error().
				To(MatchError(ContainSubstring("there is no update to undo")))
		})

		It("Should return an error for a missing backup", func() {
			Expect(run("undo", "0000000001-0.59.0")).Error().To(SatisfyAll(
				MatchError(query.ErrNotFound),
				MatchError(
					ContainSubstring("failed to restore backup 0000000001-0.59.0"),
				),
			))
		})
	})
})
