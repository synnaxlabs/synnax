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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	pd "github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	v1 "github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	secrettestutil "github.com/synnaxlabs/synnax/pkg/service/secret/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

const plainKey = "0123456789abcdef0123456789abcdef"

// storeRow stores the row of an alert task named name on testRack and returns its key.
func storeRow(ctx SpecContext, db *gorp.DB, name string) task.Key {
	GinkgoHelper()
	t := task.Task{Key: uuid.New(), Rack: testRack, Name: name, Type: pd.AlertTaskType}
	Expect(gorp.NewCreate[task.Key, task.Task]().Entry(&t).Exec(ctx, db)).
		To(Succeed())
	return t.Key
}

// routingKeyOf returns the routing key that the config data holds.
func routingKeyOf(data msgpack.EncodedJSON) secret.Key {
	GinkgoHelper()
	Expect(data).ToNot(HaveKey("plain_routing_key"))
	Expect(data["routing_key"]).To(BeAssignableToTypeOf(""))
	return uuid.MustParse(data["routing_key"].(string))
}

// expectSecret asserts that the secret with key is named name and that a task on
// testRack reads value from it.
func expectSecret(ctx SpecContext, svc *secret.Service, key secret.Key, name string) {
	GinkgoHelper()
	var s secret.Secret
	Expect(svc.NewRetrieve().
		Where(secret.MatchKeys(key)).
		Entry(&s).
		Exec(ctx, nil)).To(Succeed())
	Expect(s.Base().Name).To(Equal(name))
	Expect(svc.Resolve(ctx, nil, testRack.OntologyID(), uuid.Nil(), key)).
		To(Equal(map[secret.Key][]byte{key: []byte(plainKey)}))
}

var _ = Describe("Service", func() {
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
			otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
			g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
				DB:       db,
				Ontology: otg,
			}))
			secrets := secrettestutil.OpenService(ctx, db, otg, g, pd.FindSecretUses)
			var old v1.TaskConfig
			old.Key, old.RoutingKey = storeRow(ctx, db, "Pump alerts"), plainKey
			old.Alerts = []v1.Alert{{Key: "a", Status: "s"}}
			Expect(gorp.WrapWriter[uuid.UUID, v1.TaskConfig](db).Set(ctx, old)).
				To(Succeed())
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
			for _, s := range alertSvc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf("pagerduty_alert"))
		})
	})

	Describe("Write", func() {
		It("Should store a decoded alert config under the given key", func(
			ctx SpecContext,
		) {
			key, routingKey := uuid.New(), createRoutingKey(ctx, plainKey)
			MustSucceed(alertSvc.Alert.Write(ctx, nil, key, "", msgpack.EncodedJSON{
				"routing_key": routingKey.String(),
				"auto_start":  true,
				"alerts": []any{map[string]any{
					"key":    "alert-1",
					"status": "status-1",
				}},
			}))
			data := MustSucceed(alertSvc.Alert.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(routingKeyOf(data)).To(Equal(routingKey))
			Expect(data["auto_start"]).To(BeTrue())
		})

		It("Should move a plain routing key into a secret in the same transaction",
			func(ctx SpecContext) {
				key := storeRow(ctx, db, "Pump alerts")
				tx := db.OpenTx()
				data := MustSucceed(alertSvc.Alert.Write(
					ctx,
					tx,
					key,
					"Pump alerts",
					msgpack.EncodedJSON{"plain_routing_key": plainKey},
				))
				routingKey := routingKeyOf(data)
				Expect(secretSvc.NewRetrieve().
					Where(secret.MatchKeys(routingKey)).
					Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
				Expect(tx.Commit(ctx)).To(Succeed())
				Expect(tx.Close()).To(Succeed())
				expectSecret(ctx, secretSvc, routingKey, "Pump alerts routing key")
				Expect(routingKeyOf(MustSucceed(alertSvc.Alert.Read(ctx, nil, key)))).
					To(Equal(routingKey))
			},
		)

		It("Should create no secret when the write's transaction aborts", func(
			ctx SpecContext,
		) {
			tx := db.OpenTx()
			data := MustSucceed(alertSvc.Alert.Write(
				ctx,
				tx,
				uuid.New(),
				"Aborted",
				msgpack.EncodedJSON{"plain_routing_key": plainKey},
			))
			Expect(tx.Close()).To(Succeed())
			Expect(secretSvc.NewRetrieve().
				Where(secret.MatchKeys(routingKeyOf(data))).
				Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Normalize", func() {
		DescribeTable("Should store the routing key of an old shape as a secret",
			func(ctx SpecContext, version imex.Version, blob msgpack.EncodedJSON) {
				normalized := MustSucceed(alertSvc.Alert.Normalize(ctx, version, blob))
				Expect(normalized).To(HaveKeyWithValue("plain_routing_key", plainKey))
				Expect(normalized).ToNot(HaveKey("routing_key"))
				key := storeRow(ctx, db, "Imported")
				data := MustSucceed(alertSvc.Alert.Write(
					ctx,
					nil,
					key,
					"Imported",
					normalized,
				))
				expectSecret(ctx, secretSvc, routingKeyOf(data), "Imported routing key")
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

	Describe("FindSecretUses", func() {
		It("Should return each task whose routing key is one of the secrets", func(
			ctx SpecContext,
		) {
			routingKey := createRoutingKey(ctx, plainKey)
			t := storeTask(ctx, "Pump alerts", pd.TaskConfig{RoutingKey: &routingKey})
			createRoutingKey(ctx, plainKey)
			Expect(pd.FindSecretUses(ctx, db, []secret.Key{routingKey})).
				To(Equal([]secret.Use{{
					Resource: t.OntologyID(),
					Name:     "Pump alerts",
					Racks:    []ontology.ID{testRack.OntologyID()},
					Secret:   routingKey,
				}}))
		})

		It("Should give a task on no rack no racks", func(ctx SpecContext) {
			routingKey := createRoutingKey(ctx, plainKey)
			t := task.Task{Key: uuid.New(), Name: "Draft", Type: pd.AlertTaskType}
			Expect(gorp.NewCreate[task.Key, task.Task]().Entry(&t).Exec(ctx, db)).
				To(Succeed())
			MustSucceed(alertSvc.Alert.Write(ctx, nil, t.Key, t.Name, encodeConfig(
				pd.TaskConfig{RoutingKey: &routingKey},
			)))
			uses := MustSucceed(pd.FindSecretUses(ctx, db, []secret.Key{routingKey}))
			Expect(uses).To(HaveLen(1))
			Expect(uses[0].Racks).To(BeEmpty())
		})

		It("Should return no uses for a secret that no task uses", func(
			ctx SpecContext,
		) {
			Expect(pd.FindSecretUses(ctx, db, []secret.Key{uuid.New()})).
				To(BeEmpty())
		})
	})

	Describe("Secret delete", func() {
		It("Should refuse to delete a routing key that a task uses", func(
			ctx SpecContext,
		) {
			routingKey := createRoutingKey(ctx, plainKey)
			t := storeTask(ctx, "Pump alerts", pd.TaskConfig{RoutingKey: &routingKey})
			Expect(secretSvc.NewWriter(nil).Delete(ctx, routingKey)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					`cannot delete secret "Routing key" because task "Pump alerts" uses it`,
				)),
			))
			Expect(alertSvc.Alert.Delete(ctx, nil, t.Key)).To(Succeed())
			Expect(secretSvc.NewWriter(nil).Delete(ctx, routingKey)).To(Succeed())
		})
	})
})
