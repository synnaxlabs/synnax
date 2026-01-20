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
)

var _ = Describe("Cmd", func() {
	Describe("Version", func() {
		It("Should print the version when using the --version flag", func() {
			var buf bytes.Buffer
			Expect(cmd.ExecuteWithArgs([]string{"--version"}, &buf)).To(Succeed())
			Expect(buf.String()).To(Equal(version))
		})
	})
})
