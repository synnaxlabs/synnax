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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	pd "github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	v1 "github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	secrettestutil "github.com/synnaxlabs/synnax/pkg/service/secret/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

const plainKey = "0123456789abcdef0123456789abcdef"

// routingKeyOf returns the routing key that the config data holds.
func routingKeyOf(data msgpack.EncodedJSON) secret.Key {
	GinkgoHelper()
	Expect(data).ToNot(HaveKey("plain_routing_key"))
	Expect(data["routing_key"]).To(BeAssignableToTypeOf(""))
	return uuid.MustParse(data["routing_key"].(string))
}

// expectSecret asserts that the secret with key is named name and holds the plain
// routing key.
func expectSecret(ctx SpecContext, svc *secret.Service, key secret.Key, name string) {
	GinkgoHelper()
	var s secret.Secret
	Expect(svc.NewRetrieve().
		Where(secret.MatchKeys(key)).
		Entry(&s).
		Exec(ctx, nil)).To(Succeed())
	Expect(s.Base().Name).To(Equal(name))
	Expect(svc.Reveal(ctx, nil, key)).
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
		secrets = secrettestutil.OpenService(ctx, db)
		svc = MustOpen(pd.OpenService(ctx, pd.ServiceConfig{
			DB:     db,
			Secret: secrets,
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
		Expect(secrets.NewWriter(nil).Create(ctx, &s, []byte(plainKey))).To(Succeed())
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
			Entry("db", pd.ServiceConfig{Secret: &secret.Service{}}, "db"),
			Entry("secret", pd.ServiceConfig{DB: &gorp.DB{}}, "secret"),
		)

		It("Should move each routing key of a stored v1 config into a secret", func(
			ctx SpecContext,
		) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			t := task.Task{Key: uuid.New(), Name: "Pump alerts", Type: pd.AlertTaskType}
			Expect(gorp.NewCreate[task.Key, task.Task]().Entry(&t).Exec(ctx, db)).
				To(Succeed())
			old := v1.TaskConfig{Key: t.Key, RoutingKey: plainKey}
			old.Alerts = []v1.Alert{{Key: "a", Status: "s"}}
			Expect(gorp.WrapWriter[uuid.UUID, v1.TaskConfig](db).Set(ctx, old)).
				To(Succeed())
			MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
			secrets := secrettestutil.OpenService(ctx, db)
			svc := MustOpen(pd.OpenService(ctx, pd.ServiceConfig{
				DB:     db,
				Secret: secrets,
			}))
			data := MustSucceed(svc.Alert.Read(ctx, nil, old.Key))
			expectSecret(ctx, secrets, routingKeyOf(data), "Pump alerts routing key")
			Expect(data["alerts"]).To(HaveLen(1))
		})
	})

	Describe("Stores", func() {
		It("Should expose the alert store", func() {
			types := []string{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
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
			data := MustSucceed(svc.Alert.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(routingKeyOf(data)).To(Equal(routingKey))
			Expect(data["auto_start"]).To(BeTrue())
		})

		It("Should move a plain routing key into a secret in the same transaction",
			func(ctx SpecContext) {
				tx := db.OpenTx()
				rec := MustSucceed(svc.Alert.Prepare(
					ctx,
					tx,
					uuid.New(),
					"Pump alerts",
					msgpack.EncodedJSON{"plain_routing_key": plainKey},
				))
				routingKey := routingKeyOf(rec.Data())
				Expect(secrets.NewRetrieve().
					Where(secret.MatchKeys(routingKey)).
					Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
				Expect(tx.Commit(ctx)).To(Succeed())
				Expect(tx.Close()).To(Succeed())
				expectSecret(ctx, secrets, routingKey, "Pump alerts routing key")
			},
		)

		It("Should create no secret when the transaction aborts", func(
			ctx SpecContext,
		) {
			tx := db.OpenTx()
			rec := MustSucceed(svc.Alert.Prepare(
				ctx,
				tx,
				uuid.New(),
				"Aborted",
				msgpack.EncodedJSON{"plain_routing_key": plainKey},
			))
			Expect(tx.Close()).To(Succeed())
			Expect(secrets.NewRetrieve().
				Where(secret.MatchKeys(routingKeyOf(rec.Data()))).
				Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Normalize", func() {
		DescribeTable("Should store the routing key of an old shape as a secret",
			func(ctx SpecContext, version imex.Version, blob msgpack.EncodedJSON) {
				normalized := MustSucceed(svc.Alert.Normalize(ctx, nil, version, blob))
				Expect(normalized).To(HaveKeyWithValue("plain_routing_key", plainKey))
				Expect(normalized).ToNot(HaveKey("routing_key"))
				key := MustSucceed(tasks.Create(ctx, svc.Alert, normalized))
				data := MustSucceed(svc.Alert.Read(ctx, nil, key))
				expectSecret(
					ctx,
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
			Expect(secrets.Uses(ctx, nil, routingKey)).To(Equal([]resource.ID{{
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
			Expect(secrets.NewWriter(nil).Delete(ctx, routingKey)).To(
				MatchError(gorp.RestrictError{
					Target:    gorp.NewTarget("secret", routingKey),
					Holder:    gorp.NewTarget(config.TableName(pd.AlertTaskType), key),
					Reference: "routing_key",
				}),
			)
			Expect(svc.Alert.Delete(ctx, nil, key)).To(Succeed())
			Expect(secrets.NewWriter(nil).Delete(ctx, routingKey)).To(Succeed())
		})

		It("Should reject a routing key that does not exist", func(ctx SpecContext) {
			Expect(tasks.Create(ctx, svc.Alert, msgpack.EncodedJSON{
				"routing_key": uuid.New().String(),
			})).Error().To(MatchError(ContainSubstring("routing_key")))
		})
	})
})
