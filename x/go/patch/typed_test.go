// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package patch_test

import (
	"fmt"
	"math"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// span and document carry the Patch methods that Oracle generates.

type span struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

func (sv *span) Patch(c patch.Cursor) error {
	switch c.Segment() {
	case "start":
		return patch.Value(&sv.Start, c.Next())
	case "end":
		return patch.Value(&sv.End, c.Next())
	default:
		return c.UnknownField()
	}
}

type document struct {
	Name   string  `json:"name"`
	Count  uint8   `json:"count"`
	Span   span    `json:"span"`
	Window *span   `json:"window,omitzero"`
	Note   *string `json:"note,omitzero"`
}

func (dv *document) Patch(c patch.Cursor) error {
	switch c.Segment() {
	case "name":
		return patch.Value(&dv.Name, c.Next())
	case "count":
		return patch.Value(&dv.Count, c.Next())
	case "span":
		if c = c.Next(); c.Done() {
			return patch.Value(&dv.Span, c)
		}
		return dv.Span.Patch(c)
	case "window":
		if c = c.Next(); c.Done() || dv.Window == nil {
			return patch.OptionalValue(&dv.Window, c)
		}
		next := *dv.Window
		if err := next.Patch(c); err != nil {
			return err
		}
		dv.Window = &next
		return nil
	case "note":
		return patch.OptionalValue(&dv.Note, c.Next())
	default:
		return c.UnknownField()
	}
}

type scalars struct {
	S   string  `json:"s"`
	B   bool    `json:"b"`
	F   float64 `json:"f"`
	F32 float32 `json:"f32"`
	I   int     `json:"i"`
	I8  int8    `json:"i8"`
	I16 int16   `json:"i16"`
	I32 int32   `json:"i32"`
	I64 int64   `json:"i64"`
	U   uint    `json:"u"`
	U8  uint8   `json:"u8"`
	U16 uint16  `json:"u16"`
	U32 uint32  `json:"u32"`
	U64 uint64  `json:"u64"`
}

func (sv *scalars) Patch(c patch.Cursor) error {
	switch c.Segment() {
	case "s":
		return patch.Value(&sv.S, c.Next())
	case "b":
		return patch.Value(&sv.B, c.Next())
	case "f":
		return patch.Value(&sv.F, c.Next())
	case "f32":
		return patch.Value(&sv.F32, c.Next())
	case "i":
		return patch.Value(&sv.I, c.Next())
	case "i8":
		return patch.Value(&sv.I8, c.Next())
	case "i16":
		return patch.Value(&sv.I16, c.Next())
	case "i32":
		return patch.Value(&sv.I32, c.Next())
	case "i64":
		return patch.Value(&sv.I64, c.Next())
	case "u":
		return patch.Value(&sv.U, c.Next())
	case "u8":
		return patch.Value(&sv.U8, c.Next())
	case "u16":
		return patch.Value(&sv.U16, c.Next())
	case "u32":
		return patch.Value(&sv.U32, c.Next())
	case "u64":
		return patch.Value(&sv.U64, c.Next())
	default:
		return c.UnknownField()
	}
}

type patcher[D any] interface {
	*D
	Patch(c patch.Cursor) error
}

func apply[D any, P patcher[D]](doc P, a patch.Action) error {
	c, err := patch.Begin(a)
	if err != nil {
		return err
	}
	return doc.Patch(c)
}

// expectLikeEdit expects the typed apply to give the result that patch.Edit gives on
// the JSON form of doc, and to fail where patch.Edit fails.
func expectLikeEdit[D any, P patcher[D]](doc D, a patch.Action) {
	GinkgoHelper()
	want, wantErr := patch.Edit(doc, a.Apply)
	got := doc
	err := apply(P(&got), a)
	if wantErr != nil {
		Expect(err).To(MatchError(validate.ErrValidation))
		return
	}
	Expect(err).ToNot(HaveOccurred())
	Expect(got).To(Equal(want))
}

// gain has a field with a schema default and a default group, as Oracle generates them.
type gain struct {
	Factor float64 `json:"factor"`
	Lower  float64 `json:"lower"`
	Upper  float64 `json:"upper"`
}

func (g *gain) SetUngroupedDefaults() { g.Factor = 1 }

func (g *gain) DefaultGroups() {
	if g.Lower == 0 && g.Upper == 0 {
		g.Upper = 1
	}
}

func (gv *gain) Patch(c patch.Cursor) error {
	switch c.Segment() {
	case "factor":
		return patch.Defaulted(&gv.Factor, c.Next(), func(def *gain) {
			gv.Factor = def.Factor
		})
	case "upper":
		return patch.Defaulted(&gv.Upper, c.Next(), func(def *gain) {
			gv.Upper = def.Upper
		})
	default:
		return c.UnknownField()
	}
}

var _ = Describe("Patch", func() {
	var base document
	BeforeEach(func() {
		base = document{Name: "Hot fire", Count: 3, Span: span{Start: 1, End: 5}}
	})
	set := func(value any, path ...string) patch.Set {
		return patch.Set{Path: path, Value: value}
	}

	Describe("Agreement with the JSON form", func() {
		window := func(d document) document {
			d.Window = &span{Start: 2, End: 4}
			return d
		}
		DescribeTable("Should give the result of an edit of the JSON form",
			func(withWindow bool, a patch.Action) {
				doc := base
				if withWindow {
					doc = window(doc)
				}
				expectLikeEdit(doc, a)
			},
			Entry("a set of a field", false, set("Cold flow", "name")),
			Entry("a set of a field to null", false, set(nil, "name")),
			Entry("a set of a field to the wrong type", false, set(5.0, "name")),
			Entry("a set into a field", false, set("x", "name", "y")),
			Entry("a set of an integer", false, set(255.0, "count")),
			Entry("a set of an integer that overflows", false, set(256.0, "count")),
			Entry("a set of a struct", false, set(map[string]any{
				"start": "7", "end": 9.0,
			}, "span")),
			Entry("a set of a struct that lacks a member", false, set(map[string]any{
				"start": 1.0,
			}, "span")),
			Entry("a set of a struct with an unknown member", false, set(map[string]any{
				"start": 1.0, "end": 2.0, "x": 3.0,
			}, "span")),
			Entry("a set into a struct", false, set(6.0, "span", "start")),
			Entry("a set of a struct to null", false, set(nil, "span")),
			Entry("a set of an absent struct", false, set(map[string]any{
				"start": 1.0, "end": 2.0,
			}, "window")),
			Entry("a set into an absent struct", false, set(3.0, "window", "start")),
			Entry("a set into an optional struct", true, set(3.0, "window", "start")),
			Entry("a set of an optional struct to null", true, set(nil, "window")),
			Entry("a set of an optional field", false, set("hi", "note")),
			Entry("a set into an optional field", false, set("hi", "note", "x")),
			Entry("a set of an empty path", false, set("x")),
			Entry("a set of a prototype", false, set("x", "__proto__")),
			Entry("a set of an undeclared field", false, set("x", "tags")),
			Entry("a removal of an optional struct", true, patch.Remove{Key: "window"}),
			Entry("a removal of an optional field", false, patch.Remove{Key: "note"}),
			Entry("a removal of a field", false, patch.Remove{Key: "name"}),
			Entry("a removal in a struct", false, patch.Remove{
				Path: []string{"span"}, Key: "start",
			}),
			Entry("a removal in an absent struct", false, patch.Remove{
				Path: []string{"window"}, Key: "start",
			}),
			Entry("a removal in an optional struct", true, patch.Remove{
				Path: []string{"window"}, Key: "start",
			}),
			Entry("an insert into a field", false, patch.Insert{
				Path: []string{"name"}, Value: map[string]any{"key": "a"},
			}),
			Entry("an insert into an undeclared list", false, patch.Insert{
				Path: []string{"tags"}, Value: map[string]any{"key": "a"},
			}),
			Entry("an insert into the document", false, patch.Insert{
				Value: map[string]any{"key": "a"},
			}),
			Entry("a move in a field", false, patch.Move{
				Path: []string{"name"}, Key: "a",
			}),
			Entry("a move in an absent struct", false, patch.Move{
				Path: []string{"window", "list"}, Key: "a",
			}),
		)

		DescribeTable("Should give the result of an edit of the JSON form for a scalar",
			func(field string, value any) {
				expectLikeEdit(scalars{}, set(value, field))
			},
			func() []TableEntry {
				values := []any{
					"x",
					"\xff",
					true,
					0.0,
					math.Copysign(0, -1),
					1.0,
					1.5,
					-1.0,
					127.0,
					128.0,
					255.0,
					256.0,
					32767.0,
					65535.0,
					65536.0,
					2147483647.0,
					2147483648.0,
					4294967295.0,
					4294967296.0,
					float64(1 << 53),
					float64(
						1<<53 + 2,
					),
					float64(1 << 63),
					float64(1 << 64),
					1e21,
					math.NaN(),
					math.Inf(1),
					5,
					"5",
					nil,
				}
				fields := []string{
					"s", "b", "f", "f32", "i", "i8", "i16", "i32", "i64", "u", "u8",
					"u16", "u32", "u64",
				}
				var entries []TableEntry
				for _, f := range fields {
					for _, v := range values {
						entries = append(
							entries,
							Entry(fmt.Sprintf("%s %#v", f, v), f, v),
						)
					}
				}
				return entries
			}(),
		)
	})

	Describe("Stricter rules", func() {
		DescribeTable("Should reject an action that an edit of the JSON form ignores",
			func(a patch.Action, msg string) {
				doc := base
				Expect(apply(&doc, a)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(msg)),
				))
				Expect(doc).To(Equal(base))
			},
			Entry(
				"a removal of an undeclared field",
				patch.Remove{Key: "tags"},
				"[tags]: the path names no editable field",
			),
			Entry(
				"a move in an undeclared list",
				patch.Move{Path: []string{"tags"}, Key: "a"},
				"[tags]: the path names no editable field",
			),
			Entry(
				"an action with a guard",
				patch.Set{
					Path:   []string{"name"},
					Value:  "Cold flow",
					Guards: []patch.Guard{{Path: []string{"name"}, Value: "Hot fire"}},
				},
				"[name]: the guard names no union that the path goes into",
			),
		)
	})

	Describe("Errors", func() {
		DescribeTable("Should name the path of the failure",
			func(a patch.Action, msg string) {
				doc := base
				Expect(apply(&doc, a)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(msg)),
				))
			},
			Entry(
				"a value of the wrong type",
				set(5.0, "name"),
				"[name]: decoding value",
			),
			Entry(
				"a struct that lacks a member",
				set(map[string]any{"start": 1.0}, "span"),
				"[span, end]: the document requires a value",
			),
			Entry(
				"a removal of a field",
				patch.Remove{Path: []string{"span"}, Key: "end"},
				"[span, end]: the document requires a value",
			),
			Entry(
				"a set into a field",
				set("x", "name", "y"),
				"[name]: the value takes only a whole set",
			),
			Entry(
				"a set into an absent struct",
				set(3.0, "window", "start"),
				"[window]: the value is absent",
			),
			Entry(
				"an insert into a field",
				patch.Insert{Path: []string{"name"}, Value: map[string]any{"key": "a"}},
				"[name]: the value is not a list",
			),
		)

		It("Should leave the document unchanged", func() {
			doc := base
			Expect(apply(&doc, set(map[string]any{"start": 1.0}, "span"))).
				To(MatchError(validate.ErrValidation))
			Expect(doc).To(Equal(base))
		})
	})

	Describe("Ownership", func() {
		It("Should copy an optional struct before it changes", func() {
			original := &span{Start: 2, End: 4}
			base.Window = original
			doc := base
			Expect(apply(&doc, set(3.0, "window", "start"))).To(Succeed())
			Expect(doc.Window).To(Equal(&span{Start: 3, End: 4}))
			Expect(original).To(Equal(&span{Start: 2, End: 4}))
		})
	})

	Describe("Defaulted", func() {
		partial := func(doc *gain, a patch.Action) (bool, error) {
			c, err := patch.Begin(a)
			if err != nil {
				return false, err
			}
			var filled bool
			return filled, doc.Patch(c.Partial(&filled))
		}
		doc := func() gain { return gain{Factor: 3, Lower: 2, Upper: 5} }

		It("Should leave the schema default for a removal", func() {
			g := doc()
			Expect(partial(&g, patch.Remove{Key: "factor"})).To(BeTrue())
			Expect(g.Factor).To(Equal(1.0))
		})

		It("Should leave the schema default for a null set", func() {
			g := doc()
			Expect(partial(&g, patch.Set{Path: []string{"factor"}})).To(BeTrue())
			Expect(g.Factor).To(Equal(1.0))
		})

		It("Should keep an explicit zero", func() {
			g := doc()
			zero := patch.Set{Path: []string{"factor"}, Value: 0.0}
			Expect(partial(&g, zero)).To(BeFalse())
			Expect(g.Factor).To(BeZero())
		})

		It("Should leave the zero value in a default group", func() {
			g := doc()
			Expect(partial(&g, patch.Remove{Key: "upper"})).To(BeTrue())
			Expect(g.Upper).To(BeZero())
		})

		It("Should reject a path that goes into the field", func() {
			g := doc()
			into := patch.Set{Path: []string{"factor", "x"}, Value: 0.0}
			Expect(partial(&g, into)).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should reject a removal with an unchecked guard", func() {
			g := doc()
			guarded := patch.Remove{
				Key:    "factor",
				Guards: []patch.Guard{{Path: []string{"factor"}, Value: "a"}},
			}
			Expect(partial(&g, guarded)).Error().To(MatchError(ContainSubstring(
				"the guard names no union that the path goes into",
			)))
			Expect(g.Factor).To(Equal(3.0))
		})

		It("Should reject a removal from a strict document", func() {
			g := doc()
			c := MustSucceed(patch.Begin(patch.Remove{Key: "factor"}))
			Expect(g.Patch(c)).To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Allocations", func() {
		It("Should set a string without allocating", func() {
			doc := base
			a := set("Cold flow", "name")
			var err error
			Expect(testing.AllocsPerRun(100, func() {
				var c patch.Cursor
				if c, err = patch.Begin(a); err == nil {
					err = doc.Patch(c)
				}
			})).To(BeZero())
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Name).To(Equal("Cold flow"))
		})
	})
})
