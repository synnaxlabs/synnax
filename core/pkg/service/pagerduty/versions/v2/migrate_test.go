// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v1 "github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/v2"
	secret "github.com/synnaxlabs/synnax/pkg/service/secret/versions/v0"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// created is a secret that a CreateSecret call made.
type created struct {
	name  string
	value string
}

var _ = Describe("Migrate", func() {
	var (
		made   []created
		keys   []secret.Key
		create v2.CreateSecret
	)
	BeforeEach(func() {
		made, keys = nil, nil
		create = func(
			_ context.Context,
			tx gorp.Tx,
			key secret.Key,
			name string,
			value []byte,
		) error {
			GinkgoHelper()
			Expect(tx).ToNot(BeNil())
			made = append(made, created{name: name, value: string(value)})
			keys = append(keys, key)
			return nil
		}
	})

	Describe("MigrateTaskConfig", func() {
		It("Should move a routing key to the plain routing key", func(ctx SpecContext) {
			var old v1.TaskConfig
			old.Key, old.AutoStart = uuid.New(), true
			old.RoutingKey = "0123456789abcdef0123456789abcdef"
			old.Alerts = []v1.Alert{{Key: "a", Status: "s"}}
			cfg := MustSucceed(v2.MigrateTaskConfig(ctx, old))
			Expect(cfg.Key).To(Equal(old.Key))
			Expect(cfg.AutoStart).To(BeTrue())
			Expect(cfg.Alerts).To(Equal(old.Alerts))
			Expect(cfg.RoutingKey).To(BeNil())
			Expect(cfg.PlainRoutingKey).To(HaveValue(Equal(old.RoutingKey)))
		})

		It("Should leave both keys empty without a routing key", func(ctx SpecContext) {
			cfg := MustSucceed(v2.MigrateTaskConfig(ctx, v1.TaskConfig{}))
			Expect(cfg.RoutingKey).To(BeNil())
			Expect(cfg.PlainRoutingKey).To(BeNil())
		})
	})

	Describe("ResolveTaskConfig", func() {
		It("Should point the routing key at a secret that its write creates",
			func(ctx SpecContext) {
				db := DeferClose(gorp.Wrap(memkv.New()))
				StartRun(db.Run)
				cfg := v2.TaskConfig{PlainRoutingKey: new("plain")}
				write := v2.ResolveTaskConfig(create, "Pump", &cfg)
				Expect(made).To(BeEmpty())
				Expect(cfg.RoutingKey).ToNot(BeNil())
				Expect(cfg.PlainRoutingKey).To(BeNil())
				Expect(write(ctx, db)).To(Succeed())
				Expect(
					made,
				).To(Equal([]created{{name: "Pump routing key", value: "plain"}}))
				Expect(keys).To(Equal([]secret.Key{*cfg.RoutingKey}))
			},
		)

		It("Should return no write for a config without a plain routing key", func() {
			key := uuid.New()
			cfg := v2.TaskConfig{RoutingKey: &key}
			Expect(v2.ResolveTaskConfig(create, "Pump", &cfg)).To(BeNil())
			Expect(cfg.RoutingKey).To(HaveValue(Equal(key)))
		})

		It("Should return the error of the secret create from its write", func(
			ctx SpecContext,
		) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(db.Run)
			fail := func(context.Context, gorp.Tx, secret.Key, string, []byte) error {
				return errors.New("create failed")
			}
			cfg := v2.TaskConfig{PlainRoutingKey: new("plain")}
			Expect(v2.ResolveTaskConfig(fail, "Pump", &cfg)(ctx, db)).
				To(MatchError("create failed"))
		})
	})

	Describe("PlainRoutingKeyKVKey", func() {
		It("Should return the prefix followed by the config key", func() {
			key := uuid.MustParse("0d3b5a0e-6f0c-4f0e-9a43-6a1f6d2f3c11")
			Expect(string(v2.PlainRoutingKeyKVKey(key))).To(Equal(
				"sy_pagerduty_plain_routing_key/0d3b5a0e-6f0c-4f0e-9a43-6a1f6d2f3c11",
			))
		})
	})

	Describe("Migration", func() {
		It("Should lift each stored v1 config and stage its plain routing key", func(
			ctx SpecContext,
		) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(db.Run)
			var withKey, withoutKey v1.TaskConfig
			withKey.Key, withoutKey.Key = uuid.New(), uuid.New()
			withKey.RoutingKey = "0123456789abcdef0123456789abcdef"
			withKey.Alerts = []v1.Alert{{Key: "a", Status: "s"}}
			Expect(gorp.WrapWriter[uuid.UUID, v1.TaskConfig](db).
				Set(ctx, withKey, withoutKey)).To(Succeed())
			table := MustOpen(gorp.OpenTable(
				ctx,
				gorp.TableConfig[uuid.UUID, v2.TaskConfig]{
					DB:         db,
					Migrations: []gorp.Migration{v2.Migration},
				},
			))
			var cfgs []v2.TaskConfig
			Expect(table.NewRetrieve().
				Where(gorp.MatchKeys[uuid.UUID, v2.TaskConfig](
					withKey.Key,
					withoutKey.Key,
				)).
				Entries(&cfgs).
				Exec(ctx, db)).To(Succeed())
			Expect(cfgs[0].Alerts).To(Equal(withKey.Alerts))
			for _, cfg := range cfgs {
				Expect(cfg.RoutingKey).To(BeNil())
				Expect(cfg.PlainRoutingKey).To(BeNil())
			}
			staged, closer := MustSucceed2(
				db.Get(v2.PlainRoutingKeyKVKey(withKey.Key)),
			)
			Expect(string(staged)).To(Equal(withKey.RoutingKey))
			Expect(closer.Close()).To(Succeed())
			Expect(db.Get(v2.PlainRoutingKeyKVKey(withoutKey.Key))).Error().
				To(MatchError(query.ErrNotFound))
		})
	})
})
