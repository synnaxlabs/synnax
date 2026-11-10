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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

type portEntry struct {
	Key  string `json:"key"`
	Port int    `json:"port"`
}

type portUpdate struct {
	Key  string            `json:"key"`
	Port patch.Change[int] `json:"port,omitzero"`
}

type ports = patch.List[portEntry, string, portUpdate]

func portKey(p portEntry) string { return p.Key }

func portUpdateKey(u portUpdate) string { return u.Key }

func mergePort(p *portEntry, u portUpdate) error { return u.Port.ApplyTo(&p.Port) }

func mergeSettings(s *settings, u settingsUpdate) error {
	if err := u.Rate.ApplyTo(&s.Rate); err != nil {
		return err
	}
	return u.Stream.ApplyTo(&s.Stream)
}

// settingsDefaults gives stream a default of 7, and rate no default.
func settingsDefaults(u settingsUpdate) (settings, error) {
	if u.Rate.IsZero() {
		return settings{}, errors.Wrap(validate.ErrValidation, "rate is required")
	}
	return settings{Stream: 7}, nil
}

func applyPorts(base []portEntry, ops ports) ([]portEntry, error) {
	err := patch.ApplyList(&base, ops, portKey, portUpdateKey, mergePort)
	return base, err
}

var _ = Describe("Apply", func() {
	Describe("Change", func() {
		It("Should set, keep, or refuse to clear a required field", func() {
			v := 1
			Expect(patch.Change[int]{}.ApplyTo(&v)).To(Succeed())
			Expect(v).To(Equal(1))
			Expect(patch.To(0).ApplyTo(&v)).To(Succeed())
			Expect(v).To(BeZero())
			Expect(patch.Clear[int]().ApplyTo(&v)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("a required field cannot be cleared")),
			))
		})

		It("Should set, keep, or clear an optional field", func() {
			old := "a"
			v := &old
			patch.Change[string]{}.ApplyOptional(&v)
			Expect(v).To(Equal(&old))
			patch.To("b").ApplyOptional(&v)
			Expect(*v).To(Equal("b"))
			Expect(old).To(Equal("a"))
			patch.Clear[string]().ApplyOptional(&v)
			Expect(v).To(BeNil())
		})
	})

	Describe("Struct", func() {
		It("Should merge into or replace a required struct", func() {
			s := settings{Rate: 1, Stream: 2}
			merge := patch.Merge[settings](settingsUpdate{Rate: patch.To(5)})
			Expect(merge.ApplyTo(&s, mergeSettings)).To(Succeed())
			Expect(s).To(Equal(settings{Rate: 5, Stream: 2}))
			replace := patch.Replace[settingsUpdate](settings{Rate: 7})
			Expect(replace.ApplyTo(&s, mergeSettings)).To(Succeed())
			Expect(s).To(Equal(settings{Rate: 7}))
		})

		It("Should merge into a copy of an optional struct", func() {
			old := settings{Rate: 1, Stream: 2}
			s := &old
			Expect(patch.ApplyOptionalStruct(&s, patch.To(patch.Merge[settings](
				settingsUpdate{Stream: patch.To(3)},
			)), mergeSettings, settingsDefaults)).To(Succeed())
			Expect(*s).To(Equal(settings{Rate: 1, Stream: 3}))
			Expect(old).To(Equal(settings{Rate: 1, Stream: 2}))
		})

		It("Should replace or clear an optional struct", func() {
			var s *settings
			Expect(patch.ApplyOptionalStruct(
				&s,
				patch.To(patch.Replace[settingsUpdate](settings{Rate: 4})),
				mergeSettings,
				settingsDefaults,
			)).To(Succeed())
			Expect(*s).To(Equal(settings{Rate: 4}))
			Expect(patch.ApplyOptionalStruct(
				&s,
				patch.Change[settingsField]{},
				mergeSettings,
				settingsDefaults,
			)).To(Succeed())
			Expect(*s).To(Equal(settings{Rate: 4}))
			Expect(patch.ApplyOptionalStruct(
				&s,
				patch.Clear[settingsField](),
				mergeSettings,
				settingsDefaults,
			)).To(Succeed())
			Expect(s).To(BeNil())
		})

		It("Should merge into the defaults of an absent optional struct", func() {
			var s *settings
			Expect(patch.ApplyOptionalStruct(&s, patch.To(patch.Merge[settings](
				settingsUpdate{Rate: patch.To(1)},
			)), mergeSettings, settingsDefaults)).To(Succeed())
			Expect(*s).To(Equal(settings{Rate: 1, Stream: 7}))
		})

		It(
			"Should fail to merge into an absent optional struct without defaults",
			func() {
				var s *settings
				Expect(patch.ApplyOptionalStruct(&s, patch.To(patch.Merge[settings](
					settingsUpdate{Stream: patch.To(1)},
				)), mergeSettings, settingsDefaults)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("rate is required")),
				))
				Expect(s).To(BeNil())
			},
		)
	})

	Describe("ApplyList", func() {
		base := []portEntry{
			{Key: "a", Port: 1},
			{Key: "b", Port: 2},
			{Key: "c", Port: 3},
		}

		DescribeTable(
			"Should apply operators",
			func(ops ports, expected []portEntry) {
				Expect(applyPorts(base, ops)).To(Equal(expected))
			},
			Entry("none", ports{}, base),
			Entry(
				"a set",
				ports{Set: []portEntry{{Key: "z"}}},
				[]portEntry{{Key: "z"}},
			),
			Entry("a set to empty", ports{Set: []portEntry{}}, []portEntry{}),
			Entry(
				"a remove",
				ports{Remove: []string{"b"}},
				[]portEntry{{Key: "a", Port: 1}, {Key: "c", Port: 3}},
			),
			Entry(
				"an update",
				ports{Update: []portUpdate{{Key: "b", Port: patch.To(9)}}},
				[]portEntry{
					{Key: "a", Port: 1},
					{Key: "b", Port: 9},
					{Key: "c", Port: 3},
				},
			),
			Entry(
				"an insert at the end",
				ports{Insert: []portEntry{{Key: "d", Port: 4}}},
				[]portEntry{
					{Key: "a", Port: 1},
					{Key: "b", Port: 2},
					{Key: "c", Port: 3},
					{Key: "d", Port: 4},
				},
			),
			Entry(
				"a move to the start",
				ports{Move: []patch.ListMove[string]{{Key: "c"}}},
				[]portEntry{
					{Key: "c", Port: 3},
					{Key: "a", Port: 1},
					{Key: "b", Port: 2},
				},
			),
			Entry(
				"a move after null to the start",
				ports{Move: []patch.ListMove[string]{
					{Key: "c", After: patch.Clear[string]()},
				}},
				[]portEntry{
					{Key: "c", Port: 3},
					{Key: "a", Port: 1},
					{Key: "b", Port: 2},
				},
			),
			Entry(
				"a move after an entry",
				ports{Move: []patch.ListMove[string]{{Key: "a", After: patch.To("b")}}},
				[]portEntry{
					{Key: "b", Port: 2},
					{Key: "a", Port: 1},
					{Key: "c", Port: 3},
				},
			),
			Entry(
				"moves in the order given",
				ports{Move: []patch.ListMove[string]{
					{Key: "c"},
					{Key: "b", After: patch.To("c")},
				}},
				[]portEntry{
					{Key: "c", Port: 3},
					{Key: "b", Port: 2},
					{Key: "a", Port: 1},
				},
			),
			Entry(
				"a remove before an insert of the same key",
				ports{Remove: []string{"a"}, Insert: []portEntry{{Key: "a", Port: 8}}},
				[]portEntry{
					{Key: "b", Port: 2},
					{Key: "c", Port: 3},
					{Key: "a", Port: 8},
				},
			),
			Entry(
				"an insert before a move of the new entry",
				ports{
					Insert: []portEntry{{Key: "d", Port: 4}},
					Move:   []patch.ListMove[string]{{Key: "d"}},
				},
				[]portEntry{
					{Key: "d", Port: 4},
					{Key: "a", Port: 1},
					{Key: "b", Port: 2},
					{Key: "c", Port: 3},
				},
			),
		)

		DescribeTable("Should fail without a change to the list",
			func(ops ports, target error, msg string) {
				list := []portEntry{{Key: "a"}, {Key: "b"}}
				dst := list
				Expect(
					patch.ApplyList(&dst, ops, portKey, portUpdateKey, mergePort),
				).To(SatisfyAll(
					MatchError(target),
					MatchError(ContainSubstring(msg)),
				))
				Expect(dst).To(Equal(list))
				Expect(list).To(Equal([]portEntry{{Key: "a"}, {Key: "b"}}))
			},
			Entry(
				"a set beside another operator",
				ports{Set: []portEntry{}, Remove: []string{"a"}},
				validate.ErrValidation,
				"set cannot be combined with other operators",
			),
			Entry(
				"a set with a repeated key",
				ports{Set: []portEntry{{Key: "z"}, {Key: "z"}}},
				query.ErrUniqueViolation,
				"z already exists",
			),
			Entry(
				"a remove of a missing key",
				ports{Remove: []string{"b", "c9"}},
				patch.ErrStale,
				"c9 not found",
			),
			Entry(
				"an update of a missing key",
				ports{Update: []portUpdate{{Key: "c9", Port: patch.To(1)}}},
				patch.ErrStale,
				"c9 not found",
			),
			Entry(
				"a failed merge",
				ports{Update: []portUpdate{{Key: "a", Port: patch.Clear[int]()}}},
				validate.ErrValidation,
				"a: a required field cannot be cleared",
			),
			Entry(
				"an insert of an existing key",
				ports{Insert: []portEntry{{Key: "b"}}},
				query.ErrUniqueViolation,
				"b already exists",
			),
			Entry(
				"an insert of a key twice",
				ports{Insert: []portEntry{{Key: "z"}, {Key: "z"}}},
				query.ErrUniqueViolation,
				"z already exists",
			),
			Entry(
				"a move of a missing key",
				ports{Move: []patch.ListMove[string]{{Key: "c9"}}},
				patch.ErrStale,
				"c9 not found",
			),
			Entry(
				"a move after a missing key",
				ports{Move: []patch.ListMove[string]{{Key: "a", After: patch.To("c9")}}},
				patch.ErrStale,
				"c9 not found",
			),
			Entry(
				"a move after itself",
				ports{Move: []patch.ListMove[string]{{Key: "a", After: patch.To("a")}}},
				validate.ErrValidation,
				"a cannot move after itself",
			),
		)
	})

	Describe("ApplyValueList", func() {
		It("Should apply operators with each value as its key", func() {
			tags := []string{"a", "b", "c"}
			Expect(patch.ApplyValueList(&tags, patch.ValueList[string]{
				Remove: []string{"b"},
				Insert: []string{"d"},
				Move:   []patch.ListMove[string]{{Key: "d", After: patch.To("a")}},
			})).To(Succeed())
			Expect(tags).To(Equal([]string{"a", "d", "c"}))
		})

		It("Should fail for a value that is missing or exists", func() {
			tags := []string{"a"}
			Expect(patch.ApplyValueList(&tags, patch.ValueList[string]{
				Remove: []string{"z"},
			})).To(MatchError(patch.ErrStale))
			Expect(patch.ApplyValueList(&tags, patch.ValueList[string]{
				Insert: []string{"a"},
			})).To(MatchError(query.ErrUniqueViolation))
		})
	})

	Describe("Update.Check", func() {
		stored := map[string]any{
			"config": map[string]any{
				"type":     "ai",
				"channels": []any{map[string]any{"key": "c1", "type": "voltage"}},
			},
		}

		DescribeTable("Should pass when every guard holds",
			func(guards []patch.Guard) {
				Expect(patch.Update[settingsUpdate]{Guards: guards}.Check(stored)).
					To(Succeed())
			},
			Entry("no guards", nil),
			Entry("a field guard", []patch.Guard{
				{Path: []string{"config", "type"}, Value: "ai"},
			}),
			Entry("a guard through a keyed list", []patch.Guard{
				{Path: []string{"config", "channels", "c1", "type"}, Value: "voltage"},
			}),
		)

		It("Should check a typed record by its JSON form", func() {
			Expect(patch.Update[settingsUpdate]{Guards: []patch.Guard{
				{Path: []string{"key"}, Value: "a"},
			}}.Check(portEntry{Key: "a"})).To(Succeed())
		})

		DescribeTable("Should fail with ErrStale when a guard does not hold",
			func(guard patch.Guard, msg string) {
				Expect(patch.Update[settingsUpdate]{Guards: []patch.Guard{guard}}.
					Check(stored)).To(SatisfyAll(
					MatchError(patch.ErrStale),
					MatchError(ContainSubstring(msg)),
				))
			},
			Entry(
				"another value",
				patch.Guard{Path: []string{"config", "type"}, Value: "ao"},
				"[config, type] is not ao",
			),
			Entry(
				"a missing list entry",
				patch.Guard{Path: []string{"config", "channels", "c9", "type"}},
				"[config, channels, c9, type] is not ",
			),
		)
	})

	Describe("ErrStale", func() {
		It("Should decode an encoded error as ErrStale", func(ctx SpecContext) {
			encoded := errors.Encode(
				ctx,
				errors.Wrap(patch.ErrStale, "c9 not found"),
				false,
			)
			Expect(encoded.Type).To(Equal("sy.patch.stale"))
			Expect(errors.Decode(ctx, encoded)).To(SatisfyAll(
				MatchError(patch.ErrStale),
				MatchError(ContainSubstring("c9 not found")),
			))
		})
	})
})
