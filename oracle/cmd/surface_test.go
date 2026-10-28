// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import (
	"io"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("surface command", func() {
	It("Should print the types and fields of every schema", func() {
		_, cleanup := setupMiniRepo(map[string]string{
			"widget.oracle": "Valve struct {\n    name string\n}\n",
		})
		DeferCleanup(cleanup)
		Expect(MustSucceed(executeCommand(newRootCmd(), "surface"))).
			To(Equal("widget.Valve struct\nwidget.Valve.name string\n"))
	})

	It("Should fail when a schema does not analyze", func() {
		_, cleanup := setupMiniRepo(map[string]string{
			"widget.oracle": "Valve struct { name string }\nValve struct { size int32 }\n",
		})
		DeferCleanup(cleanup)
		var stderr strings.Builder
		root := newRootCmd()
		root.SetOut(io.Discard)
		root.SetErr(&stderr)
		root.SetArgs([]string{"surface"})
		Expect(root.Execute()).
			To(MatchError(ContainSubstring("duplicate type definition: widget.Valve")))
		Expect(stderr.String()).To(ContainSubstring("analysis failed"))
	})
})
