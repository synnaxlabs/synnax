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
			Color: color.MustFromHex("#00ff00"),
		}
	})
	set := func(path []string, value any) label.Action {
		return label.NewSetAction(label.SetPayload{Path: path, Value: value})
	}
	expectRejected := func(acts ...label.Action) {
		GinkgoHelper()
		next, err := label.Reduce(base, acts...)
		Expect(err).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("decoding document")),
		))
		Expect(next).To(Equal(base))
	}

	Describe("CreatePayload", func() {
		It("Should replace the label with the created label", func() {
			created := label.Label{Key: base.Key, Name: "Created"}
			Expect(label.Reduce(
				base,
				label.NewCreateAction(label.CreatePayload{Label: created}),
			)).To(Equal(created))
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
			Expect(res.Color).To(Equal(color.MustFromHex("#ff0000")))
			Expect(res.Name).To(Equal(base.Name))
		})
		It("Should reject a path outside the name and color", func() {
			expectRejected(set([]string{"key"}, uuid.New().String()))
		})
		It("Should reject a value the color cannot hold", func() {
			expectRejected(set([]string{"color"}, "not a color"))
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

	Describe("InsertPayload", func() {
		It("Should reject a list the label does not declare", func() {
			expectRejected(label.NewInsertAction(label.InsertPayload{
				Path:  []string{"tags"},
				Value: map[string]any{"key": "a"},
			}))
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
