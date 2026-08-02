// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package freeze_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin/go/freeze"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Canonical", func() {
	var (
		root     string
		resolver *versions.Resolver
		live     *resolution.Table
		write    func(rel, content string)
	)

	BeforeEach(func() {
		root = GinkgoT().TempDir()
		write = func(rel, content string) {
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		write("schemas/x/versions/telem/v0.oracle", "TimeStamp = int64\n")
		write("schemas/x/telem.oracle", "TimeStamp = int64\n")
		write("schemas/synnax/versions/channel/v0.oracle", `
import "schemas/x/versions/telem/v0"

Key = uuid

Legacy struct {
	key Key @key

	@go marshal
}

Channel struct {
	key Key @key
	created telem.TimeStamp

	@go marshal
}
`)
		// Live schema: Channel changed (new field), Key unchanged, Legacy
		// removed, Operation brand new. The cached field is omit and must not
		// freeze.
		liveSource := `
import "schemas/x/telem"

@go output "core/pkg/service/channel"

Key = uuid {
	@go version 1
}

Operation struct {
	kind string

	@go version 1
	@doc value "is a brand-new persisted type."
}

Channel struct {
	key Key @key
	created telem.TimeStamp
	ops Operation[]
	cached string {
		@go marshal omit
	}

	@go version 1
	@go marshal
	@doc value "is the live doc, not retroactively frozen."
}
`
		chains := MustSucceed(versions.Discover(root))
		resolver = versions.NewResolver(
			chains, analyzer.NewStandardFileLoader(root),
		)
		live = resolution.NewTable()
		diag := analyzer.AnalyzeSeeded(
			GinkgoT().Context(), liveSource,
			"schemas/synnax/channel.oracle", "channel",
			analyzer.NewStandardFileLoader(root), live,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
	})

	canonical := func(in freeze.Input) string {
		in.Live = live
		in.Resolver = resolver
		in.Chain = resolver.Chains()["schemas/synnax/channel"]
		return MustSucceed(freeze.Canonical(GinkgoT().Context(), in))
	}

	It("Should emit aliases, redeclarations, removals, and pins", func() {
		out := canonical(freeze.Input{N: 1, Pinned: set.New[string]()})
		Expect(out).To(ContainSubstring("Key = v0.Key"))
		Expect(out).To(ContainSubstring("Channel struct {"))
		Expect(out).To(MatchRegexp(`ops\s+Operation\[\]`))
		Expect(out).ToNot(ContainSubstring("Legacy"))
		Expect(out).ToNot(ContainSubstring("cached"))
		Expect(out).ToNot(ContainSubstring("@go version"))
		Expect(out).ToNot(ContainSubstring("@go output"))
		Expect(out).To(ContainSubstring(
			`import "schemas/x/versions/telem/v0"`,
		))
		// New names seed their doc; redeclared names inherit silently.
		Expect(out).To(ContainSubstring("is a brand-new persisted type."))
		Expect(out).ToNot(ContainSubstring("not retroactively frozen"))
	})

	It("Should be deterministic and resolve as a valid version file", func() {
		out := canonical(freeze.Input{N: 1})
		Expect(canonical(freeze.Input{N: 1})).To(Equal(out))
		write("schemas/synnax/versions/channel/v1.oracle", out)
		chains := MustSucceed(versions.Discover(root))
		fresh := versions.NewResolver(
			chains, analyzer.NewStandardFileLoader(root),
		)
		f := MustSucceed(fresh.File(
			GinkgoT().Context(), "schemas/synnax/channel", 1,
		))
		Expect(f.Aliases).To(HaveKeyWithValue(
			"Key", versions.Alias{Version: 0, Name: "Key"},
		))
		surf := MustSucceed(fresh.Surface(
			GinkgoT().Context(), "schemas/synnax/channel", 1,
		))
		Expect(surf).To(HaveKey("Channel"))
		Expect(surf).To(HaveKey("Operation"))
		Expect(surf).ToNot(HaveKey("Legacy"))
	})

	It("Should preserve explicit doc overrides and pinned markers", func() {
		out := canonical(freeze.Input{
			N:      1,
			Docs:   map[string]string{"Channel": "meaning changed at v1"},
			Pinned: set.New("Key"),
		})
		Expect(out).To(ContainSubstring("meaning changed at v1"))
		// Pinned members always declare fully so the marker rides the
		// declaration.
		Expect(out).To(ContainSubstring("@go pinned"))
		Expect(out).ToNot(ContainSubstring("Key = v0.Key"))
	})
})
