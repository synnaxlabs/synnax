// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package enumeration_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Reduce", func() {
	var base enumeration.Enumeration
	BeforeEach(func() {
		base = enumeration.Enumeration{
			Key:  uuid.New(),
			Name: "Valve",
			Entries: []enumeration.Entry{
				{Key: "closed", Value: 0, Name: "Closed"},
				{Key: "open", Value: 1, Name: "Open"},
			},
		}
	})

	It("Should rename an entry", func() {
		res := MustSucceed(enumeration.Reduce(
			base,
			enumeration.NewSetAction(enumeration.SetPayload{
				Path:  []string{"entries", "open", "name"},
				Value: "Opened",
			}),
		))
		Expect(res.Entries[1].Name).To(Equal("Opened"))
		Expect(res.Entries[0]).To(Equal(base.Entries[0]))
	})

	It("Should insert an entry after another", func() {
		res := MustSucceed(enumeration.Reduce(
			base,
			enumeration.NewInsertAction(enumeration.InsertPayload{
				Path: []string{"entries"},
				Value: map[string]any{
					"key":      "fault",
					"value":    2,
					"name":     "Fault",
					"abnormal": true,
				},
				After: new("closed"),
			}),
		))
		Expect(res.Entries).To(Equal([]enumeration.Entry{
			base.Entries[0],
			{Key: "fault", Value: 2, Name: "Fault", Abnormal: true},
			base.Entries[1],
		}))
	})

	It("Should move an entry to the start", func() {
		res := MustSucceed(enumeration.Reduce(
			base,
			enumeration.NewMoveAction(enumeration.MovePayload{
				Path: []string{"entries"},
				Key:  "open",
			}),
		))
		Expect(res.Entries).To(Equal([]enumeration.Entry{
			base.Entries[1],
			base.Entries[0],
		}))
	})

	It("Should remove an entry", func() {
		res := MustSucceed(enumeration.Reduce(
			base,
			enumeration.NewRemoveAction(enumeration.RemovePayload{
				Path: []string{"entries"},
				Key:  "closed",
			}),
		))
		Expect(res.Entries).To(Equal([]enumeration.Entry{base.Entries[1]}))
	})

	DescribeTable("Should reject", func(act enumeration.Action, msg string) {
		Expect(enumeration.Reduce(base, act)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(msg)),
		))
	},
		Entry(
			"a change to the key",
			enumeration.NewSetAction(enumeration.SetPayload{
				Path:  []string{"key"},
				Value: uuid.New().String(),
			}),
			"[key]: the path names no editable field",
		),
		Entry(
			"a create",
			enumeration.NewCreateAction(enumeration.CreatePayload{Enumeration: base}),
			"only the Core dispatches an enumeration create",
		),
	)
})
