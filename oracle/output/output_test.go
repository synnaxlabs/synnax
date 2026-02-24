// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package output_test

import (
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/output"
)

var _ = Describe("Output", func() {
	var (
		origStdout *os.File
		r, w       *os.File
	)

	BeforeEach(func() {
		origStdout = os.Stdout
		var err error
		r, w, err = os.Pipe()
		Expect(err).ToNot(HaveOccurred())
		os.Stdout = w
	})

	AfterEach(func() {
		os.Stdout = origStdout
		Expect(r.Close()).To(Succeed())
	})

	captureOutput := func(fn func()) string {
		fn()
		Expect(w.Close()).To(Succeed())
		out, err := io.ReadAll(r)
		Expect(err).ToNot(HaveOccurred())
		return string(out)
	}

	Describe("PluginStart", func() {
		It("should print a generating message", func() {
			captured := captureOutput(func() {
				output.PluginStart("go")
			})
			Expect(captured).To(ContainSubstring("go"))
			Expect(captured).To(ContainSubstring("generating"))
		})
	})

	Describe("PluginDone", func() {
		It("should print file count with singular form", func() {
			captured := captureOutput(func() {
				output.PluginDone("go", 1)
			})
			Expect(captured).To(ContainSubstring("go"))
			Expect(captured).To(ContainSubstring("1"))
			Expect(captured).To(ContainSubstring("file"))
			Expect(captured).ToNot(ContainSubstring("files"))
		})

		It("should print file count with plural form", func() {
			captured := captureOutput(func() {
				output.PluginDone("go", 5)
			})
			Expect(captured).To(ContainSubstring("5"))
			Expect(captured).To(ContainSubstring("files"))
		})
	})

	Describe("PostWriteStep", func() {
		It("should print the tool name and file count", func() {
			captured := captureOutput(func() {
				output.PostWriteStep("gofmt", 3, "formatting")
			})
			Expect(captured).To(ContainSubstring("gofmt"))
			Expect(captured).To(ContainSubstring("3"))
			Expect(captured).To(ContainSubstring("formatting"))
			Expect(captured).To(ContainSubstring("files"))
		})

		It("should use singular form for one file", func() {
			captured := captureOutput(func() {
				output.PostWriteStep("eslint", 1, "linting")
			})
			Expect(captured).To(ContainSubstring("1"))
			Expect(captured).To(ContainSubstring("file"))
			Expect(captured).ToNot(ContainSubstring("files"))
		})
	})
})
