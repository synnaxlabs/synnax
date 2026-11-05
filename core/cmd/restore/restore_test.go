// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package restore_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/cmd/restore"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Restore", func() {
	var dir string
	BeforeEach(func() {
		dir = MustSucceed(os.MkdirTemp("", "synnax-restore"))
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })
	})

	createBackup := func(name string) {
		GinkgoHelper()
		for _, part := range []string{"kv", "cesium"} {
			Expect(os.MkdirAll(filepath.Join(dir, "backups", name, part), 0o700)).
				To(Succeed())
		}
	}

	run := func(args ...string) (string, error) {
		cmd := restore.NewCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(io.Discard)
		cmd.SetArgs(append(args, "--data", dir))
		err := cmd.Execute()
		return out.String(), err
	}

	It("Should say when there are no backups", func() {
		Expect(MustSucceed(run())).To(Equal("There are no backups in " + dir + ".\n"))
	})

	It("Should list the backups", func() {
		createBackup("0000000001-0.59.0")
		createBackup("0000000002-0.60.0")
		out := MustSucceed(run())
		Expect(out).To(ContainSubstring("BACKUP"))
		Expect(out).To(MatchRegexp(`0000000001-0\.59\.0 .* 0\.59\.0\n`))
		Expect(out).To(MatchRegexp(`0000000002-0\.60\.0 .* 0\.60\.0\n`))
		Expect(out).To(ContainSubstring("synnax restore <backup>"))
	})

	It("Should restore the named backup and name the Core to run", func() {
		createBackup("0000000001-0.59.0")
		Expect(MustSucceed(run("0000000001-0.59.0"))).To(Equal(
			"Restored backup 0000000001-0.59.0.\nRun Core 0.59.0 on the data.\n",
		))
		Expect(os.ReadFile(filepath.Join(dir, "VERSION"))).
			To(BeEquivalentTo("0.59.0"))
	})

	It("Should name where it moved channels the backup does not know", func() {
		createBackup("0000000001-0.59.0")
		Expect(os.MkdirAll(filepath.Join(dir, "cesium", "7"), 0o700)).To(Succeed())
		out := MustSucceed(run("0000000001-0.59.0"))
		Expect(out).To(ContainSubstring(
			"Moved the telemetry of channels that the backup does not know to " +
				filepath.Join(dir, "backups", "orphaned-"),
		))
	})

	It("Should say when the backup does not record its version", func() {
		createBackup("0000000001-unknown")
		Expect(MustSucceed(run("0000000001-unknown"))).
			To(ContainSubstring("The backup does not record its Core version."))
	})

	It("Should return an error for a missing backup", func() {
		Expect(run("0000000001-0.59.0")).Error().To(SatisfyAll(
			MatchError(query.ErrNotFound),
			MatchError(ContainSubstring("failed to restore backup 0000000001-0.59.0")),
		))
	})
})
