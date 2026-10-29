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
	DescribeTable("Should reject", func(act label.Action) {
		Expect(label.Reduce(base, act)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("decoding document")),
		))
	},
		Entry(
			"a path outside the name and color",
			set([]string{"key"}, uuid.New().String()),
		),
		Entry("a value the color cannot hold", set([]string{"color"}, "not a color")),
		Entry(
			"a list the label does not declare",
			label.NewInsertAction(label.InsertPayload{
				Path:  []string{"tags"},
				Value: map[string]any{"key": "a"},
			}),
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
		It("Should leave the label unchanged when the field is missing", func() {
			Expect(label.Reduce(
				base,
				label.NewRemoveAction(label.RemovePayload{Key: "tags"}),
			)).To(Equal(base))
		})
	})

	Describe("MovePayload", func() {
		It("Should leave the label unchanged when the list is missing", func() {
			Expect(label.Reduce(base, label.NewMoveAction(label.MovePayload{
				Path: []string{"tags"},
				Key:  "a",
			}))).To(Equal(base))
		})
	})
})
