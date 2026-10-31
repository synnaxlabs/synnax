// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package omit_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/resolution"
)

var _ = Describe("IsType", func() {
	It("should return true when omit expression exists for struct", func() {
		t := resolution.Type{
			Form: resolution.StructForm{},
			Domains: map[string]resolution.Domain{
				"ts": {Expressions: []resolution.Expression{{Name: "omit"}}},
			},
		}
		Expect(omit.IsType(t, "ts")).To(BeTrue())
	})

	It("should return false when domain missing", func() {
		t := resolution.Type{
			Form:    resolution.StructForm{},
			Domains: map[string]resolution.Domain{},
		}
		Expect(omit.IsType(t, "ts")).To(BeFalse())
	})

	It("should return false when omit not in expressions", func() {
		t := resolution.Type{
			Form: resolution.StructForm{},
			Domains: map[string]resolution.Domain{
				"ts": {Expressions: []resolution.Expression{{Name: "output"}}},
			},
		}
		Expect(omit.IsType(t, "ts")).To(BeFalse())
	})

	It("should check correct domain", func() {
		t := resolution.Type{
			Form: resolution.StructForm{},
			Domains: map[string]resolution.Domain{
				"py": {Expressions: []resolution.Expression{{Name: "omit"}}},
			},
		}
		Expect(omit.IsType(t, "ts")).To(BeFalse())
		Expect(omit.IsType(t, "py")).To(BeTrue())
	})

	It("should return true when omit expression exists for enum", func() {
		t := resolution.Type{
			Form: resolution.EnumForm{},
			Domains: map[string]resolution.Domain{
				"ts": {Expressions: []resolution.Expression{{Name: "omit"}}},
			},
		}
		Expect(omit.IsType(t, "ts")).To(BeTrue())
	})

	It("should return false when enum domain missing", func() {
		t := resolution.Type{
			Form:    resolution.EnumForm{},
			Domains: map[string]resolution.Domain{},
		}
		Expect(omit.IsType(t, "ts")).To(BeFalse())
	})

	It("should return false when omit not in enum expressions", func() {
		t := resolution.Type{
			Form: resolution.EnumForm{},
			Domains: map[string]resolution.Domain{
				"ts": {Expressions: []resolution.Expression{{Name: "output"}}},
			},
		}
		Expect(omit.IsType(t, "ts")).To(BeFalse())
	})
})

var _ = Describe("Fields", func() {
	field := func(name string, langs ...string) resolution.Field {
		f := resolution.Field{Name: name, Domains: map[string]resolution.Domain{}}
		for _, l := range langs {
			f.Domains[l] = resolution.Domain{
				Expressions: []resolution.Expression{{Name: "omit"}},
			}
		}
		return f
	}
	table := &resolution.Table{Types: []resolution.Type{{
		QualifiedName: "auth.Session",
		Form: resolution.StructForm{Fields: []resolution.Field{
			field("key"), field("hash", "ts", "py"), field("secret", "cpp"),
		}},
	}}}
	names := func(t *resolution.Table) []string {
		form := t.MustGet("auth.Session").Form.(resolution.StructForm)
		names := make([]string, len(form.Fields))
		for i, f := range form.Fields {
			names[i] = f.Name
		}
		return names
	}

	It("Should drop the fields omitted in any of the languages", func() {
		Expect(names(omit.Fields(table, "ts"))).To(Equal([]string{"key", "secret"}))
		Expect(names(omit.Fields(table, "cpp", "pb"))).
			To(Equal([]string{"key", "hash"}))
	})

	It("Should not modify the source table", func() {
		omit.Fields(table, "ts", "cpp")
		Expect(names(table)).To(Equal([]string{"key", "hash", "secret"}))
	})
})

var _ = Describe("IsField", func() {
	withDomains := func(domains map[string]resolution.Domain) resolution.Field {
		return resolution.Field{Name: "hash", Domains: domains}
	}

	It("Should return true when the field omits the domain", func() {
		f := withDomains(map[string]resolution.Domain{
			"ts": {Expressions: []resolution.Expression{{Name: "omit"}}},
		})
		Expect(omit.IsField(f, "ts")).To(BeTrue())
		Expect(omit.IsField(f, "py")).To(BeFalse())
	})

	It("Should return false when the domain is missing", func() {
		Expect(omit.IsField(withDomains(nil), "ts")).To(BeFalse())
	})

	It("Should return false for a non-omit expression", func() {
		f := withDomains(map[string]resolution.Domain{
			"go": {Expressions: []resolution.Expression{{
				Name:   "marshal",
				Values: []resolution.ExpressionValue{{IdentValue: "omit"}},
			}}},
		})
		Expect(omit.IsField(f, "go")).To(BeFalse())
	})
})

var _ = Describe("Fields inheritance and actions", func() {
	tsOmit := map[string]resolution.Domain{
		"ts": {Expressions: []resolution.Expression{{Name: "omit"}}},
	}
	table := &resolution.Table{Types: []resolution.Type{
		{
			Name:          "Parent",
			QualifiedName: "auth.Parent",
			Form: resolution.StructForm{Fields: []resolution.Field{
				{Name: "name"}, {Name: "secret", Domains: tsOmit},
			}},
		},
		{
			Name:          "Child",
			QualifiedName: "auth.Child",
			Form: resolution.StructForm{
				Extends: []resolution.TypeRef{{Name: "auth.Parent"}},
				Fields:  []resolution.Field{{Name: "secret", Optional: true}},
				Actions: []resolution.Action{{
					Name: "Apply",
					Fields: []resolution.Field{
						{Name: "name"},
						{Name: "secret", Domains: tsOmit},
					},
				}},
			},
		},
	}}

	It("Should drop an override of a field its parent omits", func() {
		stripped := omit.Fields(table, "ts")
		child := stripped.MustGet("auth.Child")
		Expect(resolution.UnifiedFields(child, stripped)).To(HaveLen(1))
		Expect(resolution.UnifiedFields(child, stripped)[0].Name).To(Equal("name"))
	})

	It("Should drop omitted fields from action payloads", func() {
		child := omit.Fields(table, "ts").MustGet("auth.Child")
		form := child.Form.(resolution.StructForm)
		Expect(form.Actions[0].Fields).To(HaveLen(1))
		Expect(form.Actions[0].Fields[0].Name).To(Equal("name"))
	})
})

var _ = Describe("IsAbsent", func() {
	output := resolution.Expression{
		Name:   "output",
		Values: []resolution.ExpressionValue{{StringValue: "ts/out"}},
	}
	DescribeTable("should report whether the struct exists in the language",
		func(t resolution.Type, expected bool) {
			Expect(omit.IsAbsent(t, "ts")).To(Equal(expected))
		},
		Entry("with an output", resolution.Type{
			Domains: map[string]resolution.Domain{
				"ts": {Expressions: []resolution.Expression{output}},
			},
		}, false),
		Entry("hand-written at an output", resolution.Type{
			Domains: map[string]resolution.Domain{
				"ts": {Expressions: []resolution.Expression{output, {Name: "hand"}}},
			},
		}, false),
		Entry("in a version file", resolution.Type{
			FilePath: "schemas/x/versions/color/v0.oracle",
		}, false),
		Entry("omitted", resolution.Type{
			Domains: map[string]resolution.Domain{
				"ts": {Expressions: []resolution.Expression{output, {Name: "omit"}}},
			},
		}, true),
		Entry("with no output", resolution.Type{
			Domains: map[string]resolution.Domain{
				"go": {Expressions: []resolution.Expression{output}},
			},
		}, true),
	)
})
