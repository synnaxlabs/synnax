// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pagerduty_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	pd "github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	v1 "github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/v2"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	secrettestutil "github.com/synnaxlabs/synnax/pkg/service/secret/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

const plainKey = "0123456789abcdef0123456789abcdef"

// routingKeyOf returns the routing key that the config data holds.
func routingKeyOf(data msgpack.EncodedJSON) secret.Key {
	GinkgoHelper()
	Expect(data).ToNot(HaveKey("plain_routing_key"))
	key, ok := data["routing_key"].(string)
	Expect(ok).To(BeTrue())
	return uuid.MustParse(key)
}

// expectSecret asserts that the secret with key is named name and holds the plain
// routing key.
func expectSecret(
	ctx SpecContext,
	db *gorp.DB,
	svc *secret.Service,
	key secret.Key,
	name string,
) {
	GinkgoHelper()
	var s secret.Secret
	Expect(svc.NewRetrieve().
		Where(secret.MatchKeys(key)).
		Entry(&s).
		Exec(ctx, db)).To(Succeed())
	Expect(s.Base().Name).To(Equal(name))
	Expect(svc.Reveal(ctx, db, key)).
		To(Equal(map[secret.Key][]byte{key: []byte(plainKey)}))
}

var _ = Describe("Service", func() {
	var (
		secrets *secret.Service
		svc     *pd.Service
		tasks   *tasktestutil.Tasks
		db      *gorp.DB
	)
	BeforeEach(func(ctx SpecContext) {
		// The suite's DB holds a task table of its own.
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		secrets = secrettestutil.OpenService(ctx, db)
		svc = MustOpen(pd.OpenService(ctx, pd.ServiceConfig{
			DB:     db,
			Secret: secrets,
			Rack:   secrettestutil.CoreRack,
		}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:     db,
			Stores: svc.Stores(),
		}))
	})

	createRoutingKey := func(ctx SpecContext) secret.Key {
		GinkgoHelper()
		s := secret.Secret{Variant: secret.StoredSecret{
			Base: secret.Base{Name: "Routing key"},
		}}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return secrets.NewWriter(tx).Set(ctx, &s, []byte(plainKey))
		})).To(Succeed())
		return s.Base().Key
	}

	Describe("OpenService", func() {
		DescribeTable("Should reject a config missing a dependency", func(
			ctx SpecContext,
			cfg pd.ServiceConfig,
			field string,
		) {
			Expect(pd.OpenService(ctx, cfg)).Error().
				To(MatchError(ContainSubstring(field + ": must be non-nil")))
		},
			Entry("db", pd.ServiceConfig{Secret: &secret.Service{}, Rack: 1}, "db"),
			Entry("secret", pd.ServiceConfig{DB: &gorp.DB{}, Rack: 1}, "secret"),
		)

		It("Should reject a config without the Core's rack", func(ctx SpecContext) {
			Expect(pd.OpenService(ctx, pd.ServiceConfig{
				DB:     &gorp.DB{},
				Secret: &secret.Service{},
			})).Error().To(MatchError(ContainSubstring("rack: must be non-zero")))
		})

		It("Should lift a stored v1 config and stage its plain routing key", func(
			ctx SpecContext,
		) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(db.Run)
			old := v1.TaskConfig{Key: uuid.New(), RoutingKey: plainKey}
			old.Alerts = []v1.Alert{{Key: "a", Status: "s"}}
			Expect(gorp.WrapWriter[uuid.UUID, v1.TaskConfig](db).Set(ctx, old)).
				To(Succeed())
			MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
			svc := MustOpen(pd.OpenService(ctx, pd.ServiceConfig{
				DB:     db,
				Secret: secrettestutil.OpenService(ctx, db),
				Rack:   secrettestutil.CoreRack,
			}))
			data := MustSucceed(svc.Alert.Read(ctx, db, old.Key))
			Expect(data).ToNot(HaveKey("routing_key"))
			staged, closer := MustSucceed2(
				db.Get(v2.PlainRoutingKeyKVKey(old.Key)),
			)
			Expect(string(staged)).To(Equal(plainKey))
			Expect(closer.Close()).To(Succeed())
			Expect(data["alerts"]).To(HaveLen(1))
		})

		It("Should skip the lift on a store that Core v0.62.0-rc.1 migrated", func(
			ctx SpecContext,
		) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(db.Run)
			MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
			secrets := secrettestutil.OpenService(ctx, db)
			s := secret.Secret{Variant: secret.StoredSecret{
				Base: secret.Base{Name: "Pump alerts routing key"},
			}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return secrets.NewWriter(tx).Set(ctx, &s, []byte(plainKey))
			})).To(Succeed())
			var cfg v2.TaskConfig
			cfg.Key, cfg.RoutingKey = uuid.New(), new(s.Base().Key)
			Expect(gorp.WrapWriter[uuid.UUID, v2.TaskConfig](db).Set(ctx, cfg)).
				To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:        db,
				Namespace: "pagerduty_alert_config",
				Migrations: lo.Map(
					[]string{
						"v61_status_owner_keys",
						"v61_pagerduty_routing_key_secret",
					},
					func(key string, _ int) gorp.Migration {
						return migrate.New(
							key,
							func(context.Context, gorp.Tx, alamos.Instrumentation) error {
								return nil
							},
						)
					},
				),
			})).To(Succeed())
			svc := MustOpen(pd.OpenService(ctx, pd.ServiceConfig{
				DB:     db,
				Secret: secrets,
				Rack:   secrettestutil.CoreRack,
			}))
			data := MustSucceed(svc.Alert.Read(ctx, db, cfg.Key))
			Expect(routingKeyOf(data)).To(Equal(s.Base().Key))
			Expect(db.Get(v2.PlainRoutingKeyKVKey(cfg.Key))).Error().
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Stores", func() {
		It("Should expose the alert store", func() {
			stores := svc.Stores()
			types := make([]string, len(stores))
			for i, s := range stores {
				types[i] = s.Type()
			}
			Expect(types).To(ConsistOf("pagerduty_alert"))
		})
	})

	Describe("Prepare", func() {
		It("Should store a decoded alert config under the given key", func(
			ctx SpecContext,
		) {
			Expect(tasks.CreateStatus(ctx, "status-1")).To(Succeed())
			routingKey := createRoutingKey(ctx)
			key := MustSucceed(tasks.Create(ctx, svc.Alert, msgpack.EncodedJSON{
				"routing_key": routingKey.String(),
				"auto_start":  true,
				"alerts": []any{map[string]any{
					"key":    "alert-1",
					"status": "status-1",
				}},
			}))
			data := MustSucceed(svc.Alert.Read(ctx, db, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(routingKeyOf(data)).To(Equal(routingKey))
			Expect(data["auto_start"]).To(BeTrue())
		})

		It("Should move a plain routing key into a secret named after the task",
			func(ctx SpecContext) {
				key := MustSucceed(tasks.Create(ctx, svc.Alert, msgpack.EncodedJSON{
					"plain_routing_key": plainKey,
				}))
				data := MustSucceed(svc.Alert.Read(ctx, db, key))
				expectSecret(
					ctx,
					db,
					secrets,
					routingKeyOf(data),
					"pagerduty_alert routing key",
				)
			},
		)

		It("Should create the secret only when the record is written", func(
			ctx SpecContext,
		) {
			rec := MustSucceed(svc.Alert.Prepare(
				ctx,
				db,
				uuid.New(),
				"Pump alerts",
				msgpack.EncodedJSON{"plain_routing_key": plainKey},
			))
			routingKey := routingKeyOf(MustSucceed(rec.Data()))
			Expect(secrets.NewRetrieve().
				Where(secret.MatchKeys(routingKey)).
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return rec.Write(ctx, tx)
			})).To(MatchError(validate.ErrValidation))
			Expect(secrets.NewRetrieve().
				Where(secret.MatchKeys(routingKey)).
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Normalize", func() {
		DescribeTable("Should store the routing key of an old shape as a secret",
			func(ctx SpecContext, version imex.Version, blob msgpack.EncodedJSON) {
				normalized := MustSucceed(svc.Alert.Normalize(ctx, db, version, blob))
				Expect(normalized).To(HaveKeyWithValue("plain_routing_key", plainKey))
				Expect(normalized).ToNot(HaveKey("routing_key"))
				key := MustSucceed(tasks.Create(ctx, svc.Alert, normalized))
				data := MustSucceed(svc.Alert.Read(ctx, db, key))
				expectSecret(
					ctx,
					db,
					secrets,
					routingKeyOf(data),
					"pagerduty_alert routing key",
				)
			},
			Entry("a Console-era blob", imex.Version(0), msgpack.EncodedJSON{
				"routingKey": plainKey,
				"alerts":     []any{map[string]any{"key": "a", "enabled": true}},
			}),
			Entry("a v1 export", imex.Version(1), msgpack.EncodedJSON{
				"routing_key": plainKey,
				"alerts":      []any{map[string]any{"key": "a"}},
			}),
		)
	})

	Describe("Routing key references", func() {
		It("Should list the config as a use of its routing key", func(
			ctx SpecContext,
		) {
			routingKey := createRoutingKey(ctx)
			key := MustSucceed(tasks.Create(ctx, svc.Alert, msgpack.EncodedJSON{
				"routing_key": routingKey.String(),
			}))
			Expect(secrets.Uses(ctx, db, routingKey)).To(Equal([]resource.ID{{
				Type: resource.Type(config.TableName(pd.AlertTaskType)),
				Key:  key.String(),
			}}))
		})

		It("Should refuse to delete a routing key that a task uses", func(
			ctx SpecContext,
		) {
			routingKey := createRoutingKey(ctx)
			key := MustSucceed(tasks.Create(ctx, svc.Alert, msgpack.EncodedJSON{
				"routing_key": routingKey.String(),
			}))
			deleteKey := func(ctx context.Context) error {
				return db.WithTx(ctx, func(tx gorp.Tx) error {
					return secrets.NewWriter(tx).Delete(ctx, routingKey)
				})
			}
			Expect(deleteKey(ctx)).To(MatchError(gorp.RestrictError{
				Target:    gorp.NewTarget("secret", routingKey),
				Holder:    gorp.NewTarget(config.TableName(pd.AlertTaskType), key),
				Reference: "routing_key",
			}))
			Expect(svc.Alert.Delete(ctx, db, key)).To(Succeed())
			Expect(deleteKey(ctx)).To(Succeed())
		})

		It("Should reject a routing key that does not exist", func(ctx SpecContext) {
			Expect(tasks.Create(ctx, svc.Alert, msgpack.EncodedJSON{
				"routing_key": uuid.New().String(),
			})).Error().To(MatchError(ContainSubstring("routing_key")))
		})
	})
})
