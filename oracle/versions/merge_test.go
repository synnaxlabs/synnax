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

var _ = Describe("MergeLive", func() {
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

Use of this software is governed by the Business Source License included in the file
licenses/BSL.txt.
`)
	})

	resolver := func() (*versions.Resolver, versions.Chain) {
		chains := MustSucceed(versions.Discover(root))
		r := versions.NewResolver(chains, analyzer.NewStandardFileLoader(root))
		chain, ok := chains["schemas/synnax/channel"]
		Expect(ok).To(BeTrue())
		return r, chain
	}

	It("Should project version-owned content and carry live annotations", func(
		ctx SpecContext,
	) {
		write("schemas/synnax/versions/channel/v0.oracle", `
Key = uuid

Channel struct {
    key  Key    {
        @doc value "identifies the channel."
        @key
    }
    name string {
        @doc value "names the channel."
    }

    @doc value "is a channel."
    @go marshal
}
`)
		liveSource := `import "schemas/x/telem"

@ts output "client/ts/src/channel"
@pb

Key = uuid {
    @go hand
    @ts to_number
}

Channel struct {
    key  Key    {
        @doc value "identifies the channel."
        @key
        @index lookup
    }
    name string {
        @doc value "names the channel."
        @validate required
    }

    @doc value "is a channel."
    @go output "core/pkg/service/channel"
    @retrieve
}

New struct {
    name string
}
`
		r, chain := resolver()
		merged := MustSucceed(versions.MergeLive(ctx, r, chain, liveSource))
		Expect(merged).To(ContainSubstring("Copyright"))
		Expect(merged).To(ContainSubstring(`@ts output "client/ts/src/channel"`))
		Expect(merged).To(ContainSubstring("@pb"))
		// Version-owned marshal projected from the version file.
		Expect(merged).To(ContainSubstring("@go marshal"))
		// Live-owned annotations carried through the merge.
		Expect(merged).To(ContainSubstring("@index lookup"))
		Expect(merged).To(ContainSubstring("@validate required"))
		Expect(merged).To(MatchRegexp(`@go output +"core/pkg/service/channel"`))
		Expect(merged).To(ContainSubstring("@retrieve"))
		Expect(merged).To(ContainSubstring("@ts to_number"))
		// Wire-only declaration preserved verbatim.
		Expect(merged).To(ContainSubstring("New struct"))
		// The merge is idempotent: merging its own output changes nothing.
		Expect(versions.MergeLive(ctx, r, chain, merged)).To(Equal(merged))
	})

	It("Should overwrite version-owned drift with chain resolution", func(
		ctx SpecContext,
	) {
		write("schemas/synnax/versions/channel/v0.oracle", `
Channel struct {
    name string {
        @doc value "names the channel."
    }

    @go marshal
}
`)
		liveSource := `Channel struct {
    name uint32 {
        @doc value "hand-edited doc."
    }
    extra bool

    @go marshal
}
`
		r, chain := resolver()
		merged := MustSucceed(versions.MergeLive(ctx, r, chain, liveSource))
		Expect(merged).To(ContainSubstring("name string"))
		Expect(merged).To(ContainSubstring("names the channel."))
		Expect(merged).ToNot(ContainSubstring("hand-edited"))
		// Field membership is version-owned: the live-only field drops.
		Expect(merged).ToNot(ContainSubstring("extra"))
	})

	It("Should append chain members missing from the live file", func(
		ctx SpecContext,
	) {
		write("schemas/synnax/versions/channel/v0.oracle", `
Channel struct {
    name string
}

Added struct {
    key uint32
}
`)
		liveSource := `Channel struct {
    name string
}
`
		r, chain := resolver()
		merged := MustSucceed(versions.MergeLive(ctx, r, chain, liveSource))
		Expect(merged).To(ContainSubstring("Added struct"))
	})

	It("Should emit a skeleton for a chain with no live file", func(
		ctx SpecContext,
	) {
		write("schemas/synnax/versions/channel/v0.oracle", `
Channel struct {
    name string
}
`)
		r, chain := resolver()
		merged := MustSucceed(versions.MergeLive(ctx, r, chain, ""))
		Expect(merged).To(ContainSubstring("Copyright"))
		Expect(merged).To(ContainSubstring("Channel struct"))
	})

	It("Should resolve docs through alias lines to the definer", func(
		ctx SpecContext,
	) {
		write("schemas/synnax/versions/channel/v0.oracle", `
Channel struct {
    name string

    @doc value "is a channel."
}
`)
		write("schemas/synnax/versions/channel/v1.oracle", `
Channel = v0.Channel

Extra struct {
    key uint32
}
`)
		r, chain := resolver()
		merged := MustSucceed(versions.MergeLive(ctx, r, chain, ""))
		Expect(merged).To(ContainSubstring(`@doc value "is a channel."`))
		Expect(merged).To(ContainSubstring("Extra struct"))
	})

	It("Should import dependency live paths for pinned references", func(
		ctx SpecContext,
	) {
		write("schemas/x/versions/telem/v0.oracle", "TimeStamp = int64\n")
		write("schemas/x/telem.oracle", "TimeStamp = int64\n")
		write("schemas/synnax/versions/channel/v0.oracle", `
import "schemas/x/versions/telem/v0"

Channel struct {
    created telem.TimeStamp

    @go marshal
}
`)
		r, chain := resolver()
		merged := MustSucceed(versions.MergeLive(ctx, r, chain, ""))
		Expect(merged).To(ContainSubstring(`import "schemas/x/telem"`))
		Expect(merged).To(ContainSubstring("telem.TimeStamp"))
	})
})
