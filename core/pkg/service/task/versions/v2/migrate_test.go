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
	"strconv"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/labjack"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/task/common"
	v1 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migrations", func() {
	It(
		"Should migrate a legacy uint64-keyed task and its status to a UUID key",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
			otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
			searchIdx := MustOpen(search.OpenIndex())
			g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   searchIdx,
			}))
			labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    g,
				Search:   searchIdx,
			}))
			stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{
				Ontology: otg,
				DB:       db,
				Group:    g,
				Label:    labelSvc,
				Search:   searchIdx,
			}))
			rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
				DB:           db,
				Ontology:     otg,
				Group:        g,
				HostProvider: mock.NewStaticHostProvider(1),
				Status:       stat,
				Search:       searchIdx,
			}))

			testRack := &rack.Rack{Name: "Migration Test Rack"}
			Expect(rackSvc.NewWriter(nil).Create(ctx, testRack)).To(Succeed())

			legacyKey := v1.Key(uint64(testRack.Key)<<32 | 99)
			legacyTask := v1.Task{
				Key:  legacyKey,
				Name: "Legacy Task",
				Type: pagerduty.AlertTaskType,
				Config: msgpack.EncodedJSON{
					"routing_key": "rk-legacy",
					"data_saving": false,
				},
			}
			Expect(gorp.NewCreate[v1.Key, v1.Task]().
				Entry(&legacyTask).
				Exec(ctx, db)).To(Succeed())
			legacyID := ontology.ID{
				Type: ontology.ResourceTypeTask,
				Key:  strconv.FormatUint(uint64(legacyKey), 10),
			}
			Expect(otg.NewWriter(nil).DefineResources(ctx, legacyID)).To(Succeed())

			// Legacy data stored the task key as a msgpack float64; the v0 backfill
			// must read it through the flex decoder before the re-key runs.
			legacyStatus := status.Status[any]{
				Key:     "task:" + strconv.FormatUint(uint64(legacyKey), 10),
				Name:    "Legacy Task",
				Variant: status.VariantSuccess,
				Message: "Started",
				Time:    telem.Now(),
				Details: map[string]any{
					"task":    float64(legacyKey),
					"running": true,
					"cmd":     "start",
				},
			}
			Expect(
				status.NewWriter[any](stat, nil).Set(ctx, &legacyStatus),
			).To(Succeed())

			pd := MustOpen(pagerduty.OpenService(ctx, pagerduty.ServiceConfig{
				DB:       db,
				Ontology: otg,
			}))
			configs := MustSucceed(common.NewConfigRegistry(pd.Stores()...))
			svc := MustOpen(task.OpenService(ctx, task.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    g,
				Rack:     rackSvc,
				Status:   stat,
				Search:   searchIdx,
				ImEx:     imex.NewService(),
				Configs:  configs,
			}))

			var migrated task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchNames("Legacy Task")).
				Entry(&migrated).
				Exec(ctx, nil)).To(Succeed())
			Expect(migrated.Key).ToNot(Equal(uuid.Nil))
			Expect(migrated.Rack).To(Equal(testRack.Key))
			Expect(migrated.Config).To(HaveKeyWithValue("routing_key", "rk-legacy"))
			Expect(migrated.Config).ToNot(HaveKey("data_saving"))
			Expect(migrated.Config).To(HaveKey("key"))
			Expect(migrated.ConfigHash).ToNot(BeEmpty())

			var restoredStatus task.Status
			Expect(status.NewRetrieve[task.StatusDetails](stat).
				Where(status.MatchKeys[task.StatusDetails](task.OntologyID(migrated.Key).String())).
				Entry(&restoredStatus).
				Exec(ctx, nil)).To(Succeed())
			Expect(restoredStatus.Details.Task).To(Equal(migrated.Key))
			Expect(restoredStatus.Details.Running).To(BeTrue())
			Expect(restoredStatus.Details.Cmd).To(Equal("start"))

			Expect(otg.NewRetrieve().
				WhereIDs(task.OntologyID(migrated.Key)).
				Exists(ctx, nil)).To(BeTrue())
			Expect(otg.NewRetrieve().WhereIDs(legacyID).Exists(ctx, nil)).To(BeFalse())

			staged, closer := MustSucceed2(db.Get(ctx, v2.LegacyKeyKVKey(legacyKey)))
			Expect(string(staged)).To(Equal(migrated.Key.String()))
			Expect(closer.Close()).To(Succeed())
		},
	)

	It(
		"Should decompose stored configs, rename types, and quarantine unknowns",
		func(ctx SpecContext) {
			// Legacy rows from released builds are msgpack-encoded, while the opened
			// services run the production Orc codec. Seed through an msgpack view of
			// the same KV so the fixtures land in their real on-disk form.
			kvDB := memkv.New()
			seedDB := gorp.Wrap(kvDB, gorp.WithCodec(msgpack.Codec))
			db := DeferClose(gorp.Wrap(kvDB))
			otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
			searchIdx := MustOpen(search.OpenIndex())
			g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   searchIdx,
			}))
			labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    g,
				Search:   searchIdx,
			}))
			stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{
				Ontology: otg,
				DB:       db,
				Group:    g,
				Label:    labelSvc,
				Search:   searchIdx,
			}))
			rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
				DB:           db,
				Ontology:     otg,
				Group:        g,
				HostProvider: mock.NewStaticHostProvider(1),
				Status:       stat,
				Search:       searchIdx,
			}))
			testRack := &rack.Rack{Name: "Decompose Test Rack"}
			Expect(rackSvc.NewWriter(nil).Create(ctx, testRack)).To(Succeed())

			arcModuleKey := uuid.New().String()
			pdTask := v1.Task{
				Key:  v1.Key(uint64(testRack.Key)<<32 | 1),
				Name: "Stored PD Task",
				Type: pagerduty.AlertTaskType,
				Config: msgpack.EncodedJSON{
					"routing_key": "rk-stored",
					"data_saving": false,
				},
			}
			arcTask := v1.Task{
				Key:  v1.Key(uint64(testRack.Key)<<32 | 2),
				Name: "Stored Arc Task",
				Type: "arc",
				Config: msgpack.EncodedJSON{
					"arc_key": arcModuleKey,
					"hash":    "abc123",
				},
			}
			bogusTask := v1.Task{
				Key:    v1.Key(uint64(testRack.Key)<<32 | 3),
				Name:   "Bogus Task",
				Type:   "bogus",
				Config: msgpack.EncodedJSON{"anything": true},
			}
			ljScanTask := v1.Task{
				Key:    v1.Key(uint64(testRack.Key)<<32 | 5),
				Name:   "Stored LabJack Scan Task",
				Type:   "labjack_scan",
				Config: msgpack.EncodedJSON{},
			}
			retiredTasks := []v1.Task{
				{
					Key:    v1.Key(uint64(testRack.Key)<<32 | 6),
					Name:   "Stored Sequence Task",
					Type:   "sequence",
					Config: msgpack.EncodedJSON{"script": "return 1"},
				},
				{
					Key:    v1.Key(uint64(testRack.Key)<<32 | 7),
					Name:   "Stored Heartbeat Task",
					Type:   "heartbeat",
					Config: msgpack.EncodedJSON{},
				},
				{
					Key:    v1.Key(uint64(testRack.Key)<<32 | 8),
					Name:   "Stored OPC Scanner Task",
					Type:   "opcScanner",
					Config: msgpack.EncodedJSON{},
				},
			}
			ljTask := v1.Task{
				Key:  v1.Key(uint64(testRack.Key)<<32 | 4),
				Name: "Stored LabJack Write Task",
				Type: "labjack_write",
				Config: msgpack.EncodedJSON{
					"device":      "dev-lj",
					"state_rate":  float64(10),
					"data_saving": true,
					"auto_start":  false,
					"channels": []any{map[string]any{
						"type":      "DO",
						"key":       "ch-do",
						"port":      "DIO4",
						"cmd_key":   float64(7),
						"state_key": float64(8),
						"enabled":   true,
					}},
				},
			}
			Expect(gorp.NewCreate[v1.Key, v1.Task]().
				Entries(&[]v1.Task{pdTask, arcTask, bogusTask, ljTask, ljScanTask}).
				Exec(ctx, seedDB)).To(Succeed())
			Expect(gorp.NewCreate[v1.Key, v1.Task]().
				Entries(&retiredTasks).
				Exec(ctx, seedDB)).To(Succeed())

			pd := MustOpen(pagerduty.OpenService(ctx, pagerduty.ServiceConfig{
				DB:       db,
				Ontology: otg,
			}))
			at := MustOpen(arctask.OpenService(ctx, arctask.ServiceConfig{
				DB:       db,
				Ontology: otg,
			}))
			lj := MustOpen(labjack.OpenService(ctx, labjack.ServiceConfig{
				DB:       db,
				Ontology: otg,
			}))
			configs := MustSucceed(common.NewConfigRegistry(
				append(append(pd.Stores(), at.Stores()...), lj.Stores()...)...,
			))
			svc := MustOpen(task.OpenService(ctx, task.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    g,
				Rack:     rackSvc,
				Status:   stat,
				Search:   searchIdx,
				ImEx:     imex.NewService(),
				Configs:  configs,
			}))

			var migratedPD task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchNames("Stored PD Task")).
				Entry(&migratedPD).
				Exec(ctx, nil)).To(Succeed())
			Expect(migratedPD.Config).To(
				HaveKeyWithValue("routing_key", "rk-stored"),
			)
			Expect(migratedPD.Config).ToNot(HaveKey("data_saving"))
			Expect(migratedPD.Config).To(HaveKey("key"))
			Expect(migratedPD.ConfigHash).ToNot(BeEmpty())
			parents := MustSucceed(otg.RetrieveParents(
				nil, task.OntologyID(migratedPD.Key),
			))
			ids := parents[task.OntologyID(migratedPD.Key)]
			Expect(ids).To(HaveLen(1))
			Expect(ids[0].Type).To(
				Equal(ontology.ResourceType(pagerduty.AlertTaskType)),
			)

			var migratedArc task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchNames("Stored Arc Task")).
				Entry(&migratedArc).
				Exec(ctx, nil)).To(Succeed())
			Expect(migratedArc.Type).To(Equal("arc_task"))
			Expect(migratedArc.Config).To(
				HaveKeyWithValue("arc_key", arcModuleKey),
			)
			Expect(migratedArc.Config).To(HaveKeyWithValue("hash", "abc123"))
			Expect(migratedArc.Config).To(
				HaveKeyWithValue("execution_mode", "AUTO"),
			)
			Expect(migratedArc.Config).To(
				HaveKeyWithValue("rt_priority", BeEquivalentTo(47)),
			)
			Expect(migratedArc.Config).To(
				HaveKeyWithValue("cpu_affinity", BeEquivalentTo(-1)),
			)

			var migratedScan task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchNames("Stored LabJack Scan Task")).
				Entry(&migratedScan).
				Exec(ctx, nil)).To(Succeed())
			Expect(migratedScan.Config).To(
				HaveKeyWithValue("rate", BeEquivalentTo(0.2)),
			)

			var migratedLJ task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchNames("Stored LabJack Write Task")).
				Entry(&migratedLJ).
				Exec(ctx, nil)).To(Succeed())
			Expect(migratedLJ.Config).To(HaveKeyWithValue("device", "dev-lj"))
			Expect(migratedLJ.Config).To(
				HaveKeyWithValue("data_saving_disabled", false),
			)
			ljChannels, ok := migratedLJ.Config["channels"].([]any)
			Expect(ok).To(BeTrue())
			Expect(ljChannels).To(HaveLen(1))
			ljCh, ok := ljChannels[0].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(ljCh).To(HaveKeyWithValue("cmd_channel", BeEquivalentTo(7)))
			Expect(ljCh).To(HaveKeyWithValue("state_channel", BeEquivalentTo(8)))
			Expect(ljCh).To(HaveKeyWithValue("disabled", false))
			Expect(ljCh).ToNot(HaveKey("cmd_key"))
			Expect(ljCh).ToNot(HaveKey("state_key"))

			Expect(svc.NewRetrieve().
				Where(task.MatchNames("Bogus Task")).
				Exists(ctx, nil)).To(BeFalse())
			rekeyed, rekeyCloser := MustSucceed2(
				db.Get(ctx, v2.LegacyKeyKVKey(bogusTask.Key)),
			)
			bogusKey := MustSucceed(uuid.Parse(string(rekeyed)))
			Expect(rekeyCloser.Close()).To(Succeed())
			staged, closer := MustSucceed2(
				db.Get(ctx, v3.QuarantineKVKey(bogusKey)),
			)
			Expect(string(staged)).To(ContainSubstring("Bogus Task"))
			Expect(closer.Close()).To(Succeed())

			for _, retired := range retiredTasks {
				Expect(svc.NewRetrieve().
					Where(task.MatchNames(retired.Name)).
					Exists(ctx, nil)).To(BeFalse())
				rekeyed, rekeyCloser := MustSucceed2(
					db.Get(ctx, v2.LegacyKeyKVKey(retired.Key)),
				)
				retiredKey := MustSucceed(uuid.Parse(string(rekeyed)))
				Expect(rekeyCloser.Close()).To(Succeed())
				blob, blobCloser := MustSucceed2(
					db.Get(ctx, v3.QuarantineKVKey(retiredKey)),
				)
				Expect(string(blob)).To(ContainSubstring(retired.Name))
				Expect(blobCloser.Close()).To(Succeed())
			}
		},
	)
})
