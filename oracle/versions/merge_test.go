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
	"strings"

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

	It(
		"Should carry live-only annotations on union variant fields, scoped per variant",
		func(ctx SpecContext) {
			write("schemas/synnax/versions/channel/v0.oracle", `
Scale union on type {
    linear {
        slope float64 = 1
    }
    map {
        scaled_min float64 = 0
        scaled_max float64 = 1
        slope      float64 = 1
    }

    @go marshal
}

Channel struct {
    scale Scale = linear

    @go marshal
}
`)
			liveSource := `@go output "core/pkg/service/channel"

Scale union on type {
    linear {
        slope float64 = 1 {
            @validate required
        }
    }
    map {
        scaled_min float64 = 0 {
            @default group "scaled"
        }
        scaled_max float64 = 1 {
            @default group "scaled"
        }
        slope      float64 = 1
    }
}

Channel struct {
    scale Scale = linear
}
`
			r, chain := resolver()
			merged := MustSucceed(versions.MergeLive(ctx, r, chain, liveSource))
			Expect(merged).To(ContainSubstring(`@default group "scaled"`))
			Expect(strings.Count(merged, `@default group "scaled"`)).To(Equal(2))
			// linear.slope carries the marker; map.slope, sharing the name, does not.
			Expect(strings.Count(merged, "@validate required")).To(Equal(1))
			Expect(versions.MergeLive(ctx, r, chain, merged)).To(Equal(merged))
		},
	)

	It("Should carry live-owned action declarations through the merge", func(
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
    name string {
        @doc value "names the channel."
    }

    action Rename {
        name string {
            @doc value "is the new name."
        }

        @doc value "renames the channel."
    }

    @go marshal
}
`
		r, chain := resolver()
		merged := MustSucceed(versions.MergeLive(ctx, r, chain, liveSource))
		Expect(merged).To(ContainSubstring("action Rename {"))
		Expect(merged).To(ContainSubstring("is the new name."))
		Expect(merged).To(ContainSubstring("renames the channel."))
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

	It("Should leave the live source of an ended chain hand-owned", func(
		ctx SpecContext,
	) {
		write("schemas/synnax/versions/channel/v0.oracle", `
Channel struct {
    name string
}
`)
		write("schemas/synnax/versions/channel/v1.oracle", "// gone\n")
		r, chain := resolver()
		Expect(versions.MergeLive(ctx, r, chain, "Channel struct {\n}\n")).
			To(Equal(""))
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

	It("Should reject an unparsable live source", func(ctx SpecContext) {
		write("schemas/synnax/versions/channel/v0.oracle", `
Channel struct {
    name string
}
`)
		r, chain := resolver()
		Expect(versions.MergeLive(ctx, r, chain, "Channel struct {{{\n")).Error().
			To(MatchError(ContainSubstring("failed to parse")))
	})

	It("Should reject an ambiguous dependency namespace", func(ctx SpecContext) {
		write("schemas/x/versions/status/v0.oracle", "Code = int64\n")
		write("schemas/synnax/status.oracle", "Info struct {\n\tmessage string\n}\n")
		write("schemas/synnax/versions/channel/v0.oracle", `
import "schemas/x/versions/status/v0"

Channel struct {
	key uuid @key
	code status.Code

	@go marshal
}
`)
		write("schemas/synnax/versions/channel/v1.oracle", `
import "schemas/synnax/status"

Channel struct {
	key uuid @key
	info status.Info {
		@go marshal omit
	}

	@go marshal
}
`)
		r, chain := resolver()
		Expect(versions.MergeLive(ctx, r, chain, "")).Error().
			To(MatchError(ContainSubstring("namespace status is ambiguous")))
	})

	It("Should carry live-owned enum value annotations", func(ctx SpecContext) {
		write("schemas/synnax/versions/channel/v0.oracle", `
Kind enum {
    linear = "linear"
    log = "log"
}

Level enum {
    low = 0
    high = 1
}

Channel struct {
    kind Kind
    level Level

    @go marshal
}
`)
		liveSource := `Kind enum {
    linear = "linear" {
        @ts label "Linear"
    }
    log = "log"
}

Level enum {
    low = 0
    high = 1
}

Channel struct {
    kind Kind
    level Level

    @go marshal
}
`
		r, chain := resolver()
		merged := MustSucceed(versions.MergeLive(ctx, r, chain, liveSource))
		Expect(merged).To(ContainSubstring(`@ts label "Linear"`))
		Expect(merged).To(MatchRegexp(`low +?= 0`))
		Expect(versions.MergeLive(ctx, r, chain, merged)).To(Equal(merged))
	})

	It("Should project unions, distincts, aliases, and defaults", func(
		ctx SpecContext,
	) {
		write("schemas/synnax/versions/channel/v0.oracle", `
Base struct {
    key uuid @key
}

Leaf struct {
    name string
}

Node union on variant {
    leaf Leaf
    inline {
        depth int64
    }

    @go marshal
}

Wide struct extends Base {
    node Node
    weight float64 = 1.5
    active bool = false
    count int64 = 2
    label string = "x"

    @go marshal
}

Span int64

Ref = Leaf
`)
		r, chain := resolver()
		merged := MustSucceed(versions.MergeLive(ctx, r, chain, ""))
		Expect(merged).To(ContainSubstring("Node union on variant {"))
		Expect(merged).To(ContainSubstring("leaf Leaf"))
		Expect(merged).To(ContainSubstring("depth int64"))
		Expect(merged).To(ContainSubstring("Wide struct extends Base {"))
		Expect(merged).To(ContainSubstring("= 1.5"))
		Expect(merged).To(ContainSubstring("= false"))
		Expect(merged).To(ContainSubstring("= 2"))
		Expect(merged).To(ContainSubstring(`= "x"`))
		Expect(merged).To(ContainSubstring("Span int64"))
		Expect(merged).To(ContainSubstring("Ref = Leaf"))
		Expect(versions.MergeLive(ctx, r, chain, merged)).To(Equal(merged))
	})

	It("Should strip the hand marker from a non-current definer", func(
		ctx SpecContext,
	) {
		write("schemas/synnax/versions/channel/v0.oracle", `
Key = uuid {
    @go hand
}
`)
		write("schemas/synnax/versions/channel/v1.oracle", `
Key = v0.Key

Channel struct {
    key Key @key

    @go hand
    @go marshal
}
`)
		r, chain := resolver()
		merged := MustSucceed(versions.MergeLive(ctx, r, chain, ""))
		Expect(strings.Count(merged, "@go hand")).To(Equal(1))
		Expect(merged).To(ContainSubstring("Key = uuid\n"))
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
