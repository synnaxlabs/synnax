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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Retrieve", func() {
	var (
		retrieveDB *gorp.DB
		svc        *device.Service
		breaker    device.Device
	)
	BeforeEach(func(ctx SpecContext) {
		retrieveDB = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(retrieveDB.Run)
		MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: retrieveDB}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: retrieveDB}))
		stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: retrieveDB}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: retrieveDB}))
		racks := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:      retrieveDB,
			HostKey: 1,
			Status:  stat,
			DriverRole: MustSucceed(
				racktestutil.CreateDriverRole(ctx, retrieveDB, roleSvc),
			),
		}))
		svc = MustOpen(device.OpenService(ctx, device.ServiceConfig{
			DB:     retrieveDB,
			Status: stat,
		}))
		created := lo.Map([]string{"Valve", "amber", "Breaker"}, func(
			name string,
			_ int,
		) device.Device {
			return device.Device{
				Rack:         racks.EmbeddedKey,
				Name:         name,
				Serial:       uuid.New().String(),
				ResourceName: uuid.New().String(),
				Model:        "PXI-6255",
			}
		})
		Expect(retrieveDB.WithTx(ctx, func(tx gorp.Tx) error {
			for i := range created {
				if err := svc.NewWriter(tx).Create(ctx, &created[i]); err != nil {
					return err
				}
			}
			return nil
		})).To(Succeed())
		breaker = created[2]
	})
	names := func(ctx SpecContext, q device.Retrieve) []string {
		GinkgoHelper()
		var res []device.Device
		Expect(q.Entries(&res).Exec(ctx, retrieveDB)).To(Succeed())
		return lo.Map(res, func(d device.Device, _ int) string { return d.Name })
	}

	Describe("MatchNames", func() {
		It("Should keep the devices with the given names", func(ctx SpecContext) {
			q := svc.NewRetrieve().Where(device.MatchNames("Valve", "amber"))
			Expect(names(ctx, q)).To(ConsistOf("Valve", "amber"))
		})

		It("Should match the name in its exact case", func(ctx SpecContext) {
			q := svc.NewRetrieve().Where(device.MatchNames("AMBER"))
			Expect(names(ctx, q)).To(BeEmpty())
		})
	})

	Describe("OrderBy", func() {
		DescribeTable(
			"Should walk the devices by name, ignoring case",
			func(ctx SpecContext, dir gorp.Direction, expected []string) {
				q := svc.NewRetrieve().OrderBy(device.OrderByName(dir))
				Expect(names(ctx, q)).To(Equal(expected))
			},
			Entry(
				"ascending",
				gorp.DirectionAsc,
				[]string{"amber", "Breaker", "Valve"},
			),
			Entry(
				"descending",
				gorp.DirectionDesc,
				[]string{"Valve", "Breaker", "amber"},
			),
		)

		DescribeTable(
			"Should start strictly past the name and key",
			func(ctx SpecContext, dir gorp.Direction, expected []string) {
				o := device.OrderByNameAfter(dir, breaker.Name, breaker.Key)
				Expect(names(ctx, svc.NewRetrieve().OrderBy(o))).To(Equal(expected))
			},
			Entry("ascending", gorp.DirectionAsc, []string{"Valve"}),
			Entry("descending", gorp.DirectionDesc, []string{"amber"}),
		)
	})
})
