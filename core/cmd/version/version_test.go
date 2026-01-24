// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package version_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/cmd/version"
)

var _ = Describe("Version", func() {
	Describe("FPrint", func() {
		It("Should print the version", func() {
			var buf bytes.Buffer
			Expect(version.FPrint(&buf)).To(Succeed())
			Expect(buf.String()).To(Equal(expected))
		})
	})
	Describe("AddCommand", func() {
		It("Should register the version subcommand", func() {
			var buf bytes.Buffer
			version.Cmd.SetOut(&buf)
			Expect(version.Cmd.Execute()).To(Succeed())
			Expect(buf.String()).To(Equal(expected))
		})
	})
})
