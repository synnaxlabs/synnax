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
	"os"
	"path/filepath"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	pdlegacy "github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/legacy"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// stack is a task service over an in-memory database, holding the PagerDuty config
// store and its own ImEx registry, plus a rack to create tasks on.
type stack struct {
	task *task.Service
	imex *imex.Service
	rack *rack.Rack
}

// openStack opens a stack whose task service excludes excluded from import and export.
func openStack(ctx context.Context, rackName string, excluded ...string) stack {
	GinkgoHelper()
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
	rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:                  db,
		Ontology:            otg,
		Group:               groupSvc,
		HostProvider:        mock.NewStaticHostProvider(1),
		Status:              statusSvc,
		HealthCheckInterval: 10 * telem.Millisecond,
		Search:              searchIdx,
	}))
	pd := MustOpen(pagerduty.OpenService(ctx, pagerduty.ServiceConfig{DB: db}))
	imexSvc := imex.NewService()
	taskSvc := MustOpen(task.OpenService(ctx, task.ServiceConfig{
		DB:           db,
		Ontology:     otg,
		Group:        groupSvc,
		Rack:         rackSvc,
		Status:       statusSvc,
		Search:       searchIdx,
		ImEx:         imexSvc,
		Configs:      MustSucceed(config.NewRegistry(pd.Stores()...)),
		ImExExcluded: excluded,
	}))
	r := &rack.Rack{Name: rackName}
	Expect(rackSvc.NewWriter(nil).Create(ctx, r)).To(Succeed())
	return stack{task: taskSvc, imex: imexSvc, rack: r}
}

var _ = Describe("ImEx", Ordered, func() {
	var (
		svc      *task.Service
		testRack *rack.Rack
		imexSvc  *imex.Service
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		s := openStack(ctx, "Test Rack")
		svc, imexSvc, testRack = s.task, s.imex, s.rack
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
				// The version is the config type's own number line: one above
				// PagerDuty's last legacy shape.
				Expect(env.Version).To(Equal(pdlegacy.LastVersion + 1))
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
			"Should import a file stamped at the current version without rewriting",
			func(ctx SpecContext) {
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
			"Should reject a file version newer than the type's",
			func(ctx SpecContext) {
				raw := MustSucceed(os.ReadFile(
					filepath.Join("testdata", "import_bad_version.json"),
				))
				var env imex.Envelope
				Expect(json.Unmarshal(raw, &env)).To(Succeed())
				Expect(imexSvc.Import(
					ctx, nil, env, imex.ImportOptions{Parent: ontology.RootID},
				)).Error().To(MatchError(ContainSubstring("newer than this Core supports")))
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

	Describe("Excluded types", Ordered, func() {
		var excluded stack
		BeforeAll(func(ctx SpecContext) {
			ShouldNotLeakGoroutines()
			excluded = openStack(ctx, "Excluded Rack", pagerduty.AlertTaskType)
		})

		It("Should register no importer for an excluded type", func() {
			Expect(excluded.imex.ImporterType(pagerduty.AlertTaskType)).
				Error().
				To(MatchError(ContainSubstring("no importer registered")))
		})

		It("Should refuse to export a task of an excluded type", func(ctx SpecContext) {
			t := &task.Task{
				Rack:   excluded.rack.Key,
				Name:   "Excluded Task",
				Type:   pagerduty.AlertTaskType,
				Config: msgpack.EncodedJSON{"routing_key": "rk-1"},
			}
			Expect(excluded.task.NewWriter(nil).Create(ctx, t)).To(Succeed())
			Expect(excluded.task.Export(ctx, t.OntologyID())).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("have no file form")),
			))
		})
	})
})
