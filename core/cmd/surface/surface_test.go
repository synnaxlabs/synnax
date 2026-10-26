// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package surface_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/synnaxlabs/synnax/cmd/surface"
)

func report(root *cobra.Command) string {
	GinkgoHelper()
	var b strings.Builder
	Expect(surface.Write(&b, root)).To(Succeed())
	return b.String()
}

func tree(dataHelp string) *cobra.Command {
	GinkgoHelper()
	root := &cobra.Command{Use: "synnax"}
	root.PersistentFlags().Bool("verbose", false, "Verbose output")
	start := &cobra.Command{Use: "start", Aliases: []string{"run"}}
	start.Flags().StringP("data", "d", "synnax-data", dataHelp)
	start.Flags().Int("workers", 4, "Workers")
	Expect(start.Flags().MarkHidden("workers")).To(Succeed())
	root.AddCommand(start)
	return root
}

var _ = Describe("Write", func() {
	It("Should write every command and flag, sorted", func() {
		Expect(
			report(tree("Data directory")),
		).To(Equal(`synnax --verbose bool default="false" persistent
synnax command
synnax start --data -d string default="synnax-data"
synnax start --workers int default="4" hidden
synnax start command aliases=run
`))
	})

	It("Should ignore help text", func() {
		Expect(report(tree("Where data lives"))).To(Equal(report(tree("Data"))))
	})

	It("Should tell persistent flags from local ones", func() {
		local := &cobra.Command{Use: "synnax"}
		local.Flags().Bool("verbose", false, "Verbose output")
		persistent := &cobra.Command{Use: "synnax"}
		persistent.PersistentFlags().Bool("verbose", false, "Verbose output")
		Expect(report(local)).ToNot(Equal(report(persistent)))
	})
})
