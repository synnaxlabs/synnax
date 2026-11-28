// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Entries", func() {
	var (
		db         *gorp.DB
		svc        *impact.Service
		entriesErr error
	)
	label, rng := resource.TypeLabel, resource.TypeRange
	labelOf := func(k string) resource.ID { return resource.ID{Type: label, Key: k} }
	rangeOf := func(k string) resource.ID { return resource.ID{Type: rng, Key: k} }
	BeforeEach(func() {
		db = DeferClose(gorp.Wrap(memkv.New()))
		entriesErr = nil
		svc = MustSucceed(impact.OpenService(impact.ServiceConfig{
			Deleters: map[resource.Type]impact.Deleter{
				label: {
					Plan: func(
						_ context.Context,
						_ gorp.Snapshot,
						ids []resource.ID,
					) ([]resource.ID, gorp.DeleteEffects, error) {
						return ids, gorp.DeleteEffects{}, nil
					},
					Delete: func(
						context.Context,
						gorp.Tx,
						[]resource.ID,
						gorp.DeleteEffects,
					) error {
						return nil
					},
					// Each label delete touches the range of its key.
					Entries: func(
						_ context.Context,
						_ gorp.Snapshot,
						removed []resource.ID,
						_ gorp.DeleteEffects,
					) ([]impact.Entry, error) {
						entries := make([]impact.Entry, len(removed))
						for i, id := range removed {
							entries[i] = impact.Entry{
								Record: rangeOf(id.Key),
								Effect: impact.EffectDelete,
								Path:   new("folder." + id.Key),
							}
						}
						return entries, entriesErr
					},
				},
			},
			RecordOf: func(t gorp.Target) resource.ID {
				return resource.ID{Type: resource.Type(t.Table), Key: t.Key}
			},
		}))
	})

	It("Should add the entries of a Deleter to the plan, sorted by record", func(
		ctx SpecContext,
	) {
		plan := MustSucceed(svc.Plan(
			ctx,
			db,
			[]resource.ID{labelOf("2"), labelOf("1")},
			impact.PlanOptions{},
		))
		Expect(plan.Entries).To(Equal([]impact.Entry{
			{Record: rangeOf("1"), Effect: impact.EffectDelete, Path: new("folder.1")},
			{Record: rangeOf("2"), Effect: impact.EffectDelete, Path: new("folder.2")},
		}))
	})

	It("Should count an added entry that the caller cannot see", func(
		ctx SpecContext,
	) {
		plan := MustSucceed(svc.Plan(
			ctx,
			db,
			[]resource.ID{labelOf("1"), labelOf("2")},
			impact.PlanOptions{
				Visible: func(id resource.ID) bool { return id != rangeOf("1") },
			},
		))
		Expect(plan.Entries).To(ConsistOf(HaveField("Record", rangeOf("2"))))
		Expect(plan.Hidden.Delete).To(BeEquivalentTo(1))
	})

	It("Should fail the plan with the error of Entries", func(ctx SpecContext) {
		entriesErr = errors.New("entries failed")
		Expect(svc.Plan(
			ctx,
			db,
			[]resource.ID{labelOf("1")},
			impact.PlanOptions{},
		)).Error().To(MatchError(entriesErr))
	})

	Describe("Token", func() {
		records := []resource.ID{labelOf("1")}
		withPath := func(path *string) impact.Plan {
			return impact.Plan{Entries: []impact.Entry{
				{Record: rangeOf("1"), Effect: impact.EffectDelete, Path: path},
			}}
		}

		It("Should change when the path of an entry changes", func() {
			Expect(impact.Token(records, withPath(new("a.b")))).
				ToNot(Equal(impact.Token(records, withPath(new("a.c")))))
		})

		It("Should tell an absent path from an empty one", func() {
			Expect(impact.Token(records, withPath(nil))).
				ToNot(Equal(impact.Token(records, withPath(new("")))))
		})
	})
})
