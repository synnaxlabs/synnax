// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/cmd"
	"github.com/synnaxlabs/synnax/cmd/exit"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Cmd", func() {
	It("Should print the version when using the --version flag", func() {
		var buf bytes.Buffer
		cmd.Cmd.SetArgs([]string{"--version"})
		cmd.Cmd.SetOut(&buf)
		Expect(cmd.Cmd.Execute()).To(Succeed())
		Expect(buf.String()).To(Equal("Synnax " + version.Full() + "\n"))
	})

	It("Should report a flag that does not parse as a validation error", func() {
		cmd.Cmd.SetArgs([]string{"version", "--bogus"})
		cmd.Cmd.SetOut(new(bytes.Buffer))
		cmd.Cmd.SetErr(new(bytes.Buffer))
		Expect(cmd.Cmd.Execute()).To(And(
			MatchError(
				func(err error) bool { return errors.Is(err, validate.ErrValidation) },
				"validate.ErrValidation",
			),
			MatchError("unknown flag: --bogus"),
		))
	})

	Describe("Config file", func() {
		BeforeEach(func() {
			flag := cmd.Cmd.PersistentFlags().Lookup("config")
			DeferCleanup(func() {
				Expect(flag.Value.Set(flag.DefValue)).To(Succeed())
				flag.Changed = false
			})
			cmd.Cmd.SetOut(new(bytes.Buffer))
			cmd.Cmd.SetErr(new(bytes.Buffer))
		})

		It(
			"Should fail with the config exit code when the file does not parse",
			func() {
				path := filepath.Join(GinkgoT().TempDir(), "config.yaml")
				Expect(
					os.WriteFile(path, []byte("insecure: [true"), 0o600),
				).To(Succeed())
				cmd.Cmd.SetArgs([]string{"version", "--config", path})
				err := cmd.Cmd.Execute()
				Expect(err).To(MatchError(ContainSubstring(path)))
				Expect(exit.Code(err)).To(Equal(exit.Config))
			},
		)

		It(
			"Should fail with the config exit code when a passed path does not exist",
			func() {
				path := filepath.Join(GinkgoT().TempDir(), "missing.yaml")
				cmd.Cmd.SetArgs([]string{"version", "--config", path})
				err := cmd.Cmd.Execute()
				Expect(err).To(MatchError(ContainSubstring(path)))
				Expect(exit.Code(err)).To(Equal(exit.Config))
			},
		)

		It("Should succeed when the default path does not exist", func() {
			flag := cmd.Cmd.PersistentFlags().Lookup("config")
			path := filepath.Join(GinkgoT().TempDir(), "missing.yaml")
			Expect(flag.Value.Set(path)).To(Succeed())
			cmd.Cmd.SetArgs([]string{"version"})
			Expect(cmd.Cmd.Execute()).To(Succeed())
		})
	})
})
