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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/cmd"
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
})
