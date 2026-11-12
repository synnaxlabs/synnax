// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/x/color"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Actions", func() {
	var base label.Label
	BeforeEach(func() {
		base = label.Label{
			Key:   uuid.New(),
			Name:  "Nominal",
			Color: MustSucceed(color.FromHex("#00ff00")),
		}
	})
	set := func(path []string, value any) label.Action {
		return label.NewSetAction(label.SetPayload{Path: path, Value: value})
	}
	DescribeTable("Should reject", func(act label.Action, msg string) {
		Expect(label.Reduce(base, act)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(msg)),
		))
	},
		Entry(
			"a path outside the name and color",
			set([]string{"key"}, uuid.New().String()),
			"[key]: the path names no editable field",
		),
		Entry(
			"a value the color cannot hold",
			set([]string{"color"}, "not a color"),
			"[color]: decoding value",
		),
		Entry(
			"a legacy object color",
			set([]string{"color"}, map[string]any{"r": 255, "g": 0, "b": 0, "a": 0.5}),
			"color is not a hex string",
		),
		Entry(
			"a path into the color",
			set([]string{"color", "a"}, 128),
			"[color]: the value takes only a whole set",
		),
		Entry(
			"an insert into a list the label does not declare",
			label.NewInsertAction(label.InsertPayload{
				Path:  []string{"tags"},
				Value: map[string]any{"key": "a"},
			}),
			"[tags]: the path names no editable field",
		),
		Entry(
			"a removal of a field the label does not declare",
			label.NewRemoveAction(label.RemovePayload{Key: "tags"}),
			"[tags]: the path names no editable field",
		),
		Entry(
			"a move in a list the label does not declare",
			label.NewMoveAction(label.MovePayload{Path: []string{"tags"}, Key: "a"}),
			"[tags]: the path names no editable field",
		),
	)

	Describe("CreatePayload", func() {
		It("Should reject a create", func() {
			Expect(label.Reduce(
				base,
				label.NewCreateAction(label.CreatePayload{Label: base}),
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("only the Core dispatches a label create")),
			))
		})
	})

	Describe("SetPayload", func() {
		It("Should set the name and keep the key and color", func() {
			res := MustSucceed(label.Reduce(base, set([]string{"name"}, "Anomaly")))
			Expect(res).To(Equal(label.Label{
				Key:   base.Key,
				Name:  "Anomaly",
				Color: base.Color,
			}))
		})
		It("Should set the color", func() {
			res := MustSucceed(label.Reduce(base, set([]string{"color"}, "#ff0000")))
			Expect(res.Color).To(Equal(MustSucceed(color.FromHex("#ff0000"))))
			Expect(res.Name).To(Equal(base.Name))
		})
	})

	Describe("RemovePayload", func() {
		It("Should reject the removal of the color", func() {
			Expect(label.Reduce(
				base,
				label.NewRemoveAction(label.RemovePayload{Key: "color"}),
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("[color]: the document requires a value")),
			))
		})
	})
})
