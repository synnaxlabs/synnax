// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Actions", func() {
	var base task.Task
	BeforeEach(func() {
		base = task.Task{
			Key:  uuid.New(),
			Name: "Alerts",
			Type: "pagerduty_alert",
			Rack: 1,
		}
	})

	Describe("CreatePayload", func() {
		It("Should reject a create", func() {
			Expect(task.Reduce(
				base,
				task.NewCreateAction(task.CreatePayload{Task: base}),
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(actions.ErrCoreOnly),
			))
		})
	})

	Describe("SetPayload", func() {
		It("Should set the name", func() {
			next := MustSucceed(task.Reduce(base, task.NewSetAction(task.SetPayload{
				Path:  []string{"name"},
				Value: "Renamed",
			})))
			Expect(next.Name).To(Equal("Renamed"))
		})

		It("Should set the rack", func() {
			next := MustSucceed(task.Reduce(base, task.NewSetAction(task.SetPayload{
				Path:  []string{"rack"},
				Value: 7,
			})))
			Expect(next.Rack).To(BeEquivalentTo(7))
		})

		It("Should reject a guard that names no union the path goes into", func() {
			Expect(task.Reduce(base, task.NewSetAction(task.SetPayload{
				Path:   []string{"name"},
				Value:  "Renamed",
				Guards: []patch.Guard{{Path: []string{"name"}, Value: "Other"}},
			}))).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"[name]: the guard names no union that the path goes into",
				)),
			))
		})

		DescribeTable("Should reject a path to a field that actions cannot edit",
			func(field string) {
				Expect(task.Reduce(base, task.NewSetAction(task.SetPayload{
					Path:  []string{field},
					Value: "x",
				}))).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("the path names no editable field")),
				))
			},
			Entry("type", "type"),
			Entry("key", "key"),
			Entry("snapshot", "snapshot"),
			Entry("config hash", "config_hash"),
		)

		It("Should replace the config whole", func() {
			next := MustSucceed(task.Reduce(base, task.NewSetAction(task.SetPayload{
				Path:  []string{"config"},
				Value: map[string]any{"rate": 10},
			})))
			Expect(next.Config).To(HaveKeyWithValue("rate", BeNumerically("==", 10)))
		})

		It("Should reject a value that the field cannot hold", func() {
			Expect(task.Reduce(base, task.NewSetAction(task.SetPayload{
				Path:  []string{"rack"},
				Value: "not a rack",
			}))).Error().To(MatchError(validate.ErrValidation))
		})
	})

	Describe("RemovePayload", func() {
		It("Should reject a removal of the name", func() {
			Expect(task.Reduce(base, task.NewRemoveAction(task.RemovePayload{
				Key: "name",
			}))).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("the document requires a value")),
			))
		})
	})
})
