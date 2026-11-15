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
	"bytes"
	"context"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	pdversions "github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	secrettestutil "github.com/synnaxlabs/synnax/pkg/service/secret/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

// stack is a task service over an in-memory database, holding the PagerDuty config
// store and its own ImEx registry, plus a rack to create tasks on.
type stack struct {
	db     *gorp.DB
	task   *task.Service
	imex   *imex.Service
	status *status.Service
	secret *secret.Service
	rack   *rack.Rack
}

// openStack opens a stack whose task service excludes excluded from import and export.
func openStack(ctx context.Context, rackName string, excluded ...string) stack {
	GinkgoHelper()
	db := DeferClose(gorp.Wrap(memkv.New()))
	MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: db,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db,
	}))
	roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
		DB: db,
	}))
	rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:                  db,
		HostKey:             1,
		Status:              statusSvc,
		HealthCheckInterval: 10 * telem.Millisecond,
		DriverRole: MustSucceed(
			racktestutil.CreateDriverRole(ctx, db, roleSvc),
		),
	}))
	secretSvc := secrettestutil.OpenService(ctx, db)
	createRoutingKeys(ctx, db, secretSvc, routingKey1)
	pd := MustOpen(pagerduty.OpenService(ctx, pagerduty.ServiceConfig{
		DB:     db,
		Secret: secretSvc,
	}))
	imexSvc := imex.NewService()
	taskSvc := MustOpen(task.OpenService(ctx, task.ServiceConfig{
		DB:           db,
		Rack:         rackSvc,
		Status:       statusSvc,
		ImEx:         imexSvc,
		Configs:      MustSucceed(config.NewRegistry(pd.Stores()...)),
		ImExExcluded: excluded,
	}))
	r := &rack.Rack{Name: rackName}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return rackSvc.NewWriter(tx).Create(ctx, r)
	})).To(Succeed())
	return stack{
		db:     db,
		task:   taskSvc,
		imex:   imexSvc,
		status: statusSvc,
		secret: secretSvc,
		rack:   r,
	}
}

// stampedRoutingKey is the routing key in testdata/import_typed_stamped.json.
const stampedRoutingKey = "5a3c0e7d-91b4-4f2e-8d6a-3b7f0c1e2d48"

var _ = Describe("ImEx", Ordered, func() {
	var (
		db        *gorp.DB
		svc       *task.Service
		testRack  *rack.Rack
		imexSvc   *imex.Service
		statusSvc *status.Service
		secretSvc *secret.Service
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		s := openStack(ctx, "Test Rack")
		db, svc, imexSvc, statusSvc, secretSvc, testRack = s.db, s.task, s.imex,
			s.status, s.secret, s.rack
	})

	Describe("Export", func() {
		It(
			"Should export a task's config flat with version, type, and name",
			func(ctx SpecContext) {
				t := &task.Task{
					Rack: testRack.Key,
					Name: "Exported Task",
					Type: pagerduty.AlertTaskType,
					Config: msgpack.EncodedJSON{
						"routing_key": routingKey1,
						"auto_start":  true,
					},
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, t)
				})).To(Succeed())

				env := MustSucceed(svc.Export(ctx, db, t.ResourceID()))
				Expect(env.Version).To(Equal(pdversions.Latest))
				Expect(env.Type).To(Equal(pagerduty.AlertTaskType))
				Expect(env.Name).To(Equal("Exported Task"))

				var body map[string]any
				Expect(
					json.Unmarshal(MustSucceed(json.Marshal(env)), &body),
				).To(Succeed())
				// The driver reads the file as its config, so config fields sit flat
				// at the top level rather than nested under a "config" key.
				Expect(body).ToNot(HaveKey("config"))
				Expect(body["routing_key"]).To(Equal(routingKey1))
				Expect(body["auto_start"]).To(BeTrue())
				Expect(body["type"]).To(Equal(pagerduty.AlertTaskType))
				Expect(body["name"]).To(Equal("Exported Task"))
				Expect(body["version"]).To(BeEquivalentTo(2))
			},
		)

		It("Should return not found for a missing key", func(ctx SpecContext) {
			id := task.ResourceID(uuid.New())
			Expect(svc.Export(ctx, db, id)).Error().To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Import", func() {
		reimport := func(ctx context.Context, raw []byte) task.Task {
			GinkgoHelper()
			var env imex.Envelope
			Expect(json.Unmarshal(raw, &env)).To(Succeed())
			var id resource.ID
			Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
				id, err = imexSvc.Import(ctx, tx, env, imex.ImportOptions{})
				return err
			})).To(Succeed())
			var imported task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchKeys(uuid.MustParse(id.Key))).
				Entry(&imported).
				Exec(ctx, db)).To(Succeed())
			return imported
		}

		It(
			"Should round-trip an exported task through import",
			func(ctx SpecContext) {
				t := &task.Task{
					Rack: testRack.Key,
					Name: "Round Trip Task",
					Type: pagerduty.AlertTaskType,
					Config: msgpack.EncodedJSON{
						"routing_key": routingKey1,
						"auto_start":  true,
						"alerts": []any{map[string]any{
							"key":      "a1",
							"disabled": true,
						}},
					},
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, t)
				})).To(Succeed())
				env := MustSucceed(svc.Export(ctx, db, t.ResourceID()))
				imported := reimport(ctx, MustSucceed(json.Marshal(env)))

				Expect(imported.Key).ToNot(Equal(t.Key))
				Expect(imported.Rack).To(Equal(rack.Key(0)))
				Expect(imported.Name).To(Equal("Round Trip Task"))
				Expect(imported.Type).To(Equal(pagerduty.AlertTaskType))
				Expect(imported.Config).To(HaveKeyWithValue("routing_key", routingKey1))
				Expect(imported.Config).To(HaveKeyWithValue("auto_start", true))
				alerts, ok := imported.Config["alerts"].([]any)
				Expect(ok).To(BeTrue())
				Expect(alerts).To(HaveLen(1))
				Expect(alerts[0]).To(HaveKeyWithValue("disabled", true))
				// The imported record mints its own key rather than reusing the
				// source task's.
				Expect(imported.Config["key"]).ToNot(Equal(t.Config["key"]))
			},
		)

		It(
			"Should upgrade a legacy Console export through the boot transforms",
			func(ctx SpecContext) {
				legacy := []byte(`{
					"type": "pagerduty_alert",
					"name": "Legacy PD Task",
					"routingKey": "rk-legacy",
					"autoStart": true,
					"alerts": [{"key": "a1", "status": "warning", "enabled": false}]
				}`)
				imported := reimport(ctx, legacy)
				Expect(imported.Config).ToNot(HaveKey("plain_routing_key"))
				Expect(imported.Config["routing_key"]).To(BeAssignableToTypeOf(""))
				routingKey := uuid.MustParse(imported.Config["routing_key"].(string))
				var s secret.Secret
				Expect(secretSvc.NewRetrieve().
					Where(secret.MatchKeys(routingKey)).
					Entry(&s).
					Exec(ctx, db)).To(Succeed())
				Expect(s.Base().Name).To(Equal("Legacy PD Task routing key"))
				Expect(imported.Config).To(HaveKeyWithValue("auto_start", true))
				alerts, ok := imported.Config["alerts"].([]any)
				Expect(ok).To(BeTrue())
				Expect(alerts[0]).To(HaveKeyWithValue("disabled", true))
				Expect(alerts[0]).ToNot(HaveKey("enabled"))
			},
		)

		It(
			"Should migrate a legacy config while another transaction is open",
			func(ctx SpecContext) {
				var env imex.Envelope
				Expect(json.Unmarshal([]byte(`{
					"type": "pagerduty_alert",
					"name": "Locked PD Task",
					"routingKey": "rk-locked"
				}`), &env)).To(Succeed())
				tx := db.OpenTx()
				DeferCleanup(tx.Close)
				writes := make(chan imex.Write, 1)
				go func() {
					defer GinkgoRecover()
					writes <- MustSucceed(imexSvc.Decode(ctx, env, ""))
				}()
				var write imex.Write
				Eventually(writes).Should(Receive(&write))
				id := MustSucceed(write(ctx, tx, imex.ImportOptions{}))
				var imported task.Task
				Expect(svc.NewRetrieve().
					Where(task.MatchKeys(uuid.MustParse(id.Key))).
					Entry(&imported).
					Exec(ctx, tx)).To(Succeed())
				routingKey := uuid.MustParse(imported.Config["routing_key"].(string))
				var s secret.Secret
				Expect(secretSvc.NewRetrieve().
					Where(secret.MatchKeys(routingKey)).
					Entry(&s).
					Exec(ctx, tx)).To(Succeed())
				Expect(s.Base().Name).To(Equal("Locked PD Task routing key"))
			},
		)

		It(
			"Should import a file stamped at the current version without rewriting",
			func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return statusSvc.NewWriter(tx).Set(ctx, &status.Status[any]{
						Key:     "stamped_status_channel",
						Name:    "Stamped",
						Variant: status.VariantInfo,
						Time:    telem.Now(),
					})
				})).To(Succeed())
				createRoutingKeys(ctx, db, secretSvc, stampedRoutingKey)
				raw := MustSucceed(os.ReadFile(
					filepath.Join("testdata", "import_typed_stamped.json"),
				))
				imported := reimport(ctx, raw)
				var body map[string]any
				Expect(json.Unmarshal(raw, &body)).To(Succeed())
				delete(body, "version")
				delete(body, "type")
				delete(body, "name")
				config := map[string]any(imported.Config)
				// The store mints a record key and fills defaults; every stamped
				// field must survive exactly as written.
				for k, v := range body {
					Expect(config[k]).To(BeComparableTo(v), "field %q changed", k)
				}
			},
		)

		It(
			"Should drop a routing key that names no secret on import",
			func(ctx SpecContext) {
				raw := bytes.ReplaceAll(
					MustSucceed(os.ReadFile(
						filepath.Join("testdata", "import_typed_stamped.json"),
					)),
					[]byte(stampedRoutingKey),
					[]byte(uuid.New().String()),
				)
				Expect(reimport(ctx, raw).Config).ToNot(HaveKey("routing_key"))
			},
		)

		It(
			"Should drop a status that does not exist from an imported config",
			func(ctx SpecContext) {
				imported := reimport(ctx, []byte(`{
					"version": 1,
					"type": "pagerduty_alert",
					"name": "Foreign Task",
					"routing_key": "rk-foreign",
					"alerts": [{"key": "a1", "status": "foreign", "component": "c"}]
				}`))
				Expect(imported.Config["alerts"]).To(HaveExactElements(SatisfyAll(
					HaveKeyWithValue("status", ""),
					HaveKeyWithValue("component", "c"),
				)))
			},
		)

		It(
			"Should reject a file version newer than the type's",
			func(ctx SpecContext) {
				raw := MustSucceed(os.ReadFile(
					filepath.Join("testdata", "import_bad_version.json"),
				))
				var env imex.Envelope
				Expect(json.Unmarshal(raw, &env)).To(Succeed())
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					_, err := imexSvc.Import(ctx, tx, env, imex.ImportOptions{})
					return err
				})).To(MatchError(ContainSubstring("newer than this Core supports")))
			},
		)

		It(
			"Should reject a file whose type has no registered importer",
			func(ctx SpecContext) {
				var env imex.Envelope
				Expect(json.Unmarshal(
					[]byte(`{"type": "not_a_task", "name": "mystery"}`),
					&env,
				)).To(Succeed())
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					_, err := imexSvc.Import(ctx, tx, env, imex.ImportOptions{})
					return err
				})).To(MatchError(ContainSubstring("no importer registered")))
			},
		)

		It(
			"Should reject a retired task type with a retirement message",
			func(ctx SpecContext) {
				var env imex.Envelope
				Expect(json.Unmarshal(
					[]byte(`{"type": "sequence", "name": "old sequence"}`),
					&env,
				)).To(Succeed())
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					_, err := imexSvc.Import(ctx, tx, env, imex.ImportOptions{})
					return err
				})).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("was removed")),
				))
			},
		)
	})

	Describe("Excluded types", Ordered, func() {
		var excluded stack
		BeforeAll(func(ctx SpecContext) {
			ShouldNotLeakGoroutines()
			excluded = openStack(ctx, "Excluded Rack", pagerduty.AlertTaskType)
		})

		It("Should register no importer for an excluded type", func(ctx SpecContext) {
			Expect(excluded.db.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := excluded.imex.Import(ctx, tx, imex.Envelope{
					Version: 1,
					Type:    pagerduty.AlertTaskType,
					Name:    "Excluded Task",
				}, imex.ImportOptions{})
				return err
			})).To(MatchError(ContainSubstring("no importer registered")))
		})

		It("Should refuse to export a task of an excluded type", func(ctx SpecContext) {
			t := &task.Task{
				Rack:   excluded.rack.Key,
				Name:   "Excluded Task",
				Type:   pagerduty.AlertTaskType,
				Config: msgpack.EncodedJSON{"routing_key": routingKey1},
			}
			Expect(excluded.db.WithTx(ctx, func(tx gorp.Tx) error {
				return excluded.task.NewWriter(tx).Create(ctx, t)
			})).To(Succeed())
			Expect(
				excluded.task.Export(ctx, excluded.db, t.ResourceID()),
			).Error().
				To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("have no file form")),
				))
		})
	})
})
