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
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/labjack/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Service", func() {
	var (
		stat       *status.Service
		rackSvc    *rack.Service
		svc        *device.Service
		channelSvc *channel.Service
		tx         gorp.Tx
		rackKey    rack.Key
	)
	BeforeEach(func(ctx SpecContext) {
		MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		stat = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
		rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:         db,
			HostKey:    node.Identity.HostKey,
			Status:     stat,
			DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roleSvc)),
		}))
		rackKey = rackSvc.EmbeddedKey
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			DB:      db,
			HostKey: node.Identity.HostKey,
			Status:  stat,
		}))
		svc = MustOpen(device.OpenService(ctx, device.ServiceConfig{DB: db}))
		tx = DeferClose(db.OpenTx())
	})

	var serial uint32
	nextSerial := func() uint32 {
		serial++
		return serial
	}
	create := func(ctx context.Context) device.Device {
		GinkgoHelper()
		d := device.Device{
			Key:        uuid.New(),
			Rack:       rackKey,
			Name:       "T7-0001",
			Serial:     nextSerial(),
			Connection: device.ConnectionUSB,
			Model:      device.ModelT7,
			Ports:      device.Ports{Variant: device.T7Ports{}},
		}
		Expect(gorp.NewCreate[device.Key, device.Device]().Entry(&d).Exec(ctx, tx)).
			To(Succeed())
		return d
	}

	Describe("Channels", func() {
		createIndex := func(ctx context.Context) channel.Key {
			GinkgoHelper()
			ch := channel.Channel{
				Name:     channeltestutil.UniqueChannelName(),
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
			Expect(channelSvc.NewWriter(tx).Create(ctx, &ch)).To(Succeed())
			return ch.Key()
		}
		bind := func(ctx context.Context, read *channel.Key) error {
			d := device.Device{
				Key:        uuid.New(),
				Rack:       rackKey,
				Name:       "T7-0002",
				Serial:     nextSerial(),
				Connection: device.ConnectionUSB,
				Model:      device.ModelT7,
				Ports:      device.Ports{Variant: device.T7Ports{}},
				ReadIndex:  read,
			}
			return gorp.NewCreate[device.Key, device.Device]().Entry(&d).Exec(ctx, tx)
		}

		It("Should keep a channel that a device binds from being deleted", func(
			ctx SpecContext,
		) {
			ch := createIndex(ctx)
			Expect(bind(ctx, new(ch))).To(Succeed())
			Expect(channelSvc.NewWriter(tx).Delete(ctx, ch, false)).
				To(MatchError(validate.ErrValidation))
		})
	})

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(device.OpenService(ctx, device.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})

		It("Should let a later config override an earlier one", func() {
			cfg := device.ServiceConfig{}.Override(device.ServiceConfig{DB: db})
			Expect(cfg.DB).To(BeIdenticalTo(db))
			Expect(cfg.Validate()).To(Succeed())
		})
	})

	Describe("Retrieve", func() {
		It("Should match devices by serial", func(ctx SpecContext) {
			d := create(ctx)
			create(ctx)
			var got []device.Device
			Expect(svc.NewRetrieve().
				Where(device.MatchSerials(d.Serial)).
				Entries(&got).
				Exec(ctx, tx)).To(Succeed())
			Expect(got).To(HaveLen(1))
			Expect(got[0].Key).To(Equal(d.Key))
		})
	})

	Describe("ResourceID", func() {
		It("Should name a device by its key under the LabJack device type", func() {
			key := uuid.New()
			Expect(device.ResourceID(key)).To(Equal(resource.ID{
				Type: resource.TypeLabjackDevice,
				Key:  key.String(),
			}))
		})

		It("Should match the ID of a stored device", func(ctx SpecContext) {
			d := create(ctx)
			Expect(device.ResourceID(d.Key)).To(Equal(d.ResourceID()))
		})
	})

	Describe("Resources", func() {
		It("Should name each device by its name", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{d.ResourceID()})).
				To(Equal([]resource.Resource{{ID: d.ResourceID(), Name: d.Name}}))
		})

		It("Should reject a key that is not a UUID", func(ctx SpecContext) {
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{{
				Type: resource.TypeLabjackDevice,
				Key:  "not-a-uuid",
			}})).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should find a device by its name", func(ctx SpecContext) {
			d := create(ctx)
			res := MustSucceed(svc.Resources().Search(
				ctx,
				tx,
				search.Query[resource.ID]{Term: d.Name, Limit: 10},
			))
			Expect(res.Keys).To(ContainElement(d.ResourceID()))
		})
	})
})
