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
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/labjack"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/synnax/pkg/service/testutil/vectors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Where", Ordered, func() {
	var (
		v   vectors.Vectors[task.Task, task.Where]
		db  *gorp.DB
		svc *task.Service
	)
	v = MustSucceed(vectors.Read[task.Task, task.Where]("tasks"))
	aliases := lo.MapEntries(v.Records, func(
		alias string,
		t task.Task,
	) (task.Key, string) {
		return t.Key, alias
	})

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB: db,
		}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB: db,
		}))
		stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB: db,
		}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
			DB: db,
		}))
		racks := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:         db,
			HostKey:    1,
			Status:     stat,
			DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roleSvc)),
		}))
		lj := MustOpen(labjack.OpenService(ctx, labjack.ServiceConfig{
			DB:     db,
			Status: stat,
		}))
		svc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
			DB:      db,
			Rack:    racks,
			Status:  stat,
			ImEx:    imex.NewService(),
			Configs: MustSucceed(config.NewRegistry(lj.Stores()...)),
		}))
		rackKeys := lo.Uniq(lo.MapToSlice(v.Records, func(
			_ string,
			t task.Task,
		) rack.Key {
			return t.Rack
		}))
		for _, k := range rackKeys {
			r := rack.Rack{Key: k, Name: "Rack"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return racks.NewWriter(tx).Create(ctx, &r)
			})).To(Succeed())
		}
		for _, t := range v.Records {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &t)
			})).To(Succeed())
		}
	})

	DescribeTable(
		"Should match the case",
		func(ctx SpecContext, c vectors.Case[task.Where]) {
			f := MustSucceed(task.MatchWhere(c.Where))
			var res []task.Task
			Expect(svc.NewRetrieve().
				SkipMissingKeys().
				Where(f).
				Entries(&res).
				Exec(ctx, db)).To(Succeed())
			Expect(lo.Map(res, func(t task.Task, _ int) string {
				return aliases[t.Key]
			})).To(ConsistOf(c.Expected))
		},
		lo.Map(v.Cases, func(c vectors.Case[task.Where], _ int) TableEntry {
			return Entry(c.Name, c)
		}),
	)

	It("Should return a path error for an invalid regex", func() {
		var w task.Where
		Expect(json.Unmarshal(
			[]byte(`{"or":[{"key":{"in":[]}},{"type":{"regex":"(a"}}]}`),
			&w,
		)).To(Succeed())
		Expect(task.MatchWhere(w)).Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError(
				"or.1.type.regex: error parsing regexp: missing closing ): `(a`: validation error",
			),
		))
	})
})
