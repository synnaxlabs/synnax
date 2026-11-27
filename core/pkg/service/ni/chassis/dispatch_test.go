// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package chassis_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ni/chassis"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Dispatch", func() {
	var (
		stat *status.Service
		svc  *chassis.Service
		c    chassis.Chassis
	)
	set := func(path []string, value any) chassis.Action {
		return chassis.NewSetAction(chassis.SetPayload{Path: path, Value: value})
	}
	dispatch := func(ctx context.Context, key chassis.Key, acts ...chassis.Action) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).WithDispatchKey("dk").Apply(ctx, key, acts...)
		})
	}
	retrieve := func(ctx context.Context) chassis.Chassis {
		GinkgoHelper()
		var got chassis.Chassis
		Expect(svc.NewRetrieve().
			Where(chassis.MatchKeys(c.Key)).
			Entry(&got).
			Exec(ctx, db)).To(Succeed())
		return got
	}
	// Dispatch commits on its own, so the chassis commits too and is deleted after each
	// spec.
	BeforeEach(func(ctx SpecContext) {
		MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		stat = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
		rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:         db,
			HostKey:    1,
			Status:     stat,
			DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roleSvc)),
		}))
		svc = MustOpen(chassis.OpenService(ctx, chassis.ServiceConfig{
			DB:     db,
			Status: stat,
		}))
		c = chassis.Chassis{
			Rack:         rackSvc.EmbeddedKey,
			Name:         "PXIe-1071",
			Serial:       uuid.New().String(),
			ResourceName: uuid.New().String(),
			Model:        "PXIe-1071",
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &c)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, c.Key)
			})).To(Succeed())
		})
	})

	It("Should rename the status when a dispatch renames the chassis", func(
		ctx SpecContext,
	) {
		Expect(dispatch(ctx, c.Key, set([]string{"name"}, "Bench"))).To(Succeed())
		var s chassis.Status
		Expect(stat.NewRetrieve[chassis.StatusDetails]().
			Where(status.MatchOwners[chassis.StatusDetails](c.ResourceID())).
			Entry(&s).
			Exec(ctx, db)).To(Succeed())
		Expect(s.Name).To(Equal("Bench"))
	})

	It("Should keep edits to different fields from separate dispatches", func(
		ctx SpecContext,
	) {
		Expect(dispatch(ctx, c.Key, set([]string{"name"}, "Bench"))).
			To(Succeed())
		Expect(dispatch(ctx, c.Key, set([]string{"model"}, "PXIe-1071 (rev B)"))).
			To(Succeed())
		got := retrieve(ctx)
		Expect(got.Name).To(Equal("Bench"))
		Expect(got.Model).To(Equal("PXIe-1071 (rev B)"))
		Expect(got.ResourceName).To(Equal(c.ResourceName))
	})

	It("Should apply none of the actions when one fails", func(ctx SpecContext) {
		Expect(dispatch(
			ctx,
			c.Key,
			set([]string{"model"}, "PXIe-1071 (rev B)"),
			set([]string{"name"}, ""),
		)).To(MatchError(ContainSubstring("name: required")))
		Expect(retrieve(ctx).Model).To(Equal("PXIe-1071"))
	})

	It("Should refuse a path that is not editable", func(ctx SpecContext) {
		Expect(dispatch(ctx, c.Key, set([]string{"key"}, uuid.New().String()))).
			To(MatchError(validate.ErrValidation))
	})

	It("Should refuse a create action", func(ctx SpecContext) {
		Expect(dispatch(
			ctx,
			c.Key,
			chassis.NewCreateAction(chassis.CreatePayload{Chassis: c}),
		)).To(MatchError(validate.ErrValidation))
	})

	It("Should refuse a chassis that does not exist", func(ctx SpecContext) {
		Expect(dispatch(ctx, uuid.New(), set([]string{"name"}, "Renamed"))).
			To(MatchError(query.ErrNotFound))
	})
})
