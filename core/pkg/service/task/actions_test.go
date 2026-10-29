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
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
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
			Config: msgpack.EncodedJSON{
				"routing_key": "rk-1",
				"alerts": []any{
					map[string]any{"key": "a", "status": "st-a"},
					map[string]any{"key": "b", "status": "st-b"},
				},
			},
		}
	})
	alertKeys := func(t task.Task) []string {
		GinkgoHelper()
		var keys []string
		for _, a := range t.Config["alerts"].([]any) {
			keys = append(keys, a.(map[string]any)["key"].(string))
		}
		return keys
	}

	Describe("CreatePayload", func() {
		It("Should reject a create", func() {
			Expect(task.Reduce(
				base,
				task.NewCreateAction(task.CreatePayload{Task: base}),
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("only the Core dispatches a task create")),
			))
		})
	})

	Describe("SetPayload", func() {
		It("Should set a field of the config", func() {
			next := MustSucceed(task.Reduce(base, task.NewSetAction(task.SetPayload{
				Path:  []string{"config", "routing_key"},
				Value: "rk-2",
			})))
			Expect(next.Config["routing_key"]).To(Equal("rk-2"))
		})

		It("Should set a field of a list entry by its key", func() {
			next := MustSucceed(task.Reduce(base, task.NewSetAction(task.SetPayload{
				Path:  []string{"config", "alerts", "b", "status"},
				Value: "st-c",
			})))
			Expect(next.Config["alerts"]).To(ContainElement(
				map[string]any{"key": "b", "status": "st-c"},
			))
		})

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

		It("Should do nothing when a guard does not hold", func() {
			Expect(task.Reduce(base, task.NewSetAction(task.SetPayload{
				Path:   []string{"config", "routing_key"},
				Value:  "rk-2",
				Guards: []patch.Guard{{Path: []string{"name"}, Value: "Other"}},
			}))).To(Equal(base))
		})

		It("Should not change the config of the given task", func() {
			MustSucceed(task.Reduce(base, task.NewSetAction(task.SetPayload{
				Path:  []string{"config", "routing_key"},
				Value: "rk-2",
			})))
			Expect(base.Config["routing_key"]).To(Equal("rk-1"))
		})

		DescribeTable("Should reject a path to a field that actions cannot edit",
			func(field string) {
				Expect(task.Reduce(base, task.NewSetAction(task.SetPayload{
					Path:  []string{field},
					Value: "x",
				}))).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(`unknown object member name "`+field)),
				))
			},
			Entry("type", "type"),
			Entry("key", "key"),
			Entry("snapshot", "snapshot"),
			Entry("config hash", "config_hash"),
		)

		It("Should reject a value that the field cannot hold", func() {
			Expect(task.Reduce(base, task.NewSetAction(task.SetPayload{
				Path:  []string{"rack"},
				Value: "not a rack",
			}))).Error().To(MatchError(validate.ErrValidation))
		})
	})

	Describe("RemovePayload", func() {
		It("Should remove a list entry by its key", func() {
			next := MustSucceed(task.Reduce(
				base,
				task.NewRemoveAction(task.RemovePayload{
					Path: []string{"config", "alerts"},
					Key:  "a",
				}),
			))
			Expect(alertKeys(next)).To(Equal([]string{"b"}))
		})
	})

	Describe("InsertPayload", func() {
		It("Should insert a list entry after the given entry", func() {
			next := MustSucceed(task.Reduce(
				base,
				task.NewInsertAction(task.InsertPayload{
					Path:  []string{"config", "alerts"},
					Value: map[string]any{"key": "c", "status": "st-c"},
					After: new("a"),
				}),
			))
			Expect(alertKeys(next)).To(Equal([]string{"a", "c", "b"}))
		})
	})

	Describe("MovePayload", func() {
		It("Should move a list entry to the front", func() {
			next := MustSucceed(task.Reduce(base, task.NewMoveAction(task.MovePayload{
				Path: []string{"config", "alerts"},
				Key:  "b",
			})))
			Expect(alertKeys(next)).To(Equal([]string{"b", "a"}))
		})
	})
})
