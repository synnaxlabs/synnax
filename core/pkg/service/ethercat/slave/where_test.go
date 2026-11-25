// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package slave_test

import (
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/ethercat/master"
	"github.com/synnaxlabs/synnax/pkg/service/ethercat/slave"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/testutil/vectors"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Where", Ordered, func() {
	var (
		v       vectors.Vectors[slave.Slave, slave.Where]
		whereDB *gorp.DB
		svc     *slave.Service
	)
	v = MustSucceed(vectors.Read[slave.Slave, slave.Where]("ethercat_slaves"))
	aliases := lo.MapEntries(v.Records, func(
		alias string,
		c slave.Slave,
	) (slave.Key, string) {
		return c.Key, alias
	})

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		whereDB = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(whereDB.Run)
		MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: whereDB}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: whereDB}))
		stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: whereDB}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: whereDB}))
		racks := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:      whereDB,
			HostKey: 1,
			Status:  stat,
			DriverRole: MustSucceed(
				racktestutil.CreateDriverRole(ctx, whereDB, roleSvc),
			),
		}))
		MustOpen(master.OpenService(ctx, master.ServiceConfig{DB: whereDB}))
		svc = MustOpen(slave.OpenService(ctx, slave.ServiceConfig{DB: whereDB}))
		rackKeys := lo.Uniq(lo.MapToSlice(v.Records, func(
			_ string,
			c slave.Slave,
		) rack.Key {
			return c.Rack
		}))
		for _, k := range rackKeys {
			r := rack.Rack{Key: k, Name: "Rack"}
			Expect(whereDB.WithTx(ctx, func(tx gorp.Tx) error {
				return racks.NewWriter(tx).Create(ctx, &r)
			})).To(Succeed())
		}
		masters := lo.UniqBy(lo.MapToSlice(v.Records, func(
			_ string,
			c slave.Slave,
		) master.Master {
			return master.Master{
				Key:       c.Master,
				Rack:      c.Rack,
				Name:      "Bus",
				Interface: "eth0",
			}
		}), func(m master.Master) master.Key { return m.Key })
		for _, m := range masters {
			Expect(whereDB.WithTx(ctx, func(tx gorp.Tx) error {
				return gorp.NewCreate[master.Key, master.Master]().Entry(&m).
					Exec(ctx, tx)
			})).To(Succeed())
		}
		for _, c := range v.Records {
			Expect(whereDB.WithTx(ctx, func(tx gorp.Tx) error {
				return gorp.NewCreate[slave.Key, slave.Slave]().Entry(&c).
					Exec(ctx, tx)
			})).To(Succeed())
		}
	})

	DescribeTable(
		"Should match the case",
		func(ctx SpecContext, c vectors.Case[slave.Where]) {
			f := MustSucceed(slave.MatchWhere(c.Where))
			var res []slave.Slave
			Expect(svc.NewRetrieve().
				SkipMissingKeys().
				Where(f).
				Entries(&res).
				Exec(ctx, whereDB)).To(Succeed())
			Expect(lo.Map(res, func(sl slave.Slave, _ int) string {
				return aliases[sl.Key]
			})).To(ConsistOf(c.Expected))
		},
		lo.Map(v.Cases, func(c vectors.Case[slave.Where], _ int) TableEntry {
			return Entry(c.Name, c)
		}),
	)

	It("Should return a path error for an invalid regex", func() {
		var w slave.Where
		Expect(json.Unmarshal(
			[]byte(`{"or":[{"key":{"in":[]}},{"name":{"regex":"(a"}}]}`),
			&w,
		)).To(Succeed())
		Expect(slave.MatchWhere(w)).Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError(
				"or.1.name.regex: error parsing regexp: missing closing ): `(a`: "+
					"validation error",
			),
		))
	})

	It("Should reject a line that sets no operator", func() {
		Expect(slave.MatchWhere(slave.Where{Name: &filter.String{}})).
			Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError(filter.ErrNoOperator),
		))
	})
})
