// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package color_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	xcolor "github.com/synnaxlabs/x/color"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Reduce", func() {
	var base color.Named
	BeforeEach(func() {
		dark := MustSucceed(xcolor.Parse("#00ff00"))
		base = color.Named{
			Key:   uuid.New(),
			Name:  "Breaker",
			Light: MustSucceed(xcolor.Parse("#ff0000")),
			Dark:  &dark,
		}
	})

	It("Should set the light color", func() {
		res := MustSucceed(color.Reduce(base, color.NewSetAction(color.SetPayload{
			Path:  []string{"light"},
			Value: "#0000ff",
		})))
		Expect(res.Light).To(Equal(MustSucceed(xcolor.Parse("#0000ff"))))
		Expect(res.Dark).To(Equal(base.Dark))
	})

	It("Should remove the dark color", func() {
		res := MustSucceed(color.Reduce(base, color.NewRemoveAction(
			color.RemovePayload{Key: "dark"},
		)))
		Expect(res.Dark).To(BeNil())
		Expect(res.Light).To(Equal(base.Light))
	})

	DescribeTable("Should reject", func(act color.Action, msg string) {
		Expect(color.Reduce(base, act)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(msg)),
		))
	},
		Entry(
			"a change to the key",
			color.NewSetAction(color.SetPayload{
				Path:  []string{"key"},
				Value: uuid.New().String(),
			}),
			"[key]: the path names no editable field",
		),
		Entry(
			"the removal of the light color",
			color.NewRemoveAction(color.RemovePayload{Key: "light"}),
			"[light]: the document requires a value",
		),
		Entry(
			"an insert into a list the named color does not declare",
			color.NewInsertAction(color.InsertPayload{
				Path:  []string{"tags"},
				Value: map[string]any{"key": "a"},
			}),
			"[tags]: the path names no editable field",
		),
		Entry(
			"a move in a list the named color does not declare",
			color.NewMoveAction(color.MovePayload{Path: []string{"tags"}, Key: "a"}),
			"[tags]: the path names no editable field",
		),
	)
})
