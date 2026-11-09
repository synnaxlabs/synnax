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
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// entry, channel, its variants, and config carry the methods that Oracle generates.

type entry struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Span span   `json:"span"`
}

func (ev *entry) Patch(c patch.Cursor) error {
	switch c.Segment() {
	case "key":
		return patch.Value(&ev.Key, c.Next())
	case "name":
		return patch.Value(&ev.Name, c.Next())
	case "span":
		if c = c.Next(); c.Done() {
			return patch.Value(&ev.Span, c)
		}
		return ev.Span.Patch(c)
	default:
		return c.UnknownField()
	}
}

func (ev *entry) PatchKey() string { return ev.Key }

type channelVariant interface{ isChannelVariant() }

type analogChannel struct {
	Key   string  `json:"key"`
	Port  string  `json:"port"`
	Range float64 `json:"range"`
}

func (analogChannel) isChannelVariant() {}

func (ac *analogChannel) Patch(c patch.Cursor) error {
	switch c.Segment() {
	case "key":
		return patch.Value(&ac.Key, c.Next())
	case "port":
		return patch.Value(&ac.Port, c.Next())
	case "range":
		return patch.Value(&ac.Range, c.Next())
	default:
		return c.UnknownField()
	}
}

type digitalChannel struct {
	Key  string `json:"key"`
	Port string `json:"port"`
	Line int32  `json:"line"`
}

func (digitalChannel) isChannelVariant() {}

func (dc *digitalChannel) Patch(c patch.Cursor) error {
	switch c.Segment() {
	case "key":
		return patch.Value(&dc.Key, c.Next())
	case "port":
		return patch.Value(&dc.Port, c.Next())
	case "line":
		return patch.Value(&dc.Line, c.Next())
	default:
		return c.UnknownField()
	}
}

type counterChannel struct {
	Key  string `json:"key"`
	Port string `json:"port"`
	Line int32  `json:"line"`
}

func (counterChannel) isChannelVariant() {}

func (cc *counterChannel) Patch(c patch.Cursor) error {
	switch c.Segment() {
	case "key":
		return patch.Value(&cc.Key, c.Next())
	case "port":
		return patch.Value(&cc.Port, c.Next())
	case "line":
		return patch.Value(&cc.Line, c.Next())
	default:
		return c.UnknownField()
	}
}

type channel struct{ Variant channelVariant }

func (u channel) MarshalJSONTo(enc *jsontext.Encoder) error {
	switch v := u.Variant.(type) {
	case nil:
		return enc.WriteToken(jsontext.Null)
	case analogChannel:
		return json.MarshalEncode(enc, struct {
			Type string `json:"type"`
			analogChannel
		}{Type: "analog", analogChannel: v})
	case digitalChannel:
		return json.MarshalEncode(enc, struct {
			Type string `json:"type"`
			digitalChannel
		}{Type: "digital", digitalChannel: v})
	case counterChannel:
		return json.MarshalEncode(enc, struct {
			Type string `json:"type"`
			counterChannel
		}{Type: "counter", counterChannel: v})
	default:
		return errors.Newf("channel: unknown variant %T", v)
	}
}

func (u *channel) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	data, err := dec.ReadValue()
	if err != nil {
		return err
	}
	if data.Kind() == 'n' {
		u.Variant = nil
		return nil
	}
	opts := dec.Options()
	var disc struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(
		data, &disc, opts, json.RejectUnknownMembers(false),
	); err != nil {
		return err
	}
	switch disc.Type {
	case "analog":
		var v struct {
			Type string `json:"type"`
			analogChannel
		}
		if err := json.Unmarshal(data, &v, opts); err != nil {
			return err
		}
		u.Variant = v.analogChannel
	case "digital":
		var v struct {
			Type string `json:"type"`
			digitalChannel
		}
		if err := json.Unmarshal(data, &v, opts); err != nil {
			return err
		}
		u.Variant = v.digitalChannel
	case "counter":
		var v struct {
			Type string `json:"type"`
			counterChannel
		}
		if err := json.Unmarshal(data, &v, opts); err != nil {
			return err
		}
		u.Variant = v.counterChannel
	default:
		return errors.Wrapf(
			validate.ErrValidation,
			"channel: unknown type %q",
			disc.Type,
		)
	}
	return nil
}

func (u *channel) Patch(c patch.Cursor) error {
	switch v := u.Variant.(type) {
	case analogChannel:
		c, ok := c.Guard("type", "analog")
		if !ok {
			return nil
		}
		if c.Segment() == "type" {
			return patch.Discriminator(u, c)
		}
		if err := v.Patch(c); err != nil {
			return err
		}
		u.Variant = v
		return nil
	case digitalChannel:
		c, ok := c.Guard("type", "digital")
		if !ok {
			return nil
		}
		if c.Segment() == "type" {
			return patch.Discriminator(u, c)
		}
		if err := v.Patch(c); err != nil {
			return err
		}
		u.Variant = v
		return nil
	case counterChannel:
		c, ok := c.Guard("type", "counter")
		if !ok {
			return nil
		}
		if c.Segment() == "type" {
			return patch.Discriminator(u, c)
		}
		if err := v.Patch(c); err != nil {
			return err
		}
		u.Variant = v
		return nil
	default:
		return patch.Absent(u, c, "type")
	}
}

func (u *channel) PatchKey() string {
	switch v := u.Variant.(type) {
	case analogChannel:
		return v.Key
	case digitalChannel:
		return v.Key
	case counterChannel:
		return v.Key
	}
	return ""
}

type config struct {
	Entries  []entry   `json:"entries"`
	Channels []channel `json:"channels"`
	Scale    channel   `json:"scale"`
	Payload  any       `json:"payload"`
	Fill     *any      `json:"fill,omitzero"`
}

func (cv *config) Patch(c patch.Cursor) error {
	switch c.Segment() {
	case "entries":
		return patch.Entries(&cv.Entries, c.Next())
	case "channels":
		return patch.Entries(&cv.Channels, c.Next())
	case "scale":
		if c = c.Next(); c.Done() {
			return patch.Value(&cv.Scale, c)
		}
		return cv.Scale.Patch(c)
	case "payload":
		return patch.Any(&cv.Payload, c.Next())
	case "fill":
		return patch.OptionalAny(&cv.Fill, c.Next())
	default:
		return c.UnknownField()
	}
}

var _ = Describe("Patch of lists, unions, and JSON values", func() {
	var base config
	BeforeEach(func() {
		base = config{
			Entries: []entry{
				{Key: "a", Name: "A", Span: span{Start: 1, End: 2}},
				{Key: "b", Name: "B", Span: span{Start: 3, End: 4}},
			},
			Channels: []channel{
				{Variant: analogChannel{Key: "c1", Port: "AIN0", Range: 10}},
				{Variant: digitalChannel{Key: "c2", Port: "DIO0", Line: 1}},
			},
			Payload: map[string]any{
				"a":    map[string]any{"b": 1.0},
				"list": []any{map[string]any{"key": "x", "v": 1.0}},
			},
		}
	})
	set := func(value any, path ...string) patch.Set {
		return patch.Set{Path: path, Value: value}
	}
	guarded := func(value any, guard []string, tag string, path ...string) patch.Set {
		return patch.Set{
			Path:   path,
			Value:  value,
			Guards: []patch.Guard{{Path: guard, Value: tag}},
		}
	}
	after := func(key string) *string { return &key }

	DescribeTable(
		"Should give the result of an edit of the JSON form",
		func(a patch.Action) { expectLikeEdit(base, a) },
		Entry("a set into an entry", set("Z", "entries", "b", "name")),
		Entry("a set deep into an entry", set(9.0, "entries", "a", "span", "end")),
		Entry("a set into a missing entry", set("Z", "entries", "z", "name")),
		Entry("a set of an entry", set(map[string]any{
			"key": "a", "name": "Z", "span": map[string]any{"start": 5.0, "end": 6.0},
		}, "entries", "a")),
		Entry("a set of an entry under another key", set(map[string]any{
			"key": "b", "name": "Z", "span": map[string]any{"start": 5.0, "end": 6.0},
		}, "entries", "a")),
		Entry("a set of a missing entry", set(map[string]any{
			"key": "z", "name": "Z", "span": map[string]any{"start": 5.0, "end": 6.0},
		}, "entries", "z")),
		Entry("a set of the list", set([]any{map[string]any{
			"key": "z", "name": "Z", "span": map[string]any{"start": 5.0, "end": 6.0},
		}}, "entries")),
		Entry("a set of an undeclared entry field", set("x", "entries", "a", "tag")),
		Entry("a removal of an entry", patch.Remove{
			Path: []string{"entries"}, Key: "a",
		}),
		Entry("a removal of a missing entry", patch.Remove{
			Path: []string{"entries"}, Key: "z",
		}),
		Entry("an insert at the front", patch.Insert{
			Path: []string{"entries"},
			Value: map[string]any{
				"key":  "c",
				"name": "C",
				"span": map[string]any{"start": 0.0, "end": 0.0},
			},
		}),
		Entry("an insert after an entry", patch.Insert{
			Path: []string{"entries"},
			Value: map[string]any{
				"key":  "c",
				"name": "C",
				"span": map[string]any{"start": 0.0, "end": 0.0},
			},
			After: after("a"),
		}),
		Entry("an insert after a missing entry", patch.Insert{
			Path: []string{"entries"},
			Value: map[string]any{
				"key":  "c",
				"name": "C",
				"span": map[string]any{"start": 0.0, "end": 0.0},
			},
			After: after("z"),
		}),
		Entry("an insert of a key that exists", patch.Insert{
			Path: []string{"entries"},
			Value: map[string]any{
				"key":  "a",
				"name": "C",
				"span": map[string]any{"start": 0.0, "end": 0.0},
			},
		}),
		Entry("an insert without a key", patch.Insert{
			Path:  []string{"entries"},
			Value: map[string]any{"name": "C"},
		}),
		Entry("an insert into an entry", patch.Insert{
			Path:  []string{"entries", "a"},
			Value: map[string]any{"key": "c"},
		}),
		Entry("a move to the front", patch.Move{Path: []string{"entries"}, Key: "b"}),
		Entry("a move after an entry", patch.Move{
			Path: []string{"entries"}, Key: "a", After: after("b"),
		}),
		Entry("a move after a missing entry", patch.Move{
			Path: []string{"entries"}, Key: "a", After: after("z"),
		}),
		Entry("a move after itself", patch.Move{
			Path: []string{"entries"}, Key: "a", After: after("a"),
		}),
		Entry("a move of a missing entry", patch.Move{
			Path: []string{"entries"}, Key: "z",
		}),
		Entry("a set into a variant", set(5.0, "channels", "c1", "range")),
		Entry(
			"a set of a field of another variant",
			set(5.0, "channels", "c1", "line"),
		),
		Entry("a set under a guard that holds", guarded(
			5.0,
			[]string{"channels", "c1", "type"},
			"analog",
			"channels",
			"c1",
			"range",
		)),
		Entry("a set under a guard that fails", guarded(
			5.0,
			[]string{"channels", "c1", "type"},
			"digital",
			"channels",
			"c1",
			"range",
		)),
		Entry("a set under a guard on a missing entry", guarded(
			5.0, []string{"channels", "z", "type"}, "analog", "channels", "z", "range",
		)),
		Entry("a set of the discriminator to a variant with the same members", set(
			"counter", "channels", "c2", "type",
		)),
		Entry("a set of the discriminator to a variant with other members", set(
			"digital", "channels", "c1", "type",
		)),
		Entry("a set of the discriminator to an unknown variant", set(
			"bogus", "channels", "c2", "type",
		)),
		Entry("a removal of the discriminator", patch.Remove{
			Path: []string{"channels", "c1"}, Key: "type",
		}),
		Entry("a set of a union entry", set(map[string]any{
			"key": "c1", "type": "digital", "port": "DIO1", "line": 2.0,
		}, "channels", "c1")),
		Entry("a set into an absent union", set(1.0, "scale", "range")),
		Entry("a set of the discriminator of an absent union", set(
			"analog", "scale", "type",
		)),
		Entry("a set under a guard on an absent union", guarded(
			1.0, []string{"scale", "type"}, "analog", "scale", "range",
		)),
		Entry("a set of an absent union", set(map[string]any{
			"key": "s", "type": "analog", "port": "AIN1", "range": 5.0,
		}, "scale")),
		Entry("a set of a JSON value", set(map[string]any{"z": 1.0}, "payload")),
		Entry("a set of a JSON value to null", set(nil, "payload")),
		Entry("a set into a JSON value", set(2.0, "payload", "a", "b")),
		Entry("a set that adds objects to a JSON value", set(2.0, "payload", "n", "m")),
		Entry("a set into a JSON list entry", set(2.0, "payload", "list", "x", "v")),
		Entry("a removal in a JSON value", patch.Remove{
			Path: []string{"payload", "a"}, Key: "b",
		}),
		Entry("an insert into a JSON list", patch.Insert{
			Path:  []string{"payload", "list"},
			Value: map[string]any{"key": "y"},
			After: after("x"),
		}),
		Entry("a move in a JSON list", patch.Move{
			Path: []string{"payload", "list"}, Key: "x",
		}),
		Entry("a set of an optional JSON value", set("v", "fill")),
		Entry("a set into an absent optional JSON value", set("v", "fill", "a")),
		Entry("a removal of an optional JSON value", patch.Remove{Key: "fill"}),
	)

	Describe("Stricter rules", func() {
		DescribeTable("Should reject an action that an edit of the JSON form accepts",
			func(a patch.Action, msg string) {
				doc := base
				Expect(apply(&doc, a)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(msg)),
				))
			},
			Entry(
				"a set of an entry that lacks a member",
				set(map[string]any{"key": "a", "name": "Z"}, "entries", "a"),
				"[entries, a, span]: the document requires a value",
			),
			Entry(
				"a removal of a field of an entry",
				patch.Remove{Path: []string{"entries", "a"}, Key: "name"},
				"[entries, a, name]: the document requires a value",
			),
			Entry(
				"an insert that lacks a member",
				patch.Insert{
					Path:  []string{"entries"},
					Value: map[string]any{"key": "c", "name": "C"},
				},
				"[entries, span]: the document requires a value",
			),
			Entry(
				"a removal of a JSON value",
				patch.Remove{Key: "payload"},
				"[payload]: the document requires a value",
			),
			Entry(
				"a guard that names no union on the path",
				guarded(
					5.0,
					[]string{"entries", "a", "name"},
					"A",
					"entries",
					"a",
					"name",
				),
				"[entries, a, name]: the guard names no union that the path goes into",
			),
			Entry(
				"a guard on the union of an entry that a set replaces",
				guarded(
					map[string]any{"type": "digital", "key": "c1", "port": "DIO1"},
					[]string{"channels", "c1", "type"},
					"analog",
					"channels",
					"c1",
				),
				"[channels, c1, type]: the guard names no union that the path goes into",
			),
			Entry(
				"a guard on the union of an entry that a removal takes out",
				patch.Remove{
					Path: []string{"channels"},
					Key:  "c1",
					Guards: []patch.Guard{
						{Path: []string{"channels", "c1", "type"}, Value: "analog"},
					},
				},
				"[channels, c1, type]: the guard names no union that the path goes into",
			),
		)
	})

	Describe("Partial documents", func() {
		partial := func(doc *config, a patch.Action) (bool, error) {
			c, err := patch.Begin(a)
			if err != nil {
				return false, err
			}
			var filled bool
			return filled, doc.Patch(c.Partial(&filled))
		}

		It("Should leave a zero value for a removal of a required field", func() {
			doc := base
			Expect(MustSucceed(partial(&doc, patch.Remove{
				Path: []string{"entries", "a"}, Key: "name",
			}))).To(BeTrue())
			Expect(doc.Entries[0].Name).To(BeEmpty())
		})

		It("Should fill the members that a value lacks", func() {
			doc := base
			Expect(MustSucceed(partial(&doc, set(map[string]any{
				"key": "a", "name": "Z",
			}, "entries", "a")))).To(BeTrue())
			Expect(doc.Entries[0]).To(Equal(entry{Key: "a", Name: "Z"}))
		})

		It("Should fill the members that a new variant lacks", func() {
			doc := base
			Expect(MustSucceed(partial(&doc, set("analog", "scale", "type")))).
				To(BeTrue())
			Expect(doc.Scale).To(Equal(channel{Variant: analogChannel{}}))
		})

		It("Should not report a value that holds every member", func() {
			doc := base
			Expect(MustSucceed(partial(&doc, set(map[string]any{
				"key":  "a",
				"name": "Z",
				"span": map[string]any{"start": 5.0, "end": 6.0},
			}, "entries", "a")))).To(BeFalse())
		})

		It("Should still reject a member the document does not declare", func() {
			doc := base
			Expect(partial(&doc, set(map[string]any{
				"key": "a", "name": "Z", "x": 1.0,
			}, "entries", "a"))).Error().To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Ownership", func() {
		It("Should copy a list before it changes", func() {
			doc := base
			Expect(apply(&doc, set("Z", "entries", "a", "name"))).To(Succeed())
			Expect(apply(&doc, set(5.0, "channels", "c1", "range"))).To(Succeed())
			Expect(base.Entries[0].Name).To(Equal("A"))
			Expect(base.Channels[0].Variant).To(Equal(
				analogChannel{Key: "c1", Port: "AIN0", Range: 10},
			))
		})

		It("Should copy a JSON value before it changes", func() {
			doc := base
			Expect(apply(&doc, set(2.0, "payload", "a", "b"))).To(Succeed())
			Expect(base.Payload).To(HaveKeyWithValue("a", map[string]any{"b": 1.0}))
		})
	})

	Describe("Allocations", func() {
		allocs := func(a patch.Action) float64 {
			GinkgoHelper()
			var err error
			n := testing.AllocsPerRun(100, func() {
				doc := base
				var c patch.Cursor
				if c, err = patch.Begin(a); err == nil {
					err = doc.Patch(c)
				}
			})
			Expect(err).ToNot(HaveOccurred())
			return n
		}

		It("Should copy the list alone to set a field of an entry", func() {
			Expect(allocs(set("Z", "entries", "b", "name"))).To(Equal(1.0))
		})

		It("Should copy the list and the variant to set a field of a variant", func() {
			Expect(allocs(set(5.0, "channels", "c2", "line"))).To(Equal(2.0))
		})
	})
})
