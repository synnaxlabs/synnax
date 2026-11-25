// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server_test

import (
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/opcua/server"
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
		v       vectors.Vectors[server.Server, server.Where]
		whereDB *gorp.DB
		svc     *server.Service
	)
	v = MustSucceed(vectors.Read[server.Server, server.Where]("opcua_servers"))
	aliases := lo.MapEntries(v.Records, func(
		alias string,
		d server.Server,
	) (server.Key, string) {
		return d.Key, alias
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
		svc = MustOpen(server.OpenService(ctx, server.ServiceConfig{DB: whereDB}))
		rackKeys := lo.Uniq(lo.MapToSlice(v.Records, func(
			_ string,
			d server.Server,
		) rack.Key {
			return d.Rack
		}))
		for _, k := range rackKeys {
			r := rack.Rack{Key: k, Name: "Rack"}
			Expect(whereDB.WithTx(ctx, func(tx gorp.Tx) error {
				return racks.NewWriter(tx).Create(ctx, &r)
			})).To(Succeed())
		}
		for _, d := range v.Records {
			Expect(whereDB.WithTx(ctx, func(tx gorp.Tx) error {
				return gorp.NewCreate[server.Key, server.Server]().Entry(&d).
					Exec(ctx, tx)
			})).To(Succeed())
		}
	})

	DescribeTable(
		"Should match the case",
		func(ctx SpecContext, c vectors.Case[server.Where]) {
			f := MustSucceed(server.MatchWhere(c.Where))
			var res []server.Server
			Expect(svc.NewRetrieve().
				SkipMissingKeys().
				Where(f).
				Entries(&res).
				Exec(ctx, whereDB)).To(Succeed())
			Expect(lo.Map(res, func(d server.Server, _ int) string {
				return aliases[d.Key]
			})).To(ConsistOf(c.Expected))
		},
		lo.Map(v.Cases, func(c vectors.Case[server.Where], _ int) TableEntry {
			return Entry(c.Name, c)
		}),
	)

	It("Should return a path error for an invalid regex", func() {
		var w server.Where
		Expect(json.Unmarshal(
			[]byte(`{"or":[{"key":{"in":[]}},{"name":{"regex":"(a"}}]}`),
			&w,
		)).To(Succeed())
		Expect(server.MatchWhere(w)).Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError(
				"or.1.name.regex: error parsing regexp: missing closing ): `(a`: "+
					"validation error",
			),
		))
	})

	It("Should reject a line that sets no operator", func() {
		Expect(server.MatchWhere(server.Where{Name: &filter.String{}})).
			Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError(filter.ErrNoOperator),
		))
	})
})
