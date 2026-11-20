// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Actions", func() {
	var base ranger.Range
	BeforeEach(func() {
		base = ranger.Range{
			Key:       uuid.New(),
			Name:      "Hot fire",
			TimeRange: telem.NewRangeSeconds(1, 5),
		}
	})
	set := func(path []string, value any) ranger.Action {
		return ranger.NewSetAction(ranger.SetPayload{Path: path, Value: value})
	}

	Describe("CreatePayload", func() {
		It("Should reject a create", func() {
			Expect(ranger.Reduce(
				base,
				ranger.NewCreateAction(ranger.CreatePayload{Range: base}),
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("only the Core dispatches a range create")),
			))
		})
	})

	Describe("SetPayload", func() {
		It("Should set the name", func() {
			res := MustSucceed(ranger.Reduce(base, set([]string{"name"}, "Cold flow")))
			Expect(res.Name).To(Equal("Cold flow"))
			Expect(res.TimeRange).To(Equal(base.TimeRange))
		})
		It("Should set the end of the time range from a string", func() {
			res := MustSucceed(ranger.Reduce(
				base,
				set([]string{"time_range", "end"}, "9000000000"),
			))
			Expect(res.TimeRange).To(Equal(telem.NewRangeSeconds(1, 9)))
		})
		It("Should set the end of the time range from a timestamp", func() {
			res := MustSucceed(ranger.Reduce(
				base,
				set([]string{"time_range", "end"}, telem.SecondTS*9),
			))
			Expect(res.TimeRange).To(Equal(telem.NewRangeSeconds(1, 9)))
		})
		It("Should set a color the range does not have", func() {
			res := MustSucceed(ranger.Reduce(base, set([]string{"color"}, "#ff0000")))
			Expect(res.Color).To(Equal(new(MustSucceed(color.Parse("#ff0000")))))
		})
		It("Should reject a path into the color", func() {
			base.Color = new(MustSucceed(color.Parse("#00ff00")))
			act := set([]string{"color", "a"}, 128)
			Expect(ranger.Reduce(base, act)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(
					ContainSubstring("[color]: the value takes only a whole set"),
				),
			))
		})
		It("Should set the parent", func() {
			parent := uuid.New()
			res := MustSucceed(ranger.Reduce(
				base,
				set([]string{"parent"}, parent.String()),
			))
			Expect(res.Parent).To(HaveValue(Equal(parent)))
		})
		It("Should keep the key, labels, and parent", func() {
			base.Labels = []label.Key{uuid.New()}
			base.Parent = new(uuid.New())
			res := MustSucceed(ranger.Reduce(base, set([]string{"name"}, "Cold flow")))
			Expect(res.Key).To(Equal(base.Key))
			Expect(res.Labels).To(Equal(base.Labels))
			Expect(res.Parent).To(Equal(base.Parent))
		})
		DescribeTable("Should reject", func(path []string, value any, msg string) {
			Expect(ranger.Reduce(base, set(path, value))).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(msg)),
			))
		},
			Entry(
				"a path outside the document",
				[]string{"key"},
				uuid.New().String(),
				"[key]: the path names no editable field",
			),
			Entry(
				"a set of the labels",
				[]string{"labels"},
				[]string{uuid.New().String()},
				"[labels]: the path names no editable field",
			),
			Entry(
				"a value the time range cannot hold",
				[]string{"time_range", "end"},
				"soon",
				"[time_range, end]: decoding value",
			),
			Entry(
				"a legacy object color",
				[]string{"color"},
				map[string]any{"r": 255, "g": 0, "b": 0, "a": 0.5},
				"color is not a hex string",
			),
		)
	})

	Describe("RemovePayload", func() {
		It("Should remove the color", func() {
			base.Color = new(MustSucceed(color.Parse("#00ff00")))
			res := MustSucceed(ranger.Reduce(
				base,
				ranger.NewRemoveAction(ranger.RemovePayload{Key: "color"}),
			))
			Expect(res.Color).To(BeNil())
		})
		It("Should remove the parent", func() {
			base.Parent = new(uuid.New())
			res := MustSucceed(ranger.Reduce(
				base,
				ranger.NewRemoveAction(ranger.RemovePayload{Key: "parent"}),
			))
			Expect(res.Parent).To(BeNil())
		})
		DescribeTable(
			"Should reject the removal of",
			func(
				act ranger.Action,
				path string,
			) {
				Expect(ranger.Reduce(base, act)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(
						ContainSubstring(path+": the document requires a value"),
					),
				))
			},
			Entry("the time range", ranger.NewRemoveAction(ranger.RemovePayload{
				Key: "time_range",
			}), "[time_range]"),
			Entry(
				"an end of the time range",
				ranger.NewRemoveAction(ranger.RemovePayload{
					Path: []string{"time_range"},
					Key:  "end",
				}),
				"[time_range, end]",
			),
		)
	})

	Describe("InsertPayload", func() {
		It("Should reject a list the range does not declare", func() {
			Expect(ranger.Reduce(
				base,
				ranger.NewInsertAction(ranger.InsertPayload{
					Path:  []string{"tags"},
					Value: map[string]any{"key": "a"},
				}),
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(
					ContainSubstring("[tags]: the path names no editable field"),
				),
			))
		})
	})

	Describe("MovePayload", func() {
		It("Should reject a list the range does not declare", func() {
			Expect(ranger.Reduce(base, ranger.NewMoveAction(ranger.MovePayload{
				Path: []string{"tags"},
				Key:  "a",
			}))).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(
					ContainSubstring("[tags]: the path names no editable field"),
				),
			))
		})
	})
})
