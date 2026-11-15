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
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v1 "github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/v2"
	secret "github.com/synnaxlabs/synnax/pkg/service/secret/versions/v0"
	task "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	"github.com/synnaxlabs/x/errors"
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
			fail := func(context.Context, gorp.Tx, secret.Key, string, []byte) error {
				return errors.New("create failed")
			}
			cfg := v2.TaskConfig{PlainRoutingKey: new("plain")}
			Expect(v2.ResolveTaskConfig(fail, "Pump", &cfg)(ctx, db)).
				To(MatchError("create failed"))
		})
	})

	Describe("NewMigration", func() {
		var db *gorp.DB
		BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })

		// store writes a v1 config and, unless orphan, the task that owns it.
		store := func(ctx SpecContext, name, routingKey string, orphan bool) uuid.UUID {
			GinkgoHelper()
			var old v1.TaskConfig
			old.Key, old.RoutingKey = uuid.New(), routingKey
			Expect(gorp.WrapWriter[uuid.UUID, v1.TaskConfig](db).Set(ctx, old)).
				To(Succeed())
			if !orphan {
				Expect(gorp.WrapWriter[task.Key, task.Task](db).Set(ctx, task.Task{
					Key:  old.Key,
					Name: name,
					Type: "pagerduty_alert",
				})).To(Succeed())
			}
			return old.Key
		}
		open := func(ctx SpecContext) (*gorp.Table[uuid.UUID, v2.TaskConfig], error) {
			return gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v2.TaskConfig]{
				DB:         db,
				Migrations: []gorp.Migration{v2.NewMigration(create)},
			})
		}
		read := func(ctx SpecContext, key uuid.UUID) v2.TaskConfig {
			GinkgoHelper()
			var cfg v2.TaskConfig
			Expect(gorp.NewRetrieve[uuid.UUID, v2.TaskConfig]().
				Where(gorp.MatchKeys[uuid.UUID, v2.TaskConfig](key)).
				Entry(&cfg).
				Exec(ctx, db)).To(Succeed())
			return cfg
		}

		It("Should move each stored routing key into a secret named after its task",
			func(ctx SpecContext) {
				withKey := store(
					ctx,
					"Pump alerts",
					"0123456789abcdef0123456789abcdef",
					false,
				)
				withoutKey := store(ctx, "Draft", "", false)
				MustOpen(open(ctx))
				Expect(made).To(Equal([]created{{
					name:  "Pump alerts routing key",
					value: "0123456789abcdef0123456789abcdef",
				}}))
				migrated := read(ctx, withKey)
				Expect(migrated.RoutingKey).ToNot(BeNil())
				Expect(migrated.PlainRoutingKey).To(BeNil())
				Expect(read(ctx, withoutKey).RoutingKey).To(BeNil())
			},
		)

		It("Should do nothing without stored configs", func(ctx SpecContext) {
			MustOpen(open(ctx))
			Expect(made).To(BeEmpty())
		})

		It("Should fail when a config has no task", func(ctx SpecContext) {
			store(ctx, "", "0123456789abcdef0123456789abcdef", true)
			Expect(open(ctx)).Error().
				To(MatchError(ContainSubstring("tasks of PagerDuty configs")))
		})

		It("Should fail with the config key when a secret create fails",
			func(ctx SpecContext) {
				key := store(ctx, "Pump", "0123456789abcdef0123456789abcdef", false)
				create = func(
					context.Context,
					gorp.Tx,
					secret.Key,
					string,
					[]byte,
				) error {
					return errors.New("create failed")
				}
				Expect(open(ctx)).Error().To(MatchError(ContainSubstring(
					fmt.Sprintf("config %s: create failed", key),
				)))
			},
		)
	})
})
