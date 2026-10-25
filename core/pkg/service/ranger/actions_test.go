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
		It("Should replace the range with the created range", func() {
			created := ranger.Range{Key: base.Key, Name: "Created"}
			Expect(ranger.Reduce(
				base,
				ranger.NewCreateAction(ranger.CreatePayload{Range: created}),
			)).To(Equal(created))
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
			Expect(res.Color).To(Equal(new(color.MustFromHex("#ff0000"))))
		})
		It("Should keep the key, labels, and parent", func() {
			base.Parent = &ranger.Range{Key: uuid.New()}
			res := MustSucceed(ranger.Reduce(base, set([]string{"name"}, "Cold flow")))
			Expect(res.Key).To(Equal(base.Key))
			Expect(res.Parent).To(Equal(base.Parent))
		})
		It("Should reject a path outside the name, time range, and color", func() {
			act := set([]string{"key"}, uuid.New().String())
			Expect(ranger.Reduce(base, act)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("decoding document")),
			))
		})
		It("Should reject a value the time range cannot hold", func() {
			act := set([]string{"time_range", "end"}, "soon")
			Expect(ranger.Reduce(base, act)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("decoding document")),
			))
		})
	})

	Describe("RemovePayload", func() {
		It("Should remove the color", func() {
			base.Color = new(color.MustFromHex("#00ff00"))
			res := MustSucceed(ranger.Reduce(
				base,
				ranger.NewRemoveAction(ranger.RemovePayload{Key: "color"}),
			))
			Expect(res.Color).To(BeNil())
		})
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
				MatchError(ContainSubstring("decoding document")),
			))
		})
	})

	Describe("MovePayload", func() {
		It("Should leave the range unchanged when the list is missing", func() {
			Expect(ranger.Reduce(base, ranger.NewMoveAction(ranger.MovePayload{
				Path: []string{"tags"},
				Key:  "a",
			}))).To(Equal(base))
		})
	})
})
