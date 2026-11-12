// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Actions", func() {
	var (
		db  *gorp.DB
		s   stack
		rec *Recorder[rack.Key, rack.Action]
		r   rack.Rack
	)
	write := func(ctx context.Context, fn func(rack.Writer) error) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return fn(s.rack.NewWriter(tx).WithDispatchKey("dk"))
		})
	}
	set := func(path string, value any) rack.Action {
		return rack.NewSetAction(rack.SetPayload{Path: []string{path}, Value: value})
	}
	retrieve := func(ctx context.Context) rack.Rack {
		GinkgoHelper()
		var res rack.Rack
		Expect(s.rack.NewRetrieve().
			Where(rack.MatchKeys(r.Key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		s = DeferClose(openStack(ctx, db, rack.ServiceConfig{}))
		rec = &Recorder[rack.Key, rack.Action]{}
		DeferCleanup(s.rack.Actions().OnChange(rec.Record))
		r = rack.Rack{Name: "Rack", Integrations: []string{"ni"}}
		Expect(write(ctx, func(w rack.Writer) error {
			return w.Create(ctx, &r)
		})).To(Succeed())
	})

	It("Should attach a create that holds the rack", func() {
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].DispatchKey).To(Equal("dk"))
		Expect(frames[0].Actions).To(Equal([]rack.Action{
			rack.NewCreateAction(rack.CreatePayload{Rack: r}),
		}))
	})

	It("Should rename the rack and its status", func(ctx SpecContext) {
		Expect(write(ctx, func(w rack.Writer) error {
			return w.Rename(ctx, r.Key, "Renamed")
		})).To(Succeed())
		Expect(retrieve(ctx).Name).To(Equal("Renamed"))
		var stat rack.Status
		Expect(s.status.NewRetrieve[rack.StatusDetails]().
			Where(status.MatchOwners[rack.StatusDetails](r.ResourceID())).
			Entry(&stat).
			Exec(ctx, db)).To(Succeed())
		Expect(stat.Name).To(Equal("Renamed"))
		Expect(
			rec.Snapshot()[1].Actions,
		).To(Equal([]rack.Action{set("name", "Renamed")}))
	})

	It("Should attach a set of the group for each rack", func(ctx SpecContext) {
		g := group.Group{Name: "Racks", Scope: "device"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return s.group.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		Expect(write(ctx, func(w rack.Writer) error {
			return w.SetGroup(ctx, &g.Key, r.Key)
		})).To(Succeed())
		Expect(retrieve(ctx).Group).To(Equal(&g.Key))
		Expect(rec.Snapshot()[1].Actions).To(Equal([]rack.Action{set("group", &g.Key)}))
	})

	DescribeTable("Should store an editable field", func(
		ctx SpecContext,
		path string,
		value any,
		expect func(*rack.Rack),
	) {
		act := set(path, value)
		Expect(write(ctx, func(w rack.Writer) error {
			return w.Apply(ctx, r.Key, act)
		})).To(Succeed())
		expected := r
		expected.Status = nil
		expect(&expected)
		Expect(retrieve(ctx)).To(Equal(expected))
		Expect(rec.Snapshot()[1].Actions).To(Equal([]rack.Action{act}))
	},
		Entry("integrations", "integrations", []string{"ni", "opc"}, func(
			r *rack.Rack,
		) {
			r.Integrations = []string{"ni", "opc"}
		}),
		Entry("version", "version", "0.50.0", func(r *rack.Rack) {
			r.Version = new("0.50.0")
		}),
	)

	It("Should refuse integrations that are not a list", func(ctx SpecContext) {
		Expect(write(ctx, func(w rack.Writer) error {
			return w.Apply(ctx, r.Key, set("integrations", nil))
		})).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("integrations")),
		))
		Expect(retrieve(ctx).Integrations).To(Equal([]string{"ni"}))
	})

	DescribeTable("Should refuse a path outside the editable fields", func(
		ctx SpecContext,
		path string,
		value any,
	) {
		Expect(write(ctx, func(w rack.Writer) error {
			return w.Apply(ctx, r.Key, set(path, value))
		})).To(MatchError(validate.ErrValidation))
		Expect(retrieve(ctx).Integrations).To(Equal([]string{"ni"}))
		Expect(rec.Snapshot()).To(HaveLen(1))
	},
		Entry("key", "key", 7),
		Entry("embedded", "embedded", true),
		Entry("role", "role", "00000000-0000-0000-0000-000000000000"),
	)

	It("Should reject a create", func(ctx SpecContext) {
		Expect(write(ctx, func(w rack.Writer) error {
			return w.Apply(
				ctx,
				r.Key,
				rack.NewCreateAction(rack.CreatePayload{Rack: r}),
			)
		})).To(MatchError(ContainSubstring("only the Core dispatches a rack create")))
	})
})
