// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Actions", func() {
	base := icd.ICD{Key: uuid.New(), Name: "Bus", Messages: []icd.Message{}}
	set := func(path []string, value any) icd.Action {
		return icd.NewSetAction(icd.SetPayload{Path: path, Value: value})
	}

	DescribeTable("Should reject", func(act icd.Action) {
		Expect(icd.Reduce(base, act)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("decoding document")),
		))
	},
		Entry("a path to the key", set([]string{"key"}, uuid.New().String())),
		Entry("a value the name cannot hold", set([]string{"name"}, 1)),
		Entry(
			"a list the ICD does not declare",
			icd.NewInsertAction(icd.InsertPayload{
				Path:  []string{"tags"},
				Value: map[string]any{"key": "a"},
			}),
		),
	)

	It("Should reject a create", func() {
		Expect(icd.Reduce(
			base,
			icd.NewCreateAction(icd.CreatePayload{ICD: base}),
		)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("only the Core dispatches an ICD create")),
		))
	})

	It("Should reject the removal of the name", func() {
		Expect(icd.Reduce(
			base,
			icd.NewRemoveAction(icd.RemovePayload{Key: "name"}),
		)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("[name]: the document requires a value")),
		))
	})

	It("Should set the name and keep the key", func() {
		Expect(icd.Reduce(base, set([]string{"name"}, "Vehicle"))).To(Equal(icd.ICD{
			Key:      base.Key,
			Name:     "Vehicle",
			Messages: []icd.Message{},
		}))
	})
})
