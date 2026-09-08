// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schemadiff_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin/go/internal/schemadiff"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/testutil"
)

func analyzeSchema(source string) *resolution.Table {
	GinkgoHelper()
	table := resolution.NewTable()
	loader := testutil.NewMockFileLoader().
		Add("schemas/synnax/other", "Thing struct {\n\tvalue string\n}\n")
	diag := analyzer.AnalyzeSeeded(
		GinkgoT().Context(), source,
		"schemas/synnax/channel.oracle", "channel",
		loader, table,
	)
	Expect(diag.Ok()).To(BeTrue(), diag.String())
	return table
}

func typeIn(t *resolution.Table, name string) resolution.Type {
	GinkgoHelper()
	typ, ok := t.Get("channel." + name)
	Expect(ok).To(BeTrue(), name)
	return typ
}

var _ = DescribeTable("SchemasEqual",
	func(oldSrc, newSrc, name string, want bool) {
		a, b := analyzeSchema(oldSrc), analyzeSchema(newSrc)
		Expect(schemadiff.SchemasEqual(
			typeIn(a, name), typeIn(b, name), a, b,
		)).To(Equal(want))
	},
	Entry("identical structs with nested references",
		`Nested struct {
	value string
}
Channel struct {
	key uuid @key
	nested Nested
	tags string[]
	attrs map<string, string>
}
`,
		`Nested struct {
	value string
}
Channel struct {
	key uuid @key
	nested Nested
	tags string[]
	attrs map<string, string>
}
`,
		"Channel", true),
	Entry("struct replaced by an enum",
		"Channel struct {\n\tkey uuid @key\n}\n",
		"Channel enum {\n\ta = \"a\"\n}\n",
		"Channel", false),
	Entry("nested type changed",
		`Nested struct {
	value string
}
Channel struct {
	key uuid @key
	nested Nested
}
`,
		`Nested struct {
	value uint32
}
Channel struct {
	key uuid @key
	nested Nested
}
`,
		"Channel", false),
	Entry("optionality flipped",
		"Channel struct {\n\tkey uuid @key\n\tname string\n}\n",
		"Channel struct {\n\tkey uuid @key\n\tname string?\n}\n",
		"Channel", false),
	Entry("field marshal directive changed",
		"Channel struct {\n\tkey uuid @key\n\tname string\n}\n",
		"Channel struct {\n\tkey uuid @key\n\tname string {\n"+
			"\t\t@go marshal json_only\n\t}\n}\n",
		"Channel", false),
	Entry("omitted fields ignored on both sides",
		"Channel struct {\n\tkey uuid @key\n}\n",
		"Channel struct {\n\tkey uuid @key\n\ttmp string {\n"+
			"\t\t@go marshal omit\n\t}\n}\n",
		"Channel", true),
	Entry("extends list length changed",
		`Base struct {
	name string
}
Channel struct extends Base {
	key uuid @key
}
`,
		"Channel struct {\n\tkey uuid @key\n}\n",
		"Channel", false),
	Entry("extended parent changed",
		`Base struct {
	name string
}
Channel struct extends Base {
	key uuid @key
}
`,
		`Base struct {
	name uint32
}
Channel struct extends Base {
	key uuid @key
}
`,
		"Channel", false),
	Entry("int enum renumbered",
		"State enum {\n\tidle = 0\n\trunning = 1\n}\n",
		"State enum {\n\tidle = 0\n\trunning = 2\n}\n",
		"State", false),
	Entry("int enum member added",
		"State enum {\n\tidle = 0\n}\n",
		"State enum {\n\tidle = 0\n\trunning = 1\n}\n",
		"State", true),
	Entry("string enum member renamed",
		"State enum {\n\tidle = \"idle\"\n}\n",
		"State enum {\n\tstopped = \"stopped\"\n}\n",
		"State", true),
	Entry("int enum replaced by string enum",
		"State enum {\n\tidle = 0\n}\n",
		"State enum {\n\tidle = \"idle\"\n}\n",
		"State", false),
	Entry("alias retargeted",
		"Key = uuid\n", "Key = string\n", "Key", false),
	Entry("alias unchanged",
		"Key = uuid\n", "Key = uuid\n", "Key", true),
	Entry("distinct base changed",
		"Key uint64 {\n\t@go marshal flex\n}\n",
		"Key uint32 {\n\t@go marshal flex\n}\n",
		"Key", false),
	Entry("distinct base unchanged",
		"Key uint64 {\n\t@go marshal flex\n}\n",
		"Key uint64 {\n\t@go marshal flex\n}\n",
		"Key", true),
	Entry("identical unions",
		unionSrc("text", "type"), unionSrc("text", "type"),
		"Payload", true),
	Entry("union variant renamed",
		unionSrc("text", "type"), unionSrc("words", "type"),
		"Payload", false),
	Entry("union discriminator changed",
		unionSrc("text", "type"), unionSrc("text", "variant"),
		"Payload", false),
	Entry("union replaced by struct",
		unionSrc("text", "type"),
		`Text struct {
	value string
}
Binary struct {
	data bytes
}
Payload struct {
	text Text
}
`,
		"Payload", false),
	Entry("union variant added",
		unionSrc("text", "type"),
		`Text struct {
	value string
}
Binary struct {
	data bytes
}
Payload union on type {
	text Text
	binary Binary
	extra Text
}
`,
		"Payload", false),
	Entry("fixed array size changed",
		"Channel struct {\n\tkey uuid @key\n\tvec float32[3]\n}\n",
		"Channel struct {\n\tkey uuid @key\n\tvec float32[4]\n}\n",
		"Channel", false),
	Entry("fixed array replaced by slice",
		"Channel struct {\n\tkey uuid @key\n\tvec float32[3]\n}\n",
		"Channel struct {\n\tkey uuid @key\n\tvec float32[]\n}\n",
		"Channel", false),
	Entry("recursive struct compares without cycling",
		"Node struct {\n\tname string\n\tchildren Node[]\n}\n",
		"Node struct {\n\tname string\n\tchildren Node[]\n}\n",
		"Node", true),
	Entry("imported references resolved through their table",
		"import \"schemas/synnax/other\"\n\n"+
			"Channel struct {\n\tkey uuid @key\n\tthing other.Thing\n}\n",
		"import \"schemas/synnax/other\"\n\n"+
			"Channel struct {\n\tkey uuid @key\n\tthing other.Thing\n}\n",
		"Channel", true),
)

func unionSrc(variant, discriminator string) string {
	return `Text struct {
	value string
}
Binary struct {
	data bytes
}
Payload union on ` + discriminator + ` {
	` + variant + ` Text
	binary Binary
}
`
}

var _ = DescribeTable("SchemaDiff",
	func(oldSrc, newSrc, entry string, want map[string]schemadiff.TypeChangeKind) {
		a, b := analyzeSchema(oldSrc), analyzeSchema(newSrc)
		result := schemadiff.SchemaDiff(
			typeIn(a, entry), typeIn(b, entry), a, b,
			func(s string) string { return s },
		)
		got := make(map[string]schemadiff.TypeChangeKind, len(result))
		for name, d := range result {
			Expect(d.QualifiedName).To(Equal(name))
			got[name] = d.Kind
		}
		Expect(got).To(Equal(want))
	},
	Entry("records a direct field addition",
		"Channel struct {\n\tkey uuid @key\n}\n",
		"Channel struct {\n\tkey uuid @key\n\tname string\n}\n",
		"Channel",
		map[string]schemadiff.TypeChangeKind{
			"channel.Channel": schemadiff.TypeChanged,
		}),
	Entry("returns empty for unchanged types",
		"Channel struct {\n\tkey uuid @key\n\tname string\n}\n",
		"Channel struct {\n\tkey uuid @key\n\tname string\n}\n",
		"Channel",
		map[string]schemadiff.TypeChangeKind{}),
	Entry("marks parents of changed descendants",
		`Nested struct {
	value string
}
Channel struct {
	key uuid @key
	nested Nested
}
`,
		`Nested struct {
	value string
	extra string
}
Channel struct {
	key uuid @key
	nested Nested
}
`,
		"Channel",
		map[string]schemadiff.TypeChangeKind{
			"channel.Channel": schemadiff.TypeDescendantChanged,
			"channel.Nested":  schemadiff.TypeChanged,
		}),
	Entry("walks through aliases",
		`Nested struct {
	value string
}
Alias = Nested
Channel struct {
	key uuid @key
	nested Alias
}
`,
		`Nested struct {
	value string
	extra string
}
Alias = Nested
Channel struct {
	key uuid @key
	nested Alias
}
`,
		"Channel",
		map[string]schemadiff.TypeChangeKind{
			"channel.Channel": schemadiff.TypeDescendantChanged,
			"channel.Alias":   schemadiff.TypeDescendantChanged,
			"channel.Nested":  schemadiff.TypeChanged,
		}),
	Entry("walks through distinct array bases",
		`Nested struct {
	value string
}
Keys Nested[] {
	@doc value "is a set of nested keys."
}
`,
		`Nested struct {
	value string
	extra string
}
Keys Nested[] {
	@doc value "is a set of nested keys."
}
`,
		"Keys",
		map[string]schemadiff.TypeChangeKind{
			"channel.Keys":   schemadiff.TypeDescendantChanged,
			"channel.Nested": schemadiff.TypeChanged,
		}),
	Entry("reports non-struct entries through deep equality",
		"State enum {\n\tidle = 0\n\trunning = 1\n}\n",
		"State enum {\n\tidle = 0\n\trunning = 2\n}\n",
		"State",
		map[string]schemadiff.TypeChangeKind{
			"channel.State": schemadiff.TypeChanged,
		}),
	Entry("detects field order changes",
		"Channel struct {\n\tkey uuid @key\n\tname string\n}\n",
		"Channel struct {\n\tname string\n\tkey uuid @key\n}\n",
		"Channel",
		map[string]schemadiff.TypeChangeKind{
			"channel.Channel": schemadiff.TypeChanged,
		}),
	Entry("detects optionality changes",
		"Channel struct {\n\tkey uuid @key\n\tname string\n}\n",
		"Channel struct {\n\tkey uuid @key\n\tname string?\n}\n",
		"Channel",
		map[string]schemadiff.TypeChangeKind{
			"channel.Channel": schemadiff.TypeChanged,
		}),
	Entry("detects field marshal directive changes",
		"Channel struct {\n\tkey uuid @key\n\tname string\n}\n",
		"Channel struct {\n\tkey uuid @key\n\tname string {\n"+
			"\t\t@go marshal json_only\n\t}\n}\n",
		"Channel",
		map[string]schemadiff.TypeChangeKind{
			"channel.Channel": schemadiff.TypeChanged,
		}),
	Entry("detects a field retargeted to another declaration",
		`Foo struct {
	a string
}
Bar struct {
	a string
}
Channel struct {
	key uuid @key
	item Foo
}
`,
		`Foo struct {
	a string
}
Bar struct {
	a string
}
Channel struct {
	key uuid @key
	item Bar
}
`,
		"Channel",
		map[string]schemadiff.TypeChangeKind{
			"channel.Channel": schemadiff.TypeChanged,
		}),
	Entry("walks map value type arguments",
		`Nested struct {
	value string
}
Channel struct {
	key uuid @key
	entries map<string, Nested>
}
`,
		`Nested struct {
	value string
	extra string
}
Channel struct {
	key uuid @key
	entries map<string, Nested>
}
`,
		"Channel",
		map[string]schemadiff.TypeChangeKind{
			"channel.Channel": schemadiff.TypeDescendantChanged,
			"channel.Nested":  schemadiff.TypeChanged,
		}),
	Entry("walks extends references",
		`Base struct {
	name string
}
Channel struct extends Base {
	key uuid @key
}
`,
		`Base struct {
	name string
	extra string
}
Channel struct extends Base {
	key uuid @key
}
`,
		"Channel",
		map[string]schemadiff.TypeChangeKind{
			"channel.Channel": schemadiff.TypeDescendantChanged,
			"channel.Base":    schemadiff.TypeChanged,
		}),
	Entry("handles recursive types without cycling",
		"Node struct {\n\tname string\n\tchildren Node[]\n}\n",
		"Node struct {\n\tname string\n\tlabel string\n"+
			"\tchildren Node[]\n}\n",
		"Node",
		map[string]schemadiff.TypeChangeKind{
			"channel.Node": schemadiff.TypeChanged,
		}),
)

var _ = Describe("StructurallyEqual markers", func() {
	compare := func(oldSrc, newSrc, name string) bool {
		GinkgoHelper()
		a, b := analyzeSchema(oldSrc), analyzeSchema(newSrc)
		return schemadiff.StructurallyEqual(typeIn(a, name), typeIn(b, name), a, b)
	}

	It("Should distinguish declarations by type-level marshal marker", func() {
		Expect(compare(
			"Channel struct {\n\tkey uuid @key\n}\n",
			"Channel struct {\n\tkey uuid @key\n\n\t@go marshal\n}\n",
			"Channel",
		)).To(BeFalse())
	})

	It("Should distinguish declarations by type-level hand marker", func() {
		Expect(compare(
			"Channel struct {\n\tkey uuid @key\n\n\t@go marshal\n}\n",
			"Channel struct {\n\tkey uuid @key\n\n\t@go marshal hand\n}\n",
			"Channel",
		)).To(BeFalse())
	})

	It("Should distinguish an enum from a struct", func() {
		Expect(compare(
			"State enum {\n\tidle = 0\n}\n",
			"State struct {\n\tname string\n}\n",
			"State",
		)).To(BeFalse())
	})

	It("Should compare imported field type names without their namespace", func() {
		src := "import \"schemas/synnax/other\"\n\n" +
			"Channel struct {\n\tkey uuid @key\n\tthing other.Thing\n}\n"
		Expect(compare(src, src, "Channel")).To(BeTrue())
	})
})

var _ = Describe("PersistedFields", func() {
	It("Should drop fields carrying a marshal omit directive", func() {
		table := analyzeSchema(
			"Channel struct {\n\tkey uuid @key\n\ttmp string {\n" +
				"\t\t@go marshal omit\n\t}\n\tname string\n}\n",
		)
		form := typeIn(table, "Channel").Form.(resolution.StructForm)
		fields := schemadiff.PersistedFields(form.Fields)
		names := make([]string, len(fields))
		for i, f := range fields {
			names[i] = f.Name
		}
		Expect(names).To(Equal([]string{"key", "name"}))
	})
})
