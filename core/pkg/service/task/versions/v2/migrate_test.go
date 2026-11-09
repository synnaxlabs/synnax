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
	"slices"
	"strconv"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/labjack"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/ni"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	secrettestutil "github.com/synnaxlabs/synnax/pkg/service/secret/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	statusv2 "github.com/synnaxlabs/synnax/pkg/service/status/versions/v2"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	v1 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func resourceExists(ctx context.Context, db *gorp.DB, id resource.ID) (bool, error) {
	return gorp.NewRetrieve[string, ontology.Resource]().
		Where(gorp.MatchKeys[string, ontology.Resource](id.String())).
		Exists(ctx, db)
}

var _ = Describe("Migrations", func() {
	It(
		"Should migrate a legacy uint64-keyed task and its status to a UUID key",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
			MustOpen(label.OpenService(ctx, label.ServiceConfig{
				DB: db,
			}))
			stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{
				DB: db,
			}))
			roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
				DB: db,
			}))
			rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
				DB:      db,
				HostKey: 1,
				Status:  stat,
				DriverRole: MustSucceed(
					racktestutil.CreateDriverRole(ctx, db, roleSvc),
				),
			}))

			testRack := &rack.Rack{Name: "Migration Test Rack"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return rackSvc.NewWriter(tx).Create(ctx, testRack)
			})).To(Succeed())

			legacyKey := v1.Key(uint64(testRack.Key)<<32 | 99)
			legacyTask := v1.Task{
				Key:    legacyKey,
				Name:   "Legacy Task",
				Type:   pagerduty.AlertTaskType,
				Config: msgpack.EncodedJSON{"data_saving": false},
			}
			Expect(gorp.NewCreate[v1.Key, v1.Task]().
				Entry(&legacyTask).
				Exec(ctx, db)).To(Succeed())
			legacyID := resource.ID{
				Type: resource.TypeTask,
				Key:  strconv.FormatUint(uint64(legacyKey), 10),
			}
			Expect(ontology.DefineResources(ctx, db, legacyID)).To(Succeed())

			// Legacy data stored the task key as a msgpack float64; the v0 backfill
			// must read it through the flex decoder before the re-key runs. The status
			// migration already gave the status its owner and key.
			legacyStatusKey, _ := statusv2.MigrateKey(legacyID.String())
			legacyStatus := status.Status[any]{
				Key:     legacyStatusKey,
				Owner:   new(legacyID),
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
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return stat.NewMigrationWriter(tx).Set(ctx, legacyStatus)
			})).To(Succeed())

			pd := MustOpen(pagerduty.OpenService(ctx, pagerduty.ServiceConfig{
				DB:     db,
				Secret: secrettestutil.OpenService(ctx, db),
			}))
			configs := MustSucceed(config.NewRegistry(pd.Stores()...))
			svc := MustOpen(task.OpenService(ctx, task.ServiceConfig{
				DB:      db,
				Rack:    rackSvc,
				Status:  stat,
				ImEx:    imex.NewService(),
				Configs: configs,
			}))

			var migrated task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchNames("Legacy Task")).
				Entry(&migrated).
				Exec(ctx, db)).To(Succeed())
			Expect(migrated.Key).ToNot(Equal(uuid.Nil()))
			Expect(migrated.Rack).To(Equal(testRack.Key))
			Expect(migrated.Config).ToNot(HaveKey("data_saving"))
			Expect(migrated.Config).To(HaveKey("key"))
			Expect(migrated.ConfigHash).ToNot(BeEmpty())

			var restoredStatus task.Status
			Expect(stat.NewRetrieve[task.StatusDetails]().
				Where(status.MatchOwners[task.StatusDetails](task.ResourceID(migrated.Key))).
				Entry(&restoredStatus).
				Exec(ctx, db)).To(Succeed())
			Expect(restoredStatus.Key).To(Equal(legacyStatusKey))
			Expect(restoredStatus.Details.Running).To(BeTrue())
			Expect(restoredStatus.Details.Cmd).To(Equal("start"))

			Expect(resourceExists(ctx, db, task.ResourceID(migrated.Key))).To(BeTrue())
			Expect(resourceExists(ctx, db, legacyID)).To(BeFalse())

			staged, closer := MustSucceed2(db.Get(ctx, v2.LegacyKeyKVKey(legacyKey)))
			Expect(string(staged)).To(Equal(migrated.Key.String()))
			Expect(closer.Close()).To(Succeed())
		},
	)

	It(
		"Should decompose stored configs, rename types, and quarantine unknowns",
		func(ctx SpecContext) {
			// Legacy rows from released builds are msgpack-encoded, while the opened
			// services run the production Orc codec. Stage through an msgpack view of
			// the same KV so the fixtures land in their real on-disk form.
			node := testutil.NewNode(ctx)
			db := node.DB
			stageDB := gorp.Wrap(db.DB, gorp.WithCodec(msgpack.Codec))
			MustOpen(label.OpenService(ctx, label.ServiceConfig{
				DB: db,
			}))
			stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{
				DB: db,
			}))
			roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
				DB: db,
			}))
			rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
				DB:      db,
				HostKey: 1,
				Status:  stat,
				DriverRole: MustSucceed(
					racktestutil.CreateDriverRole(ctx, db, roleSvc),
				),
			}))
			testRack := &rack.Rack{Name: "Decompose Test Rack"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return rackSvc.NewWriter(tx).Create(ctx, testRack)
			})).To(Succeed())
			targets := tasktestutil.NewTargets(tasktestutil.TargetsConfig{
				DB: db,
				Device: MustOpen(device.OpenService(ctx, device.ServiceConfig{
					DB:     db,
					Status: stat,
					Rack:   rackSvc,
				})),
				Status: stat,
				Rack:   testRack.Key,
			})
			Expect(targets.CreateDevice(ctx, "dev-ni")).To(Succeed())
			Expect(targets.CreateDevice(ctx, "dev-lj")).To(Succeed())
			channelSvc := MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
				Channel: node.Channel,
				DB:      db,
				HostKey: node.Identity.HostKey,
				Status:  stat,
			}))
			keys := MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, 4))
			cmdKey, stateKey, tcKey, freqKey := keys[0], keys[1], keys[2], keys[3]

			MustOpen(arc.OpenTable(ctx, arc.TableConfig{DB: db}))
			arcModule := arc.Arc{
				Variant: arc.AutomationArc{
					Base: arc.Base{
						Key:    uuid.New(),
						Name:   "Stored Arc",
						Source: arc.Source{Variant: arc.TextSource{}},
					},
				},
			}
			Expect(gorp.NewCreate[arc.Key, arc.Arc]().Entry(&arcModule).Exec(ctx, db)).
				To(Succeed())
			arcModuleKey := arcModule.Base().Key.String()
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
					"arc_key":        arcModuleKey,
					"hash":           "abc123",
					"execution_mode": "HYBRID",
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
			// A v0-era console row: snake_case from the wire, the device at the
			// config level, the renamed AI type alias, and flat cold-junction
			// fields.
			niTask := v1.Task{
				Key:  v1.Key(uint64(testRack.Key)<<32 | 9),
				Name: "Stored NI Analog Read Task",
				Type: "ni_analog_read",
				Config: msgpack.EncodedJSON{
					"device":      "dev-ni",
					"sample_rate": float64(100),
					"stream_rate": float64(10),
					"data_saving": true,
					"channels": []any{
						map[string]any{
							"key":        "ch-ni-tc",
							"channel":    float64(tcKey),
							"port":       float64(1),
							"type":       "ai_thermocouple",
							"units":      "DegC",
							"enabled":    true,
							"min_val":    float64(-100),
							"max_val":    float64(900),
							"cjc_source": "ConstVal",
							"cjc_val":    float64(25.5),
						},
						map[string]any{
							"key":          "ch-ni-freq",
							"channel":      float64(freqKey),
							"port":         float64(2),
							"type":         "ai_frequency_voltage",
							"units":        "Hz",
							"enabled":      false,
							"min_val":      float64(1),
							"max_val":      float64(5000),
							"custom_scale": map[string]any{"type": "none"},
						},
					},
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
						"cmd_key":   float64(cmdKey),
						"state_key": float64(stateKey),
						"enabled":   true,
					}},
				},
			}
			Expect(gorp.NewCreate[v1.Key, v1.Task]().
				Entries(&[]v1.Task{
					pdTask, arcTask, bogusTask, ljTask, ljScanTask, niTask,
				}).
				Exec(ctx, stageDB)).To(Succeed())
			Expect(gorp.NewCreate[v1.Key, v1.Task]().
				Entries(&retiredTasks).
				Exec(ctx, stageDB)).To(Succeed())

			secrets := secrettestutil.OpenService(ctx, db)
			pd := MustOpen(pagerduty.OpenService(ctx, pagerduty.ServiceConfig{
				DB:     db,
				Secret: secrets,
			}))
			at := MustOpen(arctask.OpenService(ctx, arctask.ServiceConfig{DB: db}))
			lj := MustOpen(labjack.OpenService(ctx, labjack.ServiceConfig{DB: db}))
			niSvc := MustOpen(ni.OpenService(ctx, ni.ServiceConfig{DB: db}))
			configs := MustSucceed(config.NewRegistry(slices.Concat(
				pd.Stores(), at.Stores(), lj.Stores(), niSvc.Stores(),
			)...))
			svc := MustOpen(task.OpenService(ctx, task.ServiceConfig{
				DB:      db,
				Rack:    rackSvc,
				Status:  stat,
				ImEx:    imex.NewService(),
				Configs: configs,
			}))

			var migratedPD task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchNames("Stored PD Task")).
				Entry(&migratedPD).
				Exec(ctx, db)).To(Succeed())
			Expect(migratedPD.Config["routing_key"]).To(BeAssignableToTypeOf(""))
			routingKey := uuid.MustParse(migratedPD.Config["routing_key"].(string))
			Expect(secrets.Reveal(ctx, db, routingKey)).
				To(HaveKeyWithValue(routingKey, []byte("rk-stored")))
			Expect(migratedPD.Config).ToNot(HaveKey("data_saving"))
			Expect(migratedPD.Config).To(HaveKey("key"))
			Expect(migratedPD.ConfigHash).ToNot(BeEmpty())

			var migratedArc task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchNames("Stored Arc Task")).
				Entry(&migratedArc).
				Exec(ctx, db)).To(Succeed())
			Expect(migratedArc.Type).To(Equal("arc"))
			Expect(migratedArc.Config).To(
				HaveKeyWithValue("arc_key", arcModuleKey),
			)
			Expect(migratedArc.Config).To(HaveKeyWithValue("hash", "abc123"))
			Expect(migratedArc.Config).To(
				HaveKeyWithValue("performance", "medium"),
			)
			Expect(migratedArc.Config).ToNot(HaveKey("execution_mode"))

			var migratedScan task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchNames("Stored LabJack Scan Task")).
				Entry(&migratedScan).
				Exec(ctx, db)).To(Succeed())
			Expect(migratedScan.Config).To(
				HaveKeyWithValue("rate", BeEquivalentTo(0.2)),
			)

			var migratedLJ task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchNames("Stored LabJack Write Task")).
				Entry(&migratedLJ).
				Exec(ctx, db)).To(Succeed())
			Expect(migratedLJ.Config).To(HaveKeyWithValue("device", "dev-lj"))
			Expect(migratedLJ.Config).To(
				HaveKeyWithValue("data_saving_disabled", false),
			)
			ljChannels, ok := migratedLJ.Config["channels"].([]any)
			Expect(ok).To(BeTrue())
			Expect(ljChannels).To(HaveLen(1))
			ljCh, ok := ljChannels[0].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(ljCh).To(HaveKeyWithValue("cmd_channel", BeEquivalentTo(cmdKey)))
			Expect(ljCh).To(HaveKeyWithValue("state_channel", BeEquivalentTo(stateKey)))
			Expect(ljCh).To(HaveKeyWithValue("disabled", false))
			Expect(ljCh).ToNot(HaveKey("cmd_key"))
			Expect(ljCh).ToNot(HaveKey("state_key"))

			var migratedNI task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchNames("Stored NI Analog Read Task")).
				Entry(&migratedNI).
				Exec(ctx, db)).To(Succeed())
			Expect(migratedNI.Config).ToNot(HaveKey("device"))
			Expect(migratedNI.Config).To(
				HaveKeyWithValue("data_saving_disabled", false),
			)
			niChannels, ok := migratedNI.Config["channels"].([]any)
			Expect(ok).To(BeTrue())
			Expect(niChannels).To(HaveLen(2))
			tc, ok := niChannels[0].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(tc).To(HaveKeyWithValue("device", "dev-ni"))
			Expect(tc).To(HaveKeyWithValue("cjc", SatisfyAll(
				HaveKeyWithValue("source", "const_val"),
				HaveKeyWithValue("val", BeEquivalentTo(25.5)),
			)))
			Expect(tc).ToNot(HaveKey("cjc_source"))
			Expect(tc).ToNot(HaveKey("cjc_val"))
			freq, ok := niChannels[1].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(freq).To(HaveKeyWithValue("device", "dev-ni"))
			Expect(freq).To(HaveKeyWithValue("type", "ai_freq_voltage"))
			Expect(freq).To(HaveKeyWithValue("disabled", true))

			Expect(svc.NewRetrieve().
				Where(task.MatchNames("Bogus Task")).
				Exists(ctx, db)).To(BeFalse())
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
					Exists(ctx, db)).To(BeFalse())
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

var _ = Describe("NameMigration", func() {
	migrated := func(ctx SpecContext, in v2.Task) v2.Task {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		Expect(gorp.NewCreate[v2.Key, v2.Task]().Entry(&in).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Task",
			Migrations: []gorp.Migration{v2.NameMigration},
		})).To(Succeed())
		var out v2.Task
		Expect(gorp.NewRetrieve[v2.Key, v2.Task]().
			Where(gorp.MatchKeys[v2.Key, v2.Task](in.Key)).
			Entry(&out).Exec(ctx, db)).To(Succeed())
		return out
	}
	key := uuid.MustParse("1a2b3c4d-0000-0000-0000-000000000000")
	DescribeTable(
		"Should name only nameless tasks",
		func(ctx SpecContext, name, taskType, expected string) {
			in := v2.Task{Key: key, Name: name, Type: taskType}
			Expect(migrated(ctx, in).Name).To(Equal(expected))
		},
		Entry("type and key", "", "ni_analog_read", "ni_analog_read 1a2b3c4d"),
		Entry("no type", "", "", "Task 1a2b3c4d"),
		Entry("named task", "Named", "ni_analog_read", "Named"),
	)
})
