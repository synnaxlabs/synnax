// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Dispatch", func() {
	var (
		svc *device.Service
		dev device.Device
	)
	set := func(path []string, value any) device.Action {
		return device.NewSetAction(device.SetPayload{Path: path, Value: value})
	}
	dispatch := func(ctx context.Context, key device.Key, acts ...device.Action) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).WithDispatchKey("dk").Apply(ctx, key, acts...)
		})
	}
	retrieve := func(ctx context.Context) device.Device {
		GinkgoHelper()
		var d device.Device
		Expect(svc.NewRetrieve().
			Where(device.MatchKeys(dev.Key)).
			Entry(&d).
			Exec(ctx, db)).To(Succeed())
		return d
	}
	// Dispatch commits on its own, so the device commits too and is deleted after each
	// spec.
	BeforeEach(func(ctx SpecContext) {
		MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
		rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:         db,
			HostKey:    node.Identity.HostKey,
			Status:     stat,
			DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roleSvc)),
		}))
		svc = MustOpen(device.OpenService(ctx, device.ServiceConfig{
			DB:     db,
			Status: stat,
		}))
		dev = device.Device{
			Rack:         rackSvc.EmbeddedKey,
			Name:         "PXI-6255",
			Serial:       uuid.New().String(),
			ResourceName: uuid.New().String(),
			Model:        "PXI-6255",
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &dev)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, dev.Key)
			})).To(Succeed())
		})
	})

	It("Should keep edits to different fields from separate dispatches", func(
		ctx SpecContext,
	) {
		Expect(dispatch(ctx, dev.Key, set([]string{"name"}, "Thermocouples"))).
			To(Succeed())
		Expect(dispatch(ctx, dev.Key, set([]string{"model"}, "PXI-6255 (rev B)"))).
			To(Succeed())
		got := retrieve(ctx)
		Expect(got.Name).To(Equal("Thermocouples"))
		Expect(got.Model).To(Equal("PXI-6255 (rev B)"))
		Expect(got.ResourceName).To(Equal(dev.ResourceName))
	})

	It("Should apply none of the actions when one fails", func(ctx SpecContext) {
		Expect(dispatch(
			ctx,
			dev.Key,
			set([]string{"name"}, "Thermocouples"),
			set([]string{"identifier"}, "1dev"),
		)).To(MatchError(ContainSubstring("Identifier must be 2 to 12 characters")))
		Expect(retrieve(ctx).Name).To(Equal("PXI-6255"))
	})

	It("Should refuse a path that is not editable", func(ctx SpecContext) {
		Expect(dispatch(ctx, dev.Key, set([]string{"key"}, uuid.New().String()))).
			To(MatchError(validate.ErrValidation))
	})

	It("Should refuse a create action", func(ctx SpecContext) {
		Expect(dispatch(
			ctx,
			dev.Key,
			device.NewCreateAction(device.CreatePayload{Device: dev}),
		)).To(MatchError(validate.ErrValidation))
	})

	It("Should refuse a device that does not exist", func(ctx SpecContext) {
		Expect(dispatch(ctx, uuid.New(), set([]string{"name"}, "Renamed"))).
			To(MatchError(query.ErrNotFound))
	})
})
