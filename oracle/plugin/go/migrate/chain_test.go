// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/go/migrate"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	. "github.com/synnaxlabs/x/testutil"
)

// generateChain builds a two-version chain for resource in a temp repo, analyzes the
// live schema at the standard output path, and returns Generate's files by path.
func generateChain(resource, v0, v1, live string) map[string]string {
	GinkgoHelper()
	root := GinkgoT().TempDir()
	write := func(rel, content string) {
		GinkgoHelper()
		full := filepath.Join(root, rel)
		Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
		Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
	}
	write("schemas/synnax/versions/"+resource+"/v0.oracle", v0)
	write("schemas/synnax/versions/"+resource+"/v1.oracle", v1)
	chains := MustSucceed(versions.Discover(root))
	resolver := versions.NewResolver(chains, analyzer.NewStandardFileLoader(root))
	table := resolution.NewTable()
	diag := analyzer.AnalyzeSeeded(
		GinkgoT().Context(),
		"@go output \"core/pkg/service/"+resource+"\"\n\n"+live,
		"schemas/synnax/"+resource+".oracle", resource,
		analyzer.NewStandardFileLoader(root), table,
	)
	Expect(diag.Ok()).To(BeTrue(), diag.String())
	resp := MustSucceed(migrate.New().Generate(&plugin.Request{
		Resolutions: table, RepoRoot: root, Versions: resolver,
	}))
	out := make(map[string]string, len(resp.Files))
	for _, f := range resp.Files {
		out[f.Path] = string(f.Content)
	}
	return out
}

var _ = Describe("Chain migrate.gen.go", func() {
	It("Should emit auto-copies as a pure function of adjacent files", func() {
		root := GinkgoT().TempDir()
		write := func(rel, content string) {
			GinkgoHelper()
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		write("schemas/synnax/versions/channel/v0.oracle", `
Channel struct {
	key uuid @key
	name string

	@go marshal
	@go migrate
}
`)
		write("schemas/synnax/versions/channel/v1.oracle", `
Channel struct {
	key uuid @key
	name string
	virtual bool

	@go marshal
	@go migrate
}
`)
		chains := MustSucceed(versions.Discover(root))
		resolver := versions.NewResolver(
			chains, analyzer.NewStandardFileLoader(root),
		)
		table := resolution.NewTable()
		diag := analyzer.AnalyzeSeeded(
			GinkgoT().Context(), `
@go output "core/pkg/service/channel"

Channel struct {
	key uuid @key
	name string
	virtual bool

}
`,
			"schemas/synnax/channel.oracle", "channel",
			analyzer.NewStandardFileLoader(root), table,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		resp := MustSucceed(migrate.New().Generate(&plugin.Request{
			Resolutions: table, RepoRoot: root, Versions: resolver,
		}))
		var content string
		for _, f := range resp.Files {
			if f.Path == "core/pkg/service/channel/versions/v1/migrate.gen.go" {
				content = string(f.Content)
			}
		}
		Expect(content).ToNot(BeEmpty())
		Expect(content).To(ContainSubstring("package v1"))
		Expect(content).To(ContainSubstring("igrateChannel"))
		Expect(content).To(ContainSubstring("Key: old.Key"))
		Expect(content).To(ContainSubstring("Name: old.Name"))
	})

	It("Should leave a field re-keyed across primitive kinds to the hand migration",
		func() {
			root := GinkgoT().TempDir()
			write := func(rel, content string) {
				GinkgoHelper()
				full := filepath.Join(root, rel)
				Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
				Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
			}
			write("schemas/synnax/versions/task/v0.oracle", `
Key uint64 {
	@go marshal flex
}

Task struct {
	key Key @key
	name string

	@go marshal
	@go migrate
}
`)
			write("schemas/synnax/versions/task/v1.oracle", `
Key = uuid

Task struct {
	key Key @key
	name string

	@go marshal
	@go migrate
}
`)
			chains := MustSucceed(versions.Discover(root))
			resolver := versions.NewResolver(
				chains, analyzer.NewStandardFileLoader(root),
			)
			table := resolution.NewTable()
			diag := analyzer.AnalyzeSeeded(
				GinkgoT().Context(), `
@go output "core/pkg/service/task"

Key = uuid

Task struct {
	key Key @key
	name string

}
`,
				"schemas/synnax/task.oracle", "task",
				analyzer.NewStandardFileLoader(root), table,
			)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			resp := MustSucceed(migrate.New().Generate(&plugin.Request{
				Resolutions: table, RepoRoot: root, Versions: resolver,
			}))
			var content string
			for _, f := range resp.Files {
				if f.Path == "core/pkg/service/task/versions/v1/migrate.gen.go" {
					content = string(f.Content)
				}
			}
			Expect(content).ToNot(BeEmpty())
			Expect(content).To(ContainSubstring("Name: old.Name"))
			Expect(content).ToNot(ContainSubstring("Key: "))
			Expect(content).ToNot(ContainSubstring("autoMigrateKey"))
		})

	It("Should copy an extends parent pinned to one dependency version", func() {
		root := GinkgoT().TempDir()
		write := func(rel, content string) {
			GinkgoHelper()
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		write("schemas/synnax/versions/base/v0.oracle", `
Base struct {
	name string

	@go marshal
}
`)
		write("schemas/synnax/base.oracle", `
@go output "core/pkg/service/base"

Base struct {
	name string
}
`)
		write("schemas/synnax/versions/node/v0.oracle", `
import "schemas/synnax/versions/base/v0"

Node struct extends base.Base {
	key uuid @key

	@go marshal
	@go migrate
}
`)
		write("schemas/synnax/versions/node/v1.oracle", `
import "schemas/synnax/versions/base/v0"

Node struct extends base.Base {
	key uuid @key
	virtual bool

	@go marshal
	@go migrate
}
`)
		chains := MustSucceed(versions.Discover(root))
		resolver := versions.NewResolver(
			chains, analyzer.NewStandardFileLoader(root),
		)
		table := resolution.NewTable()
		diag := analyzer.AnalyzeSeeded(
			GinkgoT().Context(), `
import "schemas/synnax/base"

@go output "core/pkg/service/node"

Node struct extends base.Base {
	key uuid @key
	virtual bool
}
`,
			"schemas/synnax/node.oracle", "node",
			analyzer.NewStandardFileLoader(root), table,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		resp := MustSucceed(migrate.New().Generate(&plugin.Request{
			Resolutions: table, RepoRoot: root, Versions: resolver,
		}))
		var content string
		for _, f := range resp.Files {
			if f.Path == "core/pkg/service/node/versions/v1/migrate.gen.go" {
				content = string(f.Content)
			}
		}
		Expect(content).ToNot(BeEmpty())
		Expect(content).To(ContainSubstring("Base: old.Base"))
		Expect(content).To(ContainSubstring("Key: old.Key"))
	})

	It("Should route a bumped dependency pin through its exported wrapper", func() {
		root := GinkgoT().TempDir()
		write := func(rel, content string) {
			GinkgoHelper()
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		write("schemas/synnax/versions/base/v0.oracle", `
Base struct {
	name string

	@go marshal
}
`)
		write("schemas/synnax/versions/base/v1.oracle", `
Base struct {
	name string
	extra string

	@go marshal
}
`)
		write("schemas/synnax/base.oracle", `
@go output "core/pkg/service/base"

Base struct {
	name string
	extra string
}
`)
		write("schemas/synnax/versions/node/v0.oracle", `
import "schemas/synnax/versions/base/v0"

Node struct {
	key uuid @key
	base base.Base

	@go marshal
	@go migrate
}
`)
		write("schemas/synnax/versions/node/v1.oracle", `
import "schemas/synnax/versions/base/v1"

Node struct {
	key uuid @key
	base base.Base

	@go marshal
	@go migrate
}
`)
		chains := MustSucceed(versions.Discover(root))
		resolver := versions.NewResolver(
			chains, analyzer.NewStandardFileLoader(root),
		)
		table := resolution.NewTable()
		diag := analyzer.AnalyzeSeeded(
			GinkgoT().Context(), `
import "schemas/synnax/base"

@go output "core/pkg/service/node"

Node struct {
	key uuid @key
	base base.Base
}
`,
			"schemas/synnax/node.oracle", "node",
			analyzer.NewStandardFileLoader(root), table,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		resp := MustSucceed(migrate.New().Generate(&plugin.Request{
			Resolutions: table, RepoRoot: root, Versions: resolver,
		}))
		var content string
		for _, f := range resp.Files {
			if f.Path == "core/pkg/service/node/versions/v1/migrate.gen.go" {
				content = string(f.Content)
			}
		}
		Expect(content).ToNot(BeEmpty())
		Expect(content).To(ContainSubstring(".MigrateBase(ctx, old.Base)"))
		Expect(content).To(ContainSubstring("Base: base"))
		Expect(content).To(ContainSubstring("core/pkg/service/base/versions/v1"))
	})

	DescribeTable("auto-copy scenarios",
		func(
			resource, v0, v1, live string,
			wantFile bool,
			contains, excludes []string,
		) {
			files := generateChain(resource, v0, v1, live)
			path := "core/pkg/service/" + resource + "/versions/v1/migrate.gen.go"
			content, ok := files[path]
			Expect(ok).To(Equal(wantFile), content)
			for _, s := range contains {
				Expect(content).To(ContainSubstring(s))
			}
			for _, s := range excludes {
				Expect(content).ToNot(ContainSubstring(s))
			}
		},
		Entry("converts map values through the element migration",
			"board",
			`Item struct {
	name string

	@go marshal
}

Board struct {
	key uuid @key
	entries map<string, Item>

	@go marshal
	@go migrate
}
`,
			`Item struct {
	name string
	color string

	@go marshal
}

Board struct {
	key uuid @key
	entries map<string, Item>

	@go marshal
	@go migrate
}
`,
			`Item struct {
	name string
	color string
}

Board struct {
	key uuid @key
	entries map<string, Item>
}
`,
			true,
			[]string{
				"entries := make(map[string]Item, len(old.Entries))",
				"MigrateItem(ctx, v)",
				"Entries: entries",
			},
			[]string{"Entries: old.Entries"},
		),
		Entry("auto-copies value types without migrate entries",
			"color",
			`Color struct {
	hex string
	attrs map<string, string>
	tags string[]

	@go marshal
}
`,
			`Color struct {
	hex string
	attrs map<string, string>
	tags string[]
	opacity uint8

	@go marshal
}
`,
			`Color struct {
	hex string
	attrs map<string, string>
	tags string[]
	opacity uint8
}
`,
			true,
			[]string{
				"package v1",
				"func autoMigrateColor",
				"Hex: old.Hex",
				"Attrs: old.Attrs",
				"Tags: old.Tags",
			},
			[]string{"Opacity"},
		),
		Entry("emits nothing into a tombstoned chain",
			"channel",
			`Channel struct {
	key uuid @key
	name string

	@go marshal
	@go migrate
}
`,
			"// Tombstone: channel stopped persisting at v1.\n",
			`Channel struct {
	key uuid @key
	name string
}
`,
			false, nil, nil,
		),
		Entry("decorates generic migrations with their type params",
			"box",
			`Box struct<T extends record> {
	value T
	label string

	@go marshal
}
`,
			`Box struct<T extends record> {
	value T
	label string
	extra string

	@go marshal
}
`,
			`Box struct<T extends record> {
	value T
	label string
	extra string
}
`,
			true,
			[]string{
				"func autoMigrateBox[T record]",
				"Box[T]",
				"Value: old.Value",
				"Label: old.Label",
			},
			nil,
		),
		Entry("emits nothing when the incoming version aliases everything",
			"channel",
			`Channel struct {
	key uuid @key
	name string

	@go marshal
	@go migrate
}
`,
			"Channel = v0.Channel\n",
			`Channel struct {
	key uuid @key
	name string
}
`,
			false, nil, nil,
		),
		Entry("converts a union variant by variant through a type switch",
			"board",
			`Base struct {
	key uuid

	@go marshal
}

View struct {
	type string
	args record = {}

	@go marshal
}

Tab union on variant extends Base {
	view View
	link {
		name string
	}

	@go marshal
}

Board struct {
	key uuid @key
	tabs Tab[]

	@go marshal
	@go migrate
}
`,
			`Base = v0.Base

View struct {
	type string

	@go marshal
}

Tab union on variant extends v0.Base {
	view View
	link {
		name string
	}

	@go marshal
}

Board struct {
	key uuid @key
	tabs Tab[]

	@go marshal
	@go migrate
}
`,
			`Base struct {
	key uuid
}

View struct {
	type string
}

Tab union on variant extends Base {
	view View
	link {
		name string
	}
}

Board struct {
	key uuid @key
	tabs Tab[]
}
`,
			true,
			[]string{
				"lo.MapErr(old.Tabs, func(v v0.Tab, _ int) (Tab, error)",
				"autoMigrateTab(ctx, v)",
				"func autoMigrateTab(ctx context.Context, old v0.Tab) (Tab, error)",
				"switch variant := old.Variant.(type)",
				"case nil:",
				"case v0.ViewTab:",
				"view, err := MigrateView(ctx, variant.View)",
				"return Tab{Variant: ViewTab{",
				"Base: variant.Base,",
				"View: view,",
				"case v0.LinkTab:",
				"return Tab{Variant: LinkTab{",
				"Name: variant.Name,",
				`errors.Newf("unknown Tab variant %T", variant)`,
				`"github.com/synnaxlabs/x/errors"`,
			},
			[]string{"Tab(old)"},
		),
		Entry("copies a field of a union the incoming version aliases",
			"board",
			`Link struct {
	name string

	@go marshal
}

Edge union on variant {
	link Link

	@go marshal
}

Board struct {
	key uuid @key
	edge Edge

	@go marshal
	@go migrate
}
`,
			`Link = v0.Link
Edge = v0.Edge

Board struct {
	key uuid @key
	name string
	edge Edge

	@go marshal
	@go migrate
}
`,
			`Link struct {
	name string
}

Edge union on variant {
	link Link
}

Board struct {
	key uuid @key
	name string
	edge Edge
}
`,
			true,
			[]string{"Edge: old.Edge"},
			nil,
		),
		Entry("leaves a field to the hand-written migration when an alias of a union "+
			"becomes a struct",
			"board",
			`Link struct {
	name string

	@go marshal
}

Edge union on variant {
	link Link

	@go marshal
}

Wire = Edge

Board struct {
	key uuid @key
	wire Wire

	@go marshal
	@go migrate
}
`,
			`Wire struct {
	name string

	@go marshal
}

Board struct {
	key uuid @key
	wire Wire

	@go marshal
	@go migrate
}
`,
			`Wire struct {
	name string
}

Board struct {
	key uuid @key
	wire Wire
}
`,
			true,
			[]string{"Key: old.Key"},
			[]string{"autoMigrateWire", "Wire: "},
		),
		Entry("pairs variant embeds by declaration when a union drops a base",
			"board",
			`Base struct {
	key uuid

	@go marshal
}

View struct {
	type string
	args record = {}

	@go marshal
}

Tab union on variant extends Base {
	view View

	@go marshal
}

Board struct {
	key uuid @key
	tab Tab

	@go marshal
	@go migrate
}
`,
			`View struct {
	type string

	@go marshal
}

Tab union on variant {
	view View

	@go marshal
}

Board struct {
	key uuid @key
	tab Tab

	@go marshal
	@go migrate
}
`,
			`View struct {
	type string
}

Tab union on variant {
	view View
}

Board struct {
	key uuid @key
	tab Tab
}
`,
			true,
			[]string{
				"view, err := MigrateView(ctx, variant.View)",
				"View: view,",
			},
			[]string{"Base: variant.Base"},
		),
		Entry("keeps a field variable off the type switch's bound name",
			"board",
			`View struct {
	type string
	args record = {}

	@go marshal
}

Tab union on kind {
	panel {
		variant View
	}

	@go marshal
}

Board struct {
	key uuid @key
	tab Tab

	@go marshal
	@go migrate
}
`,
			`View struct {
	type string

	@go marshal
}

Tab union on kind {
	panel {
		variant View
	}

	@go marshal
}

Board struct {
	key uuid @key
	tab Tab

	@go marshal
	@go migrate
}
`,
			`View struct {
	type string
}

Tab union on kind {
	panel {
		variant View
	}
}

Board struct {
	key uuid @key
	tab Tab
}
`,
			true,
			[]string{
				"variantVal, err := MigrateView(ctx, variant.Variant)",
				"Variant: variantVal,",
			},
			[]string{"variant, err :="},
		),
		Entry("leaves a variant the incoming version drops to the generated error",
			"board",
			`Tab union on variant {
	text {
		value string
	}
	link {
		name string
	}

	@go marshal
}

Board struct {
	key uuid @key
	tab Tab

	@go marshal
	@go migrate
}
`,
			`Tab union on variant {
	text {
		value string
	}

	@go marshal
}

Board struct {
	key uuid @key
	tab Tab

	@go marshal
	@go migrate
}
`,
			`Tab union on variant {
	text {
		value string
	}
}

Board struct {
	key uuid @key
	tab Tab
}
`,
			true,
			[]string{
				"func autoMigrateTab(_ context.Context, old v0.Tab) (Tab, error)",
				"case v0.TextTab:",
				"Value: variant.Value,",
			},
			[]string{"LinkTab"},
		),
		Entry("converts a recursive union through its own migration",
			"board",
			`Leaf struct {
	name string
	note string

	@go marshal
}

Node union on variant {
	leaf Leaf
	split {
		first Node
		last Node
	}

	@go marshal
}

Board struct {
	key uuid @key
	root Node

	@go marshal
	@go migrate
}
`,
			`Leaf struct {
	name string

	@go marshal
}

Node union on variant {
	leaf Leaf
	split {
		first Node
		last Node
	}

	@go marshal
}

Board struct {
	key uuid @key
	root Node

	@go marshal
	@go migrate
}
`,
			`Leaf struct {
	name string
}

Node union on variant {
	leaf Leaf
	split {
		first Node
		last Node
	}
}

Board struct {
	key uuid @key
	root Node
}
`,
			true,
			[]string{
				"root, err := autoMigrateNode(ctx, old.Root)",
				"case v0.SplitNode:",
				"first, err := autoMigrateNode(ctx, variant.First)",
				"last, err := autoMigrateNode(ctx, variant.Last)",
				"return Node{Variant: SplitNode{",
				"leaf, err := MigrateLeaf(ctx, variant.Leaf)",
			},
			[]string{"Node(old)"},
		),
		Entry(
			"migrates a named variant whose payload changed through the generated switch",
			"payload",
			`Text struct {
	value string

	@go marshal
}

Payload union on type {
	text Text

	@go marshal
}

Config struct {
	key uuid @key
	payloads Payload[]

	@go marshal
	@go migrate
}
`,
			`Text struct {
	value string
	size int32 = 0

	@go marshal
}

Payload union on type {
	text Text

	@go marshal
}

Config struct {
	key uuid @key
	payloads Payload[]

	@go marshal
	@go migrate
}
`,
			`Text struct {
	value string
	size int32 = 0
}

Payload union on type {
	text Text
}

Config struct {
	key uuid @key
	payloads Payload[]
}
`,
			true,
			[]string{"return autoMigratePayload(ctx, v)", "case v0.TextPayload:"},
			nil,
		),
		Entry("routes a union whose variants changed through its hand migration",
			"channel",
			`Payload union on type {
	text {
		value string
	}

	@go marshal
}
Channel struct {
	key uuid @key
	payloads Payload[]

	@go marshal
	@go migrate
}
`,
			`Payload union on type {
	binary {
		data bytes
	}

	@go marshal
}
Channel struct {
	key uuid @key
	payloads Payload[]

	@go marshal
	@go migrate
}
`,
			`Payload union on type {
	binary {
		data bytes
	}
}
Channel struct {
	key uuid @key
	payloads Payload[]
}
`,
			true,
			[]string{
				"func autoMigrateChannel",
				"return MigratePayload(ctx, v)",
				"func autoMigratePayload",
			},
			[]string{"Payload(old)"},
		),
		Entry("leaves a field whose union-ness changed to the hand migration",
			"channel",
			`Text struct {
	value string

	@go marshal
}
Binary struct {
	data bytes

	@go marshal
}
Payload struct {
	text Text

	@go marshal
}
Channel struct {
	key uuid @key
	name string
	payload Payload
	backups Payload[]

	@go marshal
	@go migrate
}
`,
			`Text struct {
	value string

	@go marshal
}
Binary struct {
	data bytes

	@go marshal
}
Payload union on type {
	text Text
	binary Binary

	@go marshal
}
Channel struct {
	key uuid @key
	name string
	payload Payload
	backups Payload[]

	@go marshal
	@go migrate
}
`,
			`Text struct {
	value string
}
Binary struct {
	data bytes
}
Payload union on type {
	text Text
	binary Binary
}
Channel struct {
	key uuid @key
	name string
	payload Payload
	backups Payload[]
}
`,
			true,
			[]string{"Name: old.Name"},
			[]string{"Payload:", "Backups"},
		),
		Entry("leaves a field retargeted to another declaration to the hand migration",
			"channel",
			`Details struct {
	a string

	@go marshal
}
Channel struct {
	key uuid @key
	name string
	details Details

	@go marshal
	@go migrate
}
`,
			`Info struct {
	a string

	@go marshal
}
Channel struct {
	key uuid @key
	name string
	details Info

	@go marshal
	@go migrate
}
`,
			`Info struct {
	a string
}
Channel struct {
	key uuid @key
	name string
	details Info
}
`,
			true,
			[]string{"Key: old.Key", "Name: old.Name"},
			[]string{"Details"},
		),
		Entry("migrates slice elements through the developer wrapper",
			"group",
			`Member struct {
	name string

	@go marshal
}
Group struct {
	key uuid @key
	members Member[]

	@go marshal
	@go migrate
}
`,
			`Member struct {
	name string
	role string

	@go marshal
}
Group struct {
	key uuid @key
	members Member[]

	@go marshal
	@go migrate
}
`,
			`Member struct {
	name string
	role string
}
Group struct {
	key uuid @key
	members Member[]
}
`,
			true,
			[]string{
				"lo.MapErr",
				"MigrateMember(ctx, v)",
				"Members: members",
				"func autoMigrateMember",
				"v0.Member",
			},
			nil,
		),
		Entry("migrates an alias-of-array element-wise",
			"member",
			`Member struct {
	name string

	@go marshal
}
Members = Member[]
`,
			`Member struct {
	name string
	role string

	@go marshal
}
Members = Member[]
`,
			`Member struct {
	name string
	role string
}
Members = Member[]
`,
			true,
			[]string{
				"func autoMigrateMembers",
				"lo.MapErr",
				"MigrateMember(ctx, v)",
				"func autoMigrateMember",
			},
			nil,
		),
		Entry("migrates a distinct-of-array element-wise",
			"keyset",
			`Member struct {
	name string

	@go marshal
}
Keys Member[] {
	@doc value "is a set of members."
}
`,
			`Member struct {
	name string
	role string

	@go marshal
}
Keys Member[] {
	@doc value "is a set of members."
}
`,
			`Member struct {
	name string
	role string
}
Keys Member[] {
	@doc value "is a set of members."
}
`,
			true,
			[]string{"func autoMigrateKeys", "lo.MapErr"},
			nil,
		),
		Entry(
			"migrates an alias of an array alias element-wise",
			"member",
			`Member struct {
	name string

	@go marshal
}
Members = Member[]
Crew = Members
`,
			`Member struct {
	name string
	role string

	@go marshal
}
Members = Member[]
Crew = Members
`,
			`Member struct {
	name string
	role string
}
Members = Member[]
Crew = Members
`,
			true,
			[]string{
				"func autoMigrateCrew",
				"lo.MapErr(old, func(v v0.Member, _ int) (Member, error) {",
			},
			[]string{"Crew(old)"},
		),
		Entry(
			"migrates a distinct over an array alias element-wise",
			"member",
			`Member struct {
	name string

	@go marshal
}
Members = Member[]
Keys Members
`,
			`Member struct {
	name string
	role string

	@go marshal
}
Members = Member[]
Keys Members
`,
			`Member struct {
	name string
	role string
}
Members = Member[]
Keys Members
`,
			true,
			[]string{
				"func autoMigrateKeys",
				"lo.MapErr(old, func(v v0.Member, _ int) (Member, error) {",
			},
			[]string{"Keys(old)"},
		),
		Entry(
			"migrates a field of an alias of a distinct array element-wise",
			"panel",
			`Member struct {
	name string

	@go marshal
}
Keys Member[]
Crew = Keys
Panel struct {
	key uuid @key
	crew Crew

	@go marshal
	@go migrate
}
`,
			`Member struct {
	name string
	role string

	@go marshal
}
Keys Member[]
Crew = Keys
Panel struct {
	key uuid @key
	crew Crew

	@go marshal
	@go migrate
}
`,
			`Member struct {
	name string
	role string
}
Keys Member[]
Crew = Keys
Panel struct {
	key uuid @key
	crew Crew
}
`,
			true,
			[]string{
				"crew, err := lo.MapErr(old.Crew, " +
					"func(v v0.Member, _ int) (Member, error) {",
			},
			[]string{"Crew(old)"},
		),
		Entry("migrates an alias of a struct alias through the target's fields",
			"member",
			`Member struct {
	name string

	@go marshal
}
Wrapper = Member
Outer = Wrapper
`,
			`Member struct {
	name string
	role string

	@go marshal
}
Wrapper = Member
Outer = Wrapper
`,
			`Member struct {
	name string
	role string
}
Wrapper = Member
Outer = Wrapper
`,
			true,
			[]string{"func autoMigrateOuter", "Name: old.Name"},
			[]string{"Outer(old)"},
		),
		Entry("migrates an alias of a generic alias with its type arguments",
			"pair",
			`Tag struct {
	name string

	@go marshal
}
Pair struct<A, B> {
	first A
	second B

	@go marshal
}
Wrapper<T> = Pair<Tag, T>
Outer = Wrapper<string>
`,
			`Tag struct {
	name string

	@go marshal
}
Pair struct<A, B> {
	first A
	second B
	third bool

	@go marshal
}
Wrapper<T> = Pair<Tag, T>
Outer = Wrapper<string>
`,
			`Tag struct {
	name string
}
Pair struct<A, B> {
	first A
	second B
	third bool
}
Wrapper<T> = Pair<Tag, T>
Outer = Wrapper<string>
`,
			true,
			[]string{
				"func autoMigrateOuter(" +
					"ctx context.Context, old v0.Outer) (Outer, error) {\n" +
					"\tfirst, err := autoMigrateTag(ctx, old.First)",
			},
			nil,
		),
		Entry(
			"migrates an alias of a union variant by variant",
			"panel",
			`Pipe struct {
	length float64

	@go marshal
}
Edge union on variant {
	pipe Pipe

	@go marshal
}
Link = Edge
Panel struct {
	key uuid @key
	link Link

	@go marshal
	@go migrate
}
`,
			`Pipe struct {
	length float64
	width float64

	@go marshal
}
Edge union on variant {
	pipe Pipe

	@go marshal
}
Link = Edge
Panel struct {
	key uuid @key
	link Link

	@go marshal
	@go migrate
}
`,
			`Pipe struct {
	length float64
	width float64
}
Edge union on variant {
	pipe Pipe
}
Link = Edge
Panel struct {
	key uuid @key
	link Link
}
`,
			true,
			[]string{
				"func autoMigrateLink",
				"case v0.PipeEdge:",
				"MigratePipe(ctx, variant.Pipe)",
			},
			[]string{"Link(old)"},
		),
		Entry("migrates an alias-of-struct through the target's fields",
			"wrapper",
			`Base struct {
	name string

	@go marshal
}
Wrapper = Base
`,
			`Base struct {
	name string
	extra string

	@go marshal
}
Wrapper = Base
`,
			`Base struct {
	name string
	extra string
}
Wrapper = Base
`,
			true,
			[]string{
				"func autoMigrateWrapper",
				"func autoMigrateBase",
				"Name: old.Name",
			},
			nil,
		),
		Entry("routes a renumbered enum field through its cast helper",
			"device",
			`State enum {
	idle = 0
	running = 1
}
Device struct {
	key uuid @key
	state State

	@go marshal
	@go migrate
}
`,
			`State enum {
	idle = 0
	running = 2
}
Device struct {
	key uuid @key
	state State

	@go marshal
	@go migrate
}
`,
			`State enum {
	idle = 0
	running = 2
}
Device struct {
	key uuid @key
	state State
}
`,
			true,
			[]string{
				"state, err := autoMigrateState(ctx, old.State)",
				"func autoMigrateState",
				"State: state",
				"(old), nil",
			},
			nil,
		),
		Entry("migrates a redeclared extends parent through its auto-copy",
			"node",
			`Base struct {
	name string

	@go marshal
}
Node struct extends Base {
	key uuid @key

	@go marshal
	@go migrate
}
`,
			`Base struct {
	name string

	@go marshal
}
Node struct extends Base {
	key uuid @key
	virtual bool

	@go marshal
	@go migrate
}
`,
			`Base struct {
	name string
}
Node struct extends Base {
	key uuid @key
	virtual bool
}
`,
			true,
			[]string{
				"base, err := autoMigrateBase(ctx, old.Base)",
				"func autoMigrateBase",
				"Key: old.Key",
			},
			nil,
		),
		Entry("leaves a parent's default override to the parent's copy",
			"reader",
			`Base struct {
	rate int32 = 10

	@go marshal
}
Reader struct extends Base {
	key uuid @key
	rate = 50

	@go marshal
	@go migrate
}
`,
			`Base struct {
	rate int32 = 10

	@go marshal
}
Reader struct extends Base {
	key uuid @key
	rate = 50
	enabled bool

	@go marshal
	@go migrate
}
`,
			`Base struct {
	rate int32 = 10
}
Reader struct extends Base {
	key uuid @key
	rate = 50
	enabled bool
}
`,
			true,
			[]string{"\t\tBase: base,\n\t\tKey: old.Key,\n\t}, nil"},
			nil,
		),
		Entry("migrates a changed extends parent through its wrapper",
			"shape",
			`Base struct {
	name string

	@go marshal
}
Shape struct extends Base {
	key uuid @key

	@go marshal
	@go migrate
}
`,
			`Base struct {
	name string
	extra string

	@go marshal
}
Shape struct extends Base {
	key uuid @key

	@go marshal
	@go migrate
}
`,
			`Base struct {
	name string
	extra string
}
Shape struct extends Base {
	key uuid @key
}
`,
			true,
			[]string{
				"base, err := MigrateBase(ctx, old.Base)",
				"Base: base",
				"func autoMigrateBase",
			},
			nil,
		),
		Entry("leaves a field whose optionality appeared to the hand migration",
			"log",
			`Log struct {
	key uuid @key
	name string
	desc string

	@go marshal
	@go migrate
}
`,
			`Log struct {
	key uuid @key
	name string
	desc string?

	@go marshal
	@go migrate
}
`,
			`Log struct {
	key uuid @key
	name string
	desc string?
}
`,
			true,
			[]string{"Name: old.Name"},
			[]string{"Desc"},
		),
		Entry("guards an optional struct field with a nil check",
			"task",
			`Details struct {
	a string

	@go marshal
}
Task struct {
	key uuid @key
	details Details?

	@go marshal
	@go migrate
}
`,
			`Details struct {
	a string
	b string

	@go marshal
}
Task struct {
	key uuid @key
	details Details?

	@go marshal
	@go migrate
}
`,
			`Details struct {
	a string
	b string
}
Task struct {
	key uuid @key
	details Details?
}
`,
			true,
			[]string{
				"var details *Details",
				"MigrateDetails(ctx, *old.Details)",
				"Details: details",
				"func autoMigrateDetails",
			},
			nil,
		),
		Entry("routes a nested migrate entry through its unexported wrapper",
			"rack",
			`Rack struct {
	name string

	@go marshal
	@go migrate
}
Controller struct {
	key uuid @key
	rack Rack

	@go marshal
	@go migrate
}
`,
			`Rack struct {
	name string
	extra string

	@go marshal
	@go migrate
}
Controller struct {
	key uuid @key
	rack Rack

	@go marshal
	@go migrate
}
`,
			`Rack struct {
	name string
	extra string
}
Controller struct {
	key uuid @key
	rack Rack
}
`,
			true,
			[]string{
				"rack, err := migrateRack(ctx, old.Rack)",
				"func autoMigrateRack",
			},
			nil,
		),
		Entry("substitutes defaulted type params before classification",
			"status",
			`Status struct<Data? = record> {
	msg string
	data Data?

	@go marshal
	@go migrate
}
`,
			`Status struct<Data? = record> {
	msg string
	cmd string
	data Data?

	@go marshal
	@go migrate
}
`,
			`Status struct<Data? = record> {
	msg string
	cmd string
	data Data?
}
`,
			true,
			[]string{"Msg: old.Msg", "Data: old.Data"},
			nil,
		),
	)
})

var _ = Describe("Plugin", func() {
	It("Should describe itself", func() {
		p := migrate.New()
		Expect(p.Name()).To(Equal("go/migrate"))
		Expect(p.Domains()).To(Equal([]string{"go"}))
		Expect(p.Requires()).To(Equal([]string{"go/types"}))
	})

	It("Should generate nothing without a versions resolver", func() {
		resp := MustSucceed(migrate.New().Generate(&plugin.Request{}))
		Expect(resp.Files).To(BeEmpty())
	})
})
