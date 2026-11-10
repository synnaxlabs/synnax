// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package view_test

import (
	"encoding/json/v2"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/view"
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Update", func() {
	decode := func(data string) (view.Update, error) {
		var u view.Update
		return u, json.Unmarshal([]byte(data), &u)
	}

	It("Should decode an update without a type as base fields only", func() {
		name := "Runs"
		Expect(MustSucceed(decode(`{"name":"Runs"}`)).Variant).
			To(Equal(view.BaseUpdate{Name: &name}))
	})

	It("Should decode an update with a type as the named variant", func() {
		u := MustSucceed(decode(`{"type":"range","where":null}`))
		Expect(u.Variant).To(BeAssignableToTypeOf(view.RangeUpdate{}))
		Expect(u.Variant.(view.RangeUpdate).Where.Cleared()).To(BeTrue())
		Expect(u.Type()).To(Equal(view.TypeRange))
	})

	It("Should reject a variant field in an update without a type", func() {
		Expect(decode(`{"name":"Runs","where":null}`)).Error().
			To(MatchError(ContainSubstring(`unknown object member name "where"`)))
	})

	It("Should encode an update without a type as base fields only", func() {
		name := "Runs"
		Expect(json.Marshal(view.Update{Variant: view.BaseUpdate{Name: &name}})).
			To(MatchJSON(`{"name":"Runs"}`))
	})

	Describe("Apply", func() {
		var v view.View
		BeforeEach(func() {
			v = view.View{Variant: view.RangeView{
				Base:  view.Base{Key: view.Key(uuid.New()), Name: "Runs"},
				Where: &ranger.Where{},
			}}
		})
		reduce := func(p view.UpdatePayload) (view.View, error) {
			return view.Reduce(v, view.NewUpdateAction(p))
		}

		It("Should rename a view without a type", func() {
			next := MustSucceed(reduce(view.UpdatePayload{
				Value: view.Update{Variant: view.BaseUpdate{Name: new("Tests")}},
			}))
			Expect(next.Base()).To(Equal(view.Base{Key: v.Base().Key, Name: "Tests"}))
			Expect(next.Variant.(view.RangeView).Where).To(Equal(&ranger.Where{}))
		})

		It("Should change the variant and keep the base fields", func() {
			next := MustSucceed(reduce(view.UpdatePayload{
				Value: view.Update{Variant: view.StatusUpdate{}},
			}))
			Expect(next.Variant).To(Equal(view.StatusView{Base: v.Base()}))
		})

		It("Should return a stale error when a guard does not hold", func() {
			next, err := reduce(view.UpdatePayload{
				Value:  view.Update{Variant: view.BaseUpdate{Name: new("Tests")}},
				Guards: []patch.Guard{{Path: []string{"type"}, Value: "status"}},
			})
			Expect(err).To(And(
				MatchError(patch.ErrStale),
				MatchError(ContainSubstring("type is not status")),
			))
			Expect(next).To(Equal(v))
		})

		It("Should require a type for a view that holds no variant", func() {
			Expect(view.Update{Variant: view.BaseUpdate{Name: new("Tests")}}.
				ApplyTo(&view.View{})).
				To(MatchError(ContainSubstring("type is required")))
		})
	})
})
