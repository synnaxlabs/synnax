// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package broker_test

import (
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/broker"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	secrettestutil "github.com/synnaxlabs/synnax/pkg/service/secret/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/testutil/vectors"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Where", Ordered, func() {
	var (
		v       vectors.Vectors[broker.Broker, broker.Where]
		whereDB *gorp.DB
		svc     *broker.Service
	)
	v = MustSucceed(vectors.Read[broker.Broker, broker.Where]("mqtt_brokers"))
	aliases := lo.MapEntries(v.Records, func(
		alias string,
		d broker.Broker,
	) (broker.Key, string) {
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
		secrets := secrettestutil.OpenService(
			ctx,
			whereDB,
			secret.ServiceConfig{EncryptionKey: secretsKey},
		)
		svc = MustOpen(broker.OpenService(ctx, broker.ServiceConfig{
			DB:     whereDB,
			Status: stat,
			Secret: secrets,
			Certificate: MustOpen(
				certificate.OpenService(ctx, certificate.ServiceConfig{
					DB:      whereDB,
					Secret:  secrets,
					Status:  stat,
					Rack:    racks.EmbeddedKey,
					HostKey: 1,
				}),
			),
			Rack: racks,
		}))
		rackKeys := lo.Uniq(lo.MapToSlice(v.Records, func(
			_ string,
			d broker.Broker,
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
				return gorp.NewCreate[broker.Key, broker.Broker]().Entry(&d).
					Exec(ctx, tx)
			})).To(Succeed())
		}
	})

	DescribeTable(
		"Should match the case",
		func(ctx SpecContext, c vectors.Case[broker.Where]) {
			f := MustSucceed(broker.MatchWhere(c.Where))
			var res []broker.Broker
			Expect(svc.NewRetrieve().
				SkipMissingKeys().
				Where(f).
				Entries(&res).
				Exec(ctx, whereDB)).To(Succeed())
			Expect(lo.Map(res, func(d broker.Broker, _ int) string {
				return aliases[d.Key]
			})).To(ConsistOf(c.Expected))
		},
		lo.Map(v.Cases, func(c vectors.Case[broker.Where], _ int) TableEntry {
			return Entry(c.Name, c)
		}),
	)

	It("Should return a path error for an invalid regex", func() {
		var w broker.Where
		Expect(json.Unmarshal(
			[]byte(`{"or":[{"key":{"in":[]}},{"name":{"regex":"(a"}}]}`),
			&w,
		)).To(Succeed())
		Expect(broker.MatchWhere(w)).Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError(
				"or.1.name.regex: error parsing regexp: missing closing ): `(a`: "+
					"validation error",
			),
		))
	})

	It("Should reject a line that sets no operator", func() {
		Expect(broker.MatchWhere(broker.Where{Name: &filter.String{}})).
			Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError(filter.ErrNoOperator),
		))
	})
})
