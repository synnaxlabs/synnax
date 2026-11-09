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
	"encoding/json/v2"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

type settings struct {
	Rate   int `json:"rate"`
	Stream int `json:"stream"`
}

type settingsUpdate struct {
	Rate   patch.Change[int] `json:"rate,omitzero"   msgpack:"rate,omitempty"`
	Stream patch.Change[int] `json:"stream,omitzero" msgpack:"stream,omitempty"`
}

type settingsField = patch.Struct[settingsUpdate, settings]

// holder has the shape of a generated update: a required and an optional scalar, a
// required and an optional struct, and the msgpack methods that carry its JSON form.
type holder struct {
	Name     patch.Change[string]                `json:"name,omitzero"     msgpack:"name,omitempty"`
	Group    patch.Change[string]                `json:"group,omitzero"    msgpack:"group,omitempty"`
	Count    patch.Change[int64]                 `json:"count,omitzero"    msgpack:"count,omitempty"`
	Settings settingsField                       `json:"settings,omitzero" msgpack:"settings,omitempty"`
	Extra    patch.Change[settingsField]         `json:"extra,omitzero"    msgpack:"extra,omitempty"`
	Tags     patch.ValueList[string]             `json:"tags,omitzero"     msgpack:"tags,omitempty"`
	Pairs    patch.List[settings, int, settings] `json:"pairs,omitzero"    msgpack:"pairs,omitempty"`
}

func (h holder) EncodeMsgpack(enc *msgpack.Encoder) error {
	return patch.EncodeMsgpack(enc, h)
}

func (h *holder) DecodeMsgpack(dec *msgpack.Decoder) error {
	return patch.DecodeMsgpack(dec, h)
}

var _ = Describe("Change", func() {
	It("Should make no change when zero", func() {
		var c patch.Change[string]
		Expect(c.IsZero()).To(BeTrue())
		Expect(c.Cleared()).To(BeFalse())
		v, ok := c.Get()
		Expect(ok).To(BeFalse())
		Expect(v).To(BeEmpty())
	})

	It("Should set the value that To gives, including a zero value", func() {
		c := patch.To("")
		Expect(c.IsZero()).To(BeFalse())
		Expect(c.Cleared()).To(BeFalse())
		Expect(MustBeOk(c.Get())).To(BeEmpty())
	})

	It("Should clear the field when cleared", func() {
		c := patch.Clear[string]()
		Expect(c.IsZero()).To(BeFalse())
		Expect(c.Cleared()).To(BeTrue())
		_, ok := c.Get()
		Expect(ok).To(BeFalse())
	})

	It("Should fail to encode a change that changes nothing", func() {
		Expect(json.MarshalWrite(io.Discard, patch.Change[string]{})).To(
			MatchError(
				ContainSubstring("a change that changes nothing has no wire form"),
			),
		)
	})

	It("Should round trip a set value through msgpack on its own", func() {
		var c patch.Change[string]
		b := MustSucceed(msgpack.Marshal(patch.To("Hot fire")))
		Expect(msgpack.Unmarshal(b, &c)).To(Succeed())
		Expect(c).To(Equal(patch.To("Hot fire")))
	})
})

var _ = Describe("Struct", func() {
	It("Should make no change when zero", func() {
		var s settingsField
		Expect(s.IsZero()).To(BeTrue())
		_, merges := s.Merged()
		_, replaces := s.Replaced()
		Expect(merges).To(BeFalse())
		Expect(replaces).To(BeFalse())
	})

	It("Should merge the partial that Merge gives", func() {
		s := patch.Merge[settings](settingsUpdate{Rate: patch.To(5)})
		Expect(MustBeOk(s.Merged())).To(Equal(settingsUpdate{Rate: patch.To(5)}))
		_, replaces := s.Replaced()
		Expect(replaces).To(BeFalse())
	})

	It("Should replace with the value that Replace gives", func() {
		s := patch.Replace[settingsUpdate](settings{Rate: 1})
		Expect(MustBeOk(s.Replaced())).To(Equal(settings{Rate: 1}))
		_, merges := s.Merged()
		Expect(merges).To(BeFalse())
	})

	DescribeTable("Should reject a JSON value that is not a merge or a replace",
		func(data, msg string) {
			var s settingsField
			Expect(json.Unmarshal([]byte(data), &s)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(msg)),
			))
		},
		Entry("null", `null`, "expected an object"),
		Entry("a number", `5`, "expected an object"),
		Entry(
			"set beside another member",
			`{"set":{"rate":1,"stream":2},"rate":3}`,
			`an object that holds "set" cannot hold other members`,
		),
	)
})

var _ = Describe("Update wire forms", func() {
	DescribeTable("Should encode, decode, and round trip through msgpack",
		func(h holder, data string) {
			Expect(json.Marshal(h)).To(MatchJSON(data))
			var decoded holder
			Expect(json.Unmarshal([]byte(data), &decoded)).To(Succeed())
			Expect(decoded).To(Equal(h))
			var viaMsgpack holder
			b := MustSucceed(msgpack.Marshal(h))
			Expect(msgpack.Unmarshal(b, &viaMsgpack)).To(Succeed())
			Expect(viaMsgpack).To(Equal(h))
		},
		Entry("no change", holder{}, `{}`),
		Entry("a set", holder{Name: patch.To("a")}, `{"name":"a"}`),
		Entry("a set to a zero value", holder{Name: patch.To("")}, `{"name":""}`),
		Entry("a clear", holder{Group: patch.Clear[string]()}, `{"group":null}`),
		Entry(
			"an integer that a float64 cannot hold",
			holder{Count: patch.To[int64](1<<62 + 1)},
			`{"count":4611686018427387905}`,
		),
		Entry(
			"a merge into a required struct",
			holder{Settings: patch.Merge[settings](settingsUpdate{Rate: patch.To(5)})},
			`{"settings":{"rate":5}}`,
		),
		Entry(
			"a replace of a required struct",
			holder{
				Settings: patch.Replace[settingsUpdate](settings{Rate: 1, Stream: 2}),
			},
			`{"settings":{"set":{"rate":1,"stream":2}}}`,
		),
		Entry(
			"a merge into an optional struct",
			holder{Extra: patch.To(patch.Merge[settings](settingsUpdate{
				Stream: patch.To(3),
			}))},
			`{"extra":{"stream":3}}`,
		),
		Entry(
			"a replace of an optional struct",
			holder{Extra: patch.To(patch.Replace[settingsUpdate](settings{Rate: 4}))},
			`{"extra":{"set":{"rate":4,"stream":0}}}`,
		),
		Entry(
			"a clear of an optional struct",
			holder{Extra: patch.Clear[settingsField]()},
			`{"extra":null}`,
		),
		Entry(
			"a set of a value list to empty",
			holder{Tags: patch.ValueList[string]{Set: []string{}}},
			`{"tags":{"set":[]}}`,
		),
		Entry(
			"keyed list operators",
			holder{Pairs: patch.List[settings, int, settings]{
				Remove: []int{1},
				Move:   []patch.ListMove[int]{{Key: 2}},
			}},
			`{"pairs":{"remove":[1],"move":[{"key":2}]}}`,
		),
	)
})
