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
	"context"
	"encoding/json"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/task/common"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ImEx", Ordered, func() {
	var (
		svc      *task.Service
		rackSvc  *rack.Service
		testRack *rack.Rack
		imexSvc  *imex.Service
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db := DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx := MustOpen(search.OpenIndex())
		groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Search:   searchIdx,
		}))
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Label:    labelSvc,
			Search:   searchIdx,
		}))
		rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:                  db,
			Ontology:            otg,
			Group:               groupSvc,
			HostProvider:        mock.NewStaticHostProvider(1),
			Status:              statusSvc,
			HealthCheckInterval: 10 * telem.Millisecond,
			Search:              searchIdx,
		}))
		pd := MustOpen(pagerduty.OpenService(ctx, pagerduty.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
		configs := MustSucceed(common.NewConfigRegistry(pd.Stores()...))
		imexSvc = imex.NewService()
		svc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Rack:     rackSvc,
			Status:   statusSvc,
			Search:   searchIdx,
			ImEx:     imexSvc,
			Configs:  configs,
		}))
		testRack = &rack.Rack{Name: "Test Rack"}
		Expect(rackSvc.NewWriter(nil).Create(ctx, testRack)).To(Succeed())
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
						"routing_key": "rk-1",
						"auto_start":  true,
					},
				}
				Expect(svc.NewWriter(nil).Create(ctx, t)).To(Succeed())

				env := MustSucceed(svc.Export(ctx, t.OntologyID()))
				Expect(env.Version).To(Equal(task.Version))
				Expect(env.Type).To(Equal(pagerduty.AlertTaskType))
				Expect(env.Name).To(Equal("Exported Task"))

				var body map[string]any
				Expect(
					json.Unmarshal(MustSucceed(json.Marshal(env)), &body),
				).To(Succeed())
				// The driver reads the file as its config, so config fields sit flat
				// at the top level rather than nested under a "config" key.
				Expect(body).ToNot(HaveKey("config"))
				Expect(body["routing_key"]).To(Equal("rk-1"))
				Expect(body["auto_start"]).To(BeTrue())
				Expect(body["type"]).To(Equal(pagerduty.AlertTaskType))
				Expect(body["name"]).To(Equal("Exported Task"))
				Expect(body["version"]).To(BeEquivalentTo(1))
			},
		)

		It("Should return not found for a missing key", func(ctx SpecContext) {
			id := task.OntologyID(uuid.New())
			Expect(svc.Export(ctx, id)).Error().To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Import", func() {
		reimport := func(ctx context.Context, raw []byte) task.Task {
			GinkgoHelper()
			var env imex.Envelope
			Expect(json.Unmarshal(raw, &env)).To(Succeed())
			id := MustSucceed(imexSvc.Import(
				ctx, nil, env, imex.ImportOptions{Parent: ontology.RootID},
			))
			var imported task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchKeys(uuid.MustParse(id.Key))).
				Entry(&imported).
				Exec(ctx, nil)).To(Succeed())
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
						"routing_key": "rk-rt",
						"auto_start":  true,
						"alerts": []any{map[string]any{
							"key":      "a1",
							"status":   "critical",
							"disabled": true,
						}},
					},
				}
				Expect(svc.NewWriter(nil).Create(ctx, t)).To(Succeed())
				env := MustSucceed(svc.Export(ctx, t.OntologyID()))
				imported := reimport(ctx, MustSucceed(json.Marshal(env)))

				Expect(imported.Key).ToNot(Equal(t.Key))
				Expect(imported.Rack).To(Equal(rack.Key(0)))
				Expect(imported.Name).To(Equal("Round Trip Task"))
				Expect(imported.Type).To(Equal(pagerduty.AlertTaskType))
				Expect(imported.Config).To(HaveKeyWithValue("routing_key", "rk-rt"))
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
				Expect(imported.Config).To(HaveKeyWithValue("routing_key", "rk-legacy"))
				Expect(imported.Config).To(HaveKeyWithValue("auto_start", true))
				alerts, ok := imported.Config["alerts"].([]any)
				Expect(ok).To(BeTrue())
				Expect(alerts[0]).To(HaveKeyWithValue("disabled", true))
				Expect(alerts[0]).ToNot(HaveKey("enabled"))
			},
		)

		It(
			"Should reject a file version newer than the exporter's",
			func(ctx SpecContext) {
				var env imex.Envelope
				Expect(json.Unmarshal(
					[]byte(
						`{"type": "pagerduty_alert", "name": "future", "version": 2}`,
					),
					&env,
				)).To(Succeed())
				Expect(imexSvc.Import(
					ctx, nil, env, imex.ImportOptions{Parent: ontology.RootID},
				)).Error().To(MatchError(ContainSubstring("unsupported task file version")))
			},
		)

		It(
			"Should reject a file whose type has no registered importer",
			func(ctx SpecContext) {
				var env imex.Envelope
				Expect(json.Unmarshal(
					[]byte(`{"type": "sequence", "name": "old sequence"}`),
					&env,
				)).To(Succeed())
				Expect(imexSvc.Import(
					ctx, nil, env, imex.ImportOptions{Parent: ontology.RootID},
				)).Error().To(MatchError(ContainSubstring("no importer registered")))
			},
		)
	})
})
