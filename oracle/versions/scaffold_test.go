// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/versions"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Scaffold", func() {
	var (
		root  string
		write func(rel, content string)
	)

	BeforeEach(func() {
		root = GinkgoT().TempDir()
		write = func(rel, content string) {
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		write("licenses/headers/template.txt", `Copyright {{YEAR}} Synnax Labs, Inc.
`)
	})

	It("Should scaffold alias lines to definers with carried imports", func(
		ctx SpecContext,
	) {
		write("schemas/x/versions/telem/v0.oracle", "TimeStamp = int64\n")
		write("schemas/synnax/versions/channel/v0.oracle", `
import "schemas/x/versions/telem/v0"

Key = uuid

Channel struct {
    key     Key @key
    created telem.TimeStamp

    @go marshal
}
`)
		write("schemas/synnax/versions/channel/v1.oracle", `
import "schemas/x/versions/telem/v0"

Key = v0.Key

Channel struct {
    key     Key @key
    created telem.TimeStamp
    name    string

    @go marshal
}
`)
		chains := MustSucceed(versions.Discover(root))
		r := versions.NewResolver(chains, analyzer.NewStandardFileLoader(root))
		out := MustSucceed(versions.Scaffold(
			ctx, r, chains["schemas/synnax/channel"],
		))
		Expect(out).To(ContainSubstring("Copyright"))
		Expect(out).To(ContainSubstring(`import "schemas/x/versions/telem/v0"`))
		// Every member aliases its definer, not the predecessor blindly.
		Expect(out).To(ContainSubstring("Key = v0.Key"))
		Expect(out).To(ContainSubstring("Channel = v1.Channel"))
	})

	It("Should redeclare omit-transient declarations in full", func(ctx SpecContext) {
		write("schemas/synnax/versions/channel/v0.oracle", `
Channel struct {
    key    uuid @key
    status Details {
        @go marshal omit
    }

    @go marshal
}

Details struct {
    running bool
}
`)
		chains := MustSucceed(versions.Discover(root))
		r := versions.NewResolver(chains, analyzer.NewStandardFileLoader(root))
		out := MustSucceed(versions.Scaffold(
			ctx, r, chains["schemas/synnax/channel"],
		))
		Expect(out).To(ContainSubstring("Channel = v0.Channel"))
		Expect(out).To(ContainSubstring("Details struct"))
		Expect(out).To(ContainSubstring("running bool"))
	})
})
