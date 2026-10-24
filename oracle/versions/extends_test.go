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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	. "github.com/synnaxlabs/x/testutil"
)

const (
	extendsV0Path = "schemas/synnax/versions/channel/v0.oracle"
	extendsV1Path = "schemas/synnax/versions/channel/v1.oracle"
	extendsV2Path = "schemas/synnax/versions/channel/v2.oracle"
)

const extendsV0 = `
Extra struct {
    note string
}

Node union on variant {
    leaf {
        value int32
    }
    box {
        width int32
        height int32 = 4
    }
}

Holder struct {
    key uuid @key
    name string {
        @doc value "names the holder."
    }
    count int32
    node Node
    old bool

    @go marshal
}
`

// renderVersion renders the declarations that version n of the channel chain defines,
// with every reference left fully qualified.
func renderVersion(ctx context.Context, files map[string]string, n int) string {
	GinkgoHelper()
	f := MustSucceed(resolverFor(files).File(ctx, "schemas/synnax/channel", n))
	decls := make([]versions.Decl, 0, len(f.Defined))
	for _, t := range f.Defined {
		decls = append(decls, versions.Decl{Type: t})
	}
	return versions.Render(decls, versions.RenderOptions{Resolve: f.Table.Get})
}

var _ = Describe("Version extends", func() {
	DescribeTable("Should resolve a patch exactly like its hand-written copy", func(
		ctx SpecContext, patch, handWritten string,
	) {
		Expect(renderVersion(ctx, map[string]string{
			extendsV0Path: extendsV0,
			extendsV1Path: patch,
		}, 1)).To(Equal(renderVersion(ctx, map[string]string{
			extendsV0Path: extendsV0,
			extendsV1Path: handWritten,
		}, 1)))
	},
		Entry("a typed field replaces the parent field in place", `
Node = v0.Node

Holder struct extends v0.Holder {
    count int64
}
`, `
Node = v0.Node

Holder struct {
    key uuid @key
    name string {
        @doc value "names the holder."
    }
    count int64
    node Node
    old bool

    @go marshal
}
`),
		Entry("a typeless field merges over the parent field in place", `
Node = v0.Node

Holder struct extends v0.Holder {
    count?
    name {
        @doc value "renames the holder."
    }
}
`, `
Node = v0.Node

Holder struct {
    key uuid @key
    name string {
        @doc value "renames the holder."
    }
    count int32?
    node Node
    old bool

    @go marshal
}
`),
		Entry("a domain removal applies to the parent field", `
Node = v0.Node

Holder struct extends v0.Holder {
    name -@doc
}
`, `
Node = v0.Node

Holder struct {
    key uuid @key
    name string
    count int32
    node Node
    old bool

    @go marshal
}
`),
		Entry("a field omit removes the parent field", `
Node = v0.Node

Holder struct extends v0.Holder {
    -old
}
`, `
Node = v0.Node

Holder struct {
    key uuid @key
    name string {
        @doc value "names the holder."
    }
    count int32
    node Node

    @go marshal
}
`),
		Entry("new fields and extends entries append", `
Node = v0.Node
Extra = v0.Extra

Holder struct extends v0.Holder, Extra {
    created int64
}
`, `
Node = v0.Node
Extra = v0.Extra

Holder struct extends Extra {
    key uuid @key
    name string {
        @doc value "names the holder."
    }
    count int32
    node Node
    old bool
    created int64

    @go marshal
}
`),
		Entry("a new field follows the parent field written above it", `
Node = v0.Node

Holder struct extends v0.Holder {
    name
    alias string
    count int64
    total int32
}
`, `
Node = v0.Node

Holder struct {
    key uuid @key
    name string {
        @doc value "names the holder."
    }
    alias string
    count int64
    total int32
    node Node
    old bool

    @go marshal
}
`),
		Entry("a new variant follows the parent variant written above it", `
Extra = v0.Extra
Holder = v0.Holder

Node union on variant extends v0.Node {
    leaf {}
    tree {
        size int32
    }
}
`, `
Extra = v0.Extra
Holder = v0.Holder

Node union on variant {
    leaf {
        value int32
    }
    tree {
        size int32
    }
    box {
        width int32
        height int32 = 4
    }
}
`),
		Entry("the copy resolves names in the new version", `
Node union on variant extends v0.Node {
    tree {
        size int32
    }
}

Holder struct extends v0.Holder {}
`, `
Node union on variant {
    leaf {
        value int32
    }
    box {
        width int32
        height int32 = 4
    }
    tree {
        size int32
    }
}

Holder struct {
    key uuid @key
    name string {
        @doc value "names the holder."
    }
    count int32
    node Node
    old bool

    @go marshal
}
`),
		Entry("an inline variant patches the parent variant's fields", `
Extra = v0.Extra
Holder = v0.Holder

Node union on variant extends v0.Node {
    box extends Extra {
        height int64
        depth int32
    }
}
`, `
Extra = v0.Extra
Holder = v0.Holder

Node union on variant {
    leaf {
        value int32
    }
    box extends Extra {
        width int32
        height int64
        depth int32
    }
}
`),
		Entry("a named variant replaces the parent variant in place", `
Extra = v0.Extra
Holder = v0.Holder

Node union on variant extends v0.Node {
    leaf Extra
}
`, `
Extra = v0.Extra
Holder = v0.Holder

Node union on variant {
    leaf Extra
    box {
        width int32
        height int32 = 4
    }
}
`),
	)

	It("Should copy through a chain of patches", func(ctx SpecContext) {
		v1 := `
Node = v0.Node

Holder struct extends v0.Holder {
    count int64
}
`
		Expect(renderVersion(ctx, map[string]string{
			extendsV0Path: extendsV0,
			extendsV1Path: v1,
			extendsV2Path: `
Node = v0.Node

Holder struct extends v1.Holder {
    -old
}
`,
		}, 2)).To(Equal(renderVersion(ctx, map[string]string{
			extendsV0Path: extendsV0,
			extendsV1Path: v1,
			extendsV2Path: `
Node = v0.Node

Holder struct {
    key uuid @key
    name string {
        @doc value "names the holder."
    }
    count int64
    node Node

    @go marshal
}
`,
		}, 2)))
	})

	DescribeTable("Should keep a removed field omitted where a base has it", func(
		ctx SpecContext, v0, patch, handWritten string,
	) {
		Expect(renderVersion(ctx, map[string]string{
			extendsV0Path: v0,
			extendsV1Path: patch,
		}, 1)).To(Equal(renderVersion(ctx, map[string]string{
			extendsV0Path: v0,
			extendsV1Path: handWritten,
		}, 1)))
	},
		Entry("on a struct", `
Base struct {
    x int32
}

Holder struct extends Base {
    x int64
    y int32
}
`, `
Base = v0.Base

Holder struct extends v0.Holder {
    -x
}
`, `
Base = v0.Base

Holder struct extends Base {
    -x
    y int32
}
`),
		Entry("on an inline variant", `
Base struct {
    x int32
}

Node union on variant {
    box extends Base {
        x int64
        y int32
    }
}
`, `
Base = v0.Base

Node union on variant extends v0.Node {
    box {
        -x
    }
}
`, `
Base = v0.Base

Node union on variant {
    box extends Base {
        -x
        y int32
    }
}
`),
		Entry("on a union base", `
Base struct {
    x int32
}

Node union on variant extends Base {
    box {
        x int64
        y int32
    }
}
`, `
Base = v0.Base

Node union on variant extends v0.Node {
    box {
        -x
    }
}
`, `
Base = v0.Base

Node union on variant extends Base {
    box {
        -x
        y int32
    }
}
`),
		Entry("when a later patch removes the field from the base", `
C struct {
    x int32
}

A struct extends C {
    x int64
}

B struct extends A {
    x int64?
}
`, `
C = v0.C

B struct extends v0.B {
    -x
}

A struct extends v0.A {
    -x
}
`, `
C = v0.C

B struct extends A {}

A struct extends C {
    -x
}
`),
		Entry("through a typeless parent field", `
Base struct {
    x int32 {
        @doc value "is x."
    }
}

Holder struct extends Base {
    x?
}
`, `
Base = v0.Base

Holder struct extends v0.Holder {
    x -@doc
}
`, `
Base = v0.Base

Holder struct extends Base {
    x? -@doc
}
`),
	)

	It("Should count an import that only the copy uses", func(ctx SpecContext) {
		f := MustSucceed(resolverFor(map[string]string{
			"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
			extendsV0Path: `
import "schemas/x/versions/telem/v0"

Holder struct {
    created telem.TimeStamp
}
`,
			extendsV1Path: `
import "schemas/x/versions/telem/v0"

Holder struct extends v0.Holder {
    name string
}
`,
		}).File(ctx, "schemas/synnax/channel", 1))
		holder := MustBeOk(f.Table.Get("v1.Holder"))
		Expect(holder.Form.(resolution.StructForm).Fields).To(HaveLen(2))
	})

	It("Should not copy an extends of a different name", func(ctx SpecContext) {
		f := MustSucceed(resolverFor(map[string]string{
			extendsV0Path: extendsV0,
			extendsV1Path: `
Node = v0.Node

Tree union on variant extends v0.Node {
    tree {
        size int32
    }
}
`,
		}).File(ctx, "schemas/synnax/channel", 1))
		form := MustBeOk(f.Table.Get("v1.Tree")).Form.(resolution.UnionForm)
		Expect(form.Included).To(ConsistOf(HaveField("Name", "v0.Node")))
		Expect(form.Declared).To(ConsistOf(HaveField("Name", "tree")))
	})

	It("Should report a copied variant's errors at the copying declaration", func(
		ctx SpecContext,
	) {
		Expect(resolverFor(map[string]string{
			extendsV0Path: extendsV0,
			extendsV1Path: `
Holder = v0.Holder

NodeBoxPayload struct {
    width int32
}

Node union on variant extends v0.Node {}
`,
		}).File(ctx, "schemas/synnax/channel", 1)).Error().To(MatchError(
			ContainSubstring(
				`8:0 error: union Node variant "box": inline payload name ` +
					"v1.NodeBoxPayload collides with an existing type",
			),
		))
	})

	It("Should reject a struct that extends an earlier union of its name", func(
		ctx SpecContext,
	) {
		Expect(resolverFor(map[string]string{
			extendsV0Path: extendsV0,
			extendsV1Path: "Node struct extends v0.Node {}\n",
		}).File(ctx, "schemas/synnax/channel", 1)).Error().To(MatchError(
			ContainSubstring("struct Node: v0.Node is not a struct"),
		))
	})

	It("Should reject a union that extends an earlier struct of its name", func(
		ctx SpecContext,
	) {
		Expect(resolverFor(map[string]string{
			extendsV0Path: extendsV0,
			extendsV1Path: "Holder union on variant extends v0.Holder {}\n",
		}).File(ctx, "schemas/synnax/channel", 1)).Error().To(MatchError(
			ContainSubstring("union Holder: v0.Holder is not a union"),
		))
	})

	It("Should reject more than one earlier version of itself", func(
		ctx SpecContext,
	) {
		Expect(resolverFor(map[string]string{
			extendsV0Path: extendsV0,
			extendsV1Path: "Holder struct extends v0.Holder {}\n",
			extendsV2Path: "Holder struct extends v0.Holder, v1.Holder {}\n",
		}).File(ctx, "schemas/synnax/channel", 2)).Error().To(MatchError(
			ContainSubstring(
				"Holder: extends more than one earlier version of itself " +
					"(v0.Holder and v1.Holder)",
			),
		))
	})
})
