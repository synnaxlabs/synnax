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
	"math"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
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

const testType = pagerduty.AlertTaskType

var _ = Describe("Task", Ordered, func() {
	var (
		db          *gorp.DB
		svc         *task.Service
		otg         *ontology.Ontology
		w           task.Writer
		tx          gorp.Tx
		rackService *rack.Service
		testRack    *rack.Rack
		stat        *status.Service
		configs     config.Registry
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
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
		stat = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    g,
			Label:    labelSvc,
			Search:   searchIdx,
		}))
		rackService = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:                  db,
			Ontology:            otg,
			Group:               g,
			HostProvider:        mock.NewStaticHostProvider(1),
			Status:              stat,
			HealthCheckInterval: 10 * telem.Millisecond,
			Search:              searchIdx,
		}))
		pd := MustOpen(pagerduty.OpenService(ctx, pagerduty.ServiceConfig{DB: db}))
		at := MustOpen(arctask.OpenService(ctx, arctask.ServiceConfig{DB: db}))
		configs = MustSucceed(config.NewRegistry(
			append(pd.Stores(), at.Stores()...)...,
		))
		svc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Rack:     rackService,
			Status:   stat,
			Search:   searchIdx,
			ImEx:     imex.NewService(),
			Configs:  configs,
		}))
		testRack = &rack.Rack{Name: "Test Rack"}
		Expect(rackService.NewWriter(db).Create(ctx, testRack)).To(Succeed())
	})
	BeforeEach(func(ctx SpecContext) {
		tx = db.OpenTx()
		w = svc.NewWriter(tx)
	})
	AfterEach(func(ctx SpecContext) {
		Expect(tx.Close()).To(Succeed())
	})
	Describe("CommandChannelKey", func() {
		It(
			"Should return zero when no channel service is configured",
			func(ctx SpecContext) {
				Expect(svc.CommandChannelKey()).To(Equal(channel.Key(0)))
			},
		)
	})
	Describe("StatusDetails msgpack round-trip", func() {
		It("Should round-trip a UUID task key", func(ctx SpecContext) {
			original := task.StatusDetails{
				Task:    uuid.New(),
				Running: true,
				Cmd:     "start",
			}
			data := MustSucceed(msgpack.Codec.Encode(ctx, original))
			var decoded task.StatusDetails
			Expect(msgpack.Codec.Decode(ctx, data, &decoded)).To(Succeed())
			Expect(decoded).To(Equal(original))
		})
	})

	Describe("Create", func() {
		It("Should mint a unique key when none is provided", func(ctx SpecContext) {
			m := &task.Task{
				Type: testType,
				Rack: testRack.Key,
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).ToNot(Equal(uuid.Nil))
			Expect(m.Name).To(Equal("Test Task"))
		})
		It("Should preserve a client-provided key", func(ctx SpecContext) {
			key := uuid.New()
			m := &task.Task{
				Type: testType,
				Key:  key,
				Rack: testRack.Key,
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(key))
		})
		It("Should create a rackless draft", func(ctx SpecContext) {
			m := &task.Task{Type: testType, Name: "Draft Task"}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).ToNot(Equal(uuid.Nil))
			Expect(m.Rack.IsZero()).To(BeTrue())
		})
	})

	Describe("ConfigHash", func() {
		create := func(ctx context.Context, config msgpack.EncodedJSON) string {
			t := &task.Task{
				Type:   testType,
				Rack:   testRack.Key,
				Name:   "Test Task",
				Config: config,
			}
			Expect(w.Create(ctx, t)).To(Succeed())
			return t.ConfigHash
		}
		It("Should assign a hash on create", func(ctx SpecContext) {
			Expect(
				create(ctx, msgpack.EncodedJSON{"routing_key": "rk-1"}),
			).ToNot(BeEmpty())
		})
		It(
			"Should hash equal configs equally regardless of key order",
			func(ctx SpecContext) {
				first := create(ctx, msgpack.EncodedJSON{
					"routing_key": "rk-1", "auto_start": true,
				})
				second := create(ctx, msgpack.EncodedJSON{
					"auto_start": true, "routing_key": "rk-1",
				})
				Expect(first).To(Equal(second))
			},
		)
		It("Should hash differing configs differently", func(ctx SpecContext) {
			first := create(ctx, msgpack.EncodedJSON{"routing_key": "rk-1"})
			second := create(ctx, msgpack.EncodedJSON{"routing_key": "rk-2"})
			Expect(first).ToNot(Equal(second))
		})
		It("Should hash a nil and an empty config identically", func(ctx SpecContext) {
			Expect(create(ctx, nil)).To(Equal(create(ctx, msgpack.EncodedJSON{})))
		})
		// msgpack clients send integers where JSON clients send floats, so the two
		// encodings of one config must not read as drift.
		It(
			"Should hash integer and integral float values identically",
			func(ctx SpecContext) {
				createArc := func(config msgpack.EncodedJSON) string {
					t := &task.Task{
						Type:   arc.TaskType,
						Rack:   testRack.Key,
						Name:   "Arc Task",
						Config: config,
					}
					Expect(w.Create(ctx, t)).To(Succeed())
					return t.ConfigHash
				}
				Expect(createArc(msgpack.EncodedJSON{"rt_priority": 50})).
					To(Equal(createArc(msgpack.EncodedJSON{"rt_priority": 50.0})))
			},
		)
		It(
			"Should restore the original hash when an edit is undone",
			func(ctx SpecContext) {
				t := &task.Task{
					Type:   testType,
					Rack:   testRack.Key,
					Name:   "Test Task",
					Config: msgpack.EncodedJSON{"routing_key": "rk-1"},
				}
				Expect(w.Create(ctx, t)).To(Succeed())
				original := t.ConfigHash
				t.Config = msgpack.EncodedJSON{"routing_key": "rk-2"}
				Expect(w.Create(ctx, t)).To(Succeed())
				Expect(t.ConfigHash).ToNot(Equal(original))
				t.Config = msgpack.EncodedJSON{"routing_key": "rk-1"}
				Expect(w.Create(ctx, t)).To(Succeed())
				Expect(t.ConfigHash).To(Equal(original))
			},
		)
		It("Should ignore a client-provided hash", func(ctx SpecContext) {
			t := &task.Task{
				Type:       testType,
				Rack:       testRack.Key,
				Name:       "Test Task",
				Config:     msgpack.EncodedJSON{"routing_key": "rk-1"},
				ConfigHash: "deadbeefdeadbeef",
			}
			Expect(w.Create(ctx, t)).To(Succeed())
			Expect(t.ConfigHash).ToNot(Equal("deadbeefdeadbeef"))
			Expect(t.ConfigHash).To(Equal(
				create(ctx, msgpack.EncodedJSON{"routing_key": "rk-1"}),
			))
		})
		// A NaN cannot encode to JSON, so the config store must reject it before
		// anything is persisted.
		It("Should reject a config that cannot be encoded", func(ctx SpecContext) {
			t := &task.Task{
				Type:   testType,
				Rack:   testRack.Key,
				Name:   "Test Task",
				Config: msgpack.EncodedJSON{"routing_key": math.NaN()},
			}
			Expect(w.Create(ctx, t)).To(MatchError(validate.ErrValidation))
		})
		It("Should persist the hash alongside the task", func(ctx SpecContext) {
			t := &task.Task{
				Type:   testType,
				Rack:   testRack.Key,
				Name:   "Test Task",
				Config: msgpack.EncodedJSON{"routing_key": "rk-1"},
			}
			Expect(w.Create(ctx, t)).To(Succeed())
			var retrieved task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchKeys(t.Key)).
				Entry(&retrieved).
				Exec(ctx, tx)).To(Succeed())
			Expect(retrieved.ConfigHash).To(Equal(t.ConfigHash))
		})
		It(
			"Should keep the existing hash when a snapshot's config is retained",
			func(ctx SpecContext) {
				t := &task.Task{
					Type:   testType,
					Rack:   testRack.Key,
					Name:   "Test Task",
					Config: msgpack.EncodedJSON{"routing_key": "rk-1"},
				}
				Expect(w.Create(ctx, t)).To(Succeed())
				snap := MustSucceed(w.Copy(ctx, t.Key, "Snapshot", true))
				Expect(snap.ConfigHash).To(Equal(t.ConfigHash))
				edited := snap
				edited.Config = msgpack.EncodedJSON{"routing_key": "rk-999"}
				Expect(w.Create(ctx, &edited)).To(Succeed())
				var retrieved task.Task
				Expect(svc.NewRetrieve().
					Where(task.MatchKeys(snap.Key)).
					Entry(&retrieved).
					Exec(ctx, tx)).To(Succeed())
				Expect(retrieved.ConfigHash).To(Equal(t.ConfigHash))
			},
		)
	})

	Describe("Config storage", func() {
		It("Should reject a task with an unregistered type", func(ctx SpecContext) {
			t := &task.Task{Type: "made_up", Rack: testRack.Key, Name: "Bad"}
			Expect(w.Create(ctx, t)).To(MatchError(validate.ErrValidation))
			Expect(w.Create(ctx, t)).To(MatchError(ContainSubstring("made_up")))
		})
		It(
			"Should store the config as a record under the task's key",
			func(ctx SpecContext) {
				t := &task.Task{
					Type:   testType,
					Rack:   testRack.Key,
					Name:   "Stored Config Task",
					Config: msgpack.EncodedJSON{"routing_key": "rk-stored"},
				}
				Expect(w.Create(ctx, t)).To(Succeed())
				store := MustBeOk(configs.Store(testType))
				data := MustSucceed(store.Read(ctx, tx, t.Key))
				Expect(data).To(HaveKeyWithValue("routing_key", "rk-stored"))
				Expect(t.Config).To(HaveKeyWithValue("routing_key", "rk-stored"))
			},
		)
		It(
			"Should overwrite the record when a task is re-configured",
			func(ctx SpecContext) {
				t := &task.Task{
					Type:   testType,
					Rack:   testRack.Key,
					Name:   "Reused Record Task",
					Config: msgpack.EncodedJSON{"routing_key": "rk-1"},
				}
				Expect(w.Create(ctx, t)).To(Succeed())
				t.Config = msgpack.EncodedJSON{"routing_key": "rk-2"}
				Expect(w.Create(ctx, t)).To(Succeed())
				store := MustBeOk(configs.Store(testType))
				data := MustSucceed(store.Read(ctx, tx, t.Key))
				Expect(data).To(HaveKeyWithValue("routing_key", "rk-2"))
			},
		)
		It(
			"Should keep the old record when a type change's config is rejected",
			func(ctx SpecContext) {
				t := &task.Task{
					Type:   testType,
					Rack:   testRack.Key,
					Name:   "Type Change Task",
					Config: msgpack.EncodedJSON{"routing_key": "rk-keep"},
				}
				Expect(w.Create(ctx, t)).To(Succeed())
				changed := *t
				changed.Type = arc.TaskType
				changed.Config = msgpack.EncodedJSON{"text": math.NaN()}
				Expect(w.Create(ctx, &changed)).To(MatchError(validate.ErrValidation))
				store := MustBeOk(configs.Store(testType))
				data := MustSucceed(store.Read(ctx, tx, t.Key))
				Expect(data).To(HaveKeyWithValue("routing_key", "rk-keep"))
			},
		)
		It(
			"Should compose the stored config on retrieve",
			func(ctx SpecContext) {
				t := &task.Task{
					Type:   testType,
					Rack:   testRack.Key,
					Name:   "Composed Task",
					Config: msgpack.EncodedJSON{"routing_key": "rk-composed"},
				}
				Expect(w.Create(ctx, t)).To(Succeed())
				var res task.Task
				Expect(svc.NewRetrieve().
					Where(task.MatchKeys(t.Key)).
					Entry(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res.Config).To(HaveKeyWithValue("routing_key", "rk-composed"))
				Expect(res.Config).To(HaveKey("key"))
			},
		)
		It(
			"Should delete the config record with the task",
			func(ctx SpecContext) {
				t := &task.Task{
					Type:   testType,
					Rack:   testRack.Key,
					Name:   "Deleted Record Task",
					Config: msgpack.EncodedJSON{"routing_key": "rk-del"},
				}
				Expect(w.Create(ctx, t)).To(Succeed())
				Expect(w.Delete(ctx, t.Key, false)).To(Succeed())
				store := MustBeOk(configs.Store(testType))
				Expect(
					store.Read(ctx, tx, t.Key),
				).Error().To(MatchError(query.ErrNotFound))
			},
		)
		It(
			"Should copy the config record when a task is copied",
			func(ctx SpecContext) {
				t := &task.Task{
					Type:   testType,
					Rack:   testRack.Key,
					Name:   "Copy Source Task",
					Config: msgpack.EncodedJSON{"routing_key": "rk-copy"},
				}
				Expect(w.Create(ctx, t)).To(Succeed())
				copied := MustSucceed(w.Copy(ctx, t.Key, "Copied", false))
				Expect(copied.Config).To(
					HaveKeyWithValue("routing_key", "rk-copy"),
				)
				Expect(copied.Key).ToNot(Equal(t.Key))
				store := MustBeOk(configs.Store(testType))
				data := MustSucceed(store.Read(ctx, tx, copied.Key))
				Expect(data).To(HaveKeyWithValue("routing_key", "rk-copy"))
			},
		)
		It(
			"Should give an internal task a resource but no group parent",
			func(ctx SpecContext) {
				t := &task.Task{
					Type:     testType,
					Rack:     testRack.Key,
					Name:     "Internal Resource Task",
					Internal: true,
				}
				Expect(w.Create(ctx, t)).To(Succeed())
				Expect(otg.NewRetrieve().
					WhereIDs(t.OntologyID()).
					Exists(ctx, tx)).To(BeTrue())
				var parents []ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(t.OntologyID()).
					TraverseTo(ontology.ParentsTraverser).
					Entries(&parents).
					Exec(ctx, tx)).To(Succeed())
				Expect(parents).To(BeEmpty())
			},
		)
		It(
			"Should give a regular task a group parent",
			func(ctx SpecContext) {
				t := &task.Task{
					Type: testType,
					Rack: testRack.Key,
					Name: "Parented Task",
				}
				Expect(w.Create(ctx, t)).To(Succeed())
				var parents []ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(t.OntologyID()).
					TraverseTo(ontology.ParentsTraverser).
					Entries(&parents).
					Exec(ctx, tx)).To(Succeed())
				Expect(parents).To(HaveLen(1))
				Expect(parents[0].ID.Type).To(Equal(ontology.ResourceTypeGroup))
			},
		)
	})

	Describe("CreateMany", func() {
		It("Should create multiple tasks", func(ctx SpecContext) {
			tasks := []task.Task{
				{
					Type: testType,
					Rack: testRack.Key,
					Name: "Task 1",
				},
				{
					Type: testType,
					Rack: testRack.Key,
					Name: "Task 2",
				},
			}
			Expect(w.CreateMany(ctx, &tasks)).To(Succeed())

			var retrieved []task.Task
			Expect(svc.NewRetrieve().Where(task.MatchKeys(
				tasks[0].Key,
				tasks[1].Key,
			)).Entries(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved).To(HaveLen(2))
		})
	})

	Describe("Copy", func() {
		It("Should copy a task", func(ctx SpecContext) {
			m := &task.Task{
				Type: testType,
				Rack: testRack.Key,
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Name).To(Equal("Test Task"))
			t := MustSucceed(w.Copy(ctx, m.Key, "Copied Task", false))
			Expect(t.Key).ToNot(Equal(uuid.Nil))
			Expect(t.Key).ToNot(Equal(m.Key))
		})

		It("Should create a snapshot of an existing task", func(ctx SpecContext) {
			m := &task.Task{
				Type: testType,
				Rack: testRack.Key,
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Name).To(Equal("Test Task"))
			t := MustSucceed(w.Copy(ctx, m.Key, "Snapshotted Task", true))
			Expect(t.Key).ToNot(Equal(m.Key))
			Expect(t.Snapshot).To(BeTrue())
		})
	})

	Describe("Retrieve", func() {
		It("Should correctly retrieve a task", func(ctx SpecContext) {
			m := &task.Task{
				Type: testType,
				Rack: testRack.Key,
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Name).To(Equal("Test Task"))
			var res task.Task
			Expect(
				svc.NewRetrieve().
					Where(task.MatchKeys(m.Key)).
					Entry(&res).
					Exec(ctx, tx),
			).To(Succeed())
			expected := *m
			// The retrieve does not ask for a status; Create returns the one it wrote.
			expected.Status = nil
			Expect(res).To(Equal(expected))
		})

		It("Should filter tasks by snapshot status", func(ctx SpecContext) {
			regular := &task.Task{
				Type: testType,
				Rack: testRack.Key,
				Name: "Regular Task",
			}
			Expect(w.Create(ctx, regular)).To(Succeed())
			snapshot := &task.Task{
				Type:     testType,
				Rack:     testRack.Key,
				Name:     "Snapshot Task",
				Snapshot: true,
			}
			Expect(w.Create(ctx, snapshot)).To(Succeed())
			var snapshots []task.Task
			Expect(
				svc.NewRetrieve().
					Where(task.MatchSnapshot(true)).
					Entries(&snapshots).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(snapshots).To(HaveLen(1))
			Expect(snapshots[0].Name).To(Equal("Snapshot Task"))
			Expect(snapshots[0].Snapshot).To(BeTrue())
			var regulars []task.Task
			Expect(
				svc.NewRetrieve().
					Where(task.MatchSnapshot(false)).
					Entries(&regulars).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(regulars).ToNot(BeEmpty())
			for _, t := range regulars {
				Expect(t.Snapshot).To(BeFalse())
			}
		})

		It("Should combine MatchSnapshot with other filters", func(ctx SpecContext) {
			snapshot1 := &task.Task{
				Type:     testType,
				Rack:     testRack.Key,
				Name:     "Snapshot Task 1",
				Snapshot: true,
			}
			Expect(w.Create(ctx, snapshot1)).To(Succeed())
			snapshot2 := &task.Task{
				Type:     testType,
				Rack:     testRack.Key,
				Name:     "Snapshot Task 2",
				Snapshot: true,
			}
			Expect(w.Create(ctx, snapshot2)).To(Succeed())
			var res task.Task
			Expect(
				svc.NewRetrieve().
					Where(task.And(task.MatchSnapshot(true), task.MatchNames("Snapshot Task 1"))).
					Entry(&res).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(res.Name).To(Equal("Snapshot Task 1"))
			Expect(res.Snapshot).To(BeTrue())
		})

		It("Should filter tasks by internal status", func(ctx SpecContext) {
			regular := &task.Task{
				Type: testType,
				Rack: testRack.Key,
				Name: "Regular Task 2",
			}
			Expect(w.Create(ctx, regular)).To(Succeed())
			internal := &task.Task{
				Type:     testType,
				Rack:     testRack.Key,
				Name:     "Internal Task",
				Internal: true,
			}
			Expect(w.Create(ctx, internal)).To(Succeed())
			var internals []task.Task
			Expect(
				svc.NewRetrieve().
					Where(task.MatchInternal(true)).
					Entries(&internals).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(internals).To(HaveLen(1))
			Expect(internals[0].Name).To(Equal("Internal Task"))
			Expect(internals[0].Internal).To(BeTrue())
			var regulars []task.Task
			Expect(
				svc.NewRetrieve().
					Where(task.MatchInternal(false)).
					Entries(&regulars).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(regulars).ToNot(BeEmpty())
			for _, t := range regulars {
				Expect(t.Internal).To(BeFalse())
			}
		})

		It("Should combine MatchInternal with other filters", func(ctx SpecContext) {
			internal1 := &task.Task{
				Type:     testType,
				Rack:     testRack.Key,
				Name:     "Internal Task 1",
				Internal: true,
			}
			Expect(w.Create(ctx, internal1)).To(Succeed())
			internal2 := &task.Task{
				Type:     testType,
				Rack:     testRack.Key,
				Name:     "Internal Task 2",
				Internal: true,
			}
			Expect(w.Create(ctx, internal2)).To(Succeed())
			var res task.Task
			Expect(
				svc.NewRetrieve().
					Where(task.And(task.MatchInternal(true), task.MatchNames("Internal Task 1"))).
					Entry(&res).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(res.Name).To(Equal("Internal Task 1"))
			Expect(res.Internal).To(BeTrue())
		})
	})

	Describe("Delete", func() {
		It(
			"Should correctly delete a task and its associated status",
			func(ctx SpecContext) {
				m := &task.Task{
					Type: testType,
					Rack: testRack.Key,
					Name: "Test Task",
				}
				Expect(w.Create(ctx, m)).To(Succeed())
				Expect(w.Delete(ctx, m.Key, false)).To(Succeed())
				Expect(
					svc.NewRetrieve().Where(task.MatchKeys(m.Key)).Exec(ctx, tx),
				).To(MatchError(query.ErrNotFound))
				var deletedStatus task.Status
				Expect(status.NewRetrieve[task.StatusDetails](stat).
					Where(status.MatchKeys[task.StatusDetails](m.OntologyID().String())).
					Entry(&deletedStatus).
					Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
			},
		)

		It("Should delete the task's config record", func(ctx SpecContext) {
			m := &task.Task{
				Type:   testType,
				Rack:   testRack.Key,
				Name:   "Config Record Task",
				Config: msgpack.EncodedJSON{"routing_key": "rk-delete"},
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			store := MustBeOk(configs.Store(testType))
			Expect(store.Read(ctx, tx, m.Key)).Error().ToNot(HaveOccurred())
			Expect(w.Delete(ctx, m.Key, false)).To(Succeed())
			Expect(
				store.Read(ctx, tx, m.Key),
			).Error().To(MatchError(query.ErrNotFound))
		})
	})

	Describe("DeleteByRacks", func() {
		It(
			"Should delete every task on the rack with its config record",
			func(ctx SpecContext) {
				other := &rack.Rack{Name: "Other Rack"}
				Expect(rackService.NewWriter(tx).Create(ctx, other)).To(Succeed())
				onRack := &task.Task{
					Type:     testType,
					Rack:     testRack.Key,
					Name:     "Internal Scan Task",
					Internal: true,
				}
				elsewhere := &task.Task{
					Type: testType,
					Rack: other.Key,
					Name: "Untouched Task",
				}
				Expect(w.Create(ctx, onRack)).To(Succeed())
				Expect(w.Create(ctx, elsewhere)).To(Succeed())
				Expect(w.DeleteByRacks(ctx, testRack.Key)).To(Succeed())
				store := MustBeOk(configs.Store(testType))
				Expect(
					svc.NewRetrieve().Where(task.MatchKeys(onRack.Key)).Exec(ctx, tx),
				).To(MatchError(query.ErrNotFound))
				Expect(
					store.Read(ctx, tx, onRack.Key),
				).Error().To(MatchError(query.ErrNotFound))
				Expect(
					svc.NewRetrieve().
						Where(task.MatchKeys(elsewhere.Key)).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(store.Read(ctx, tx, elsewhere.Key)).Error().ToNot(HaveOccurred())
			},
		)

		It("Should succeed when the rack has no tasks", func(ctx SpecContext) {
			empty := &rack.Rack{Name: "Empty Rack"}
			Expect(rackService.NewWriter(tx).Create(ctx, empty)).To(Succeed())
			Expect(w.DeleteByRacks(ctx, empty.Key)).To(Succeed())
		})
	})

	Describe("Status", func() {
		It(
			"Should create a not-deployed status when creating a task",
			func(ctx SpecContext) {
				m := &task.Task{
					Type: testType,
					Rack: testRack.Key,
					Name: "Status Test Task",
				}
				Expect(w.Create(ctx, m)).To(Succeed())

				var taskStatus task.Status
				Expect(status.NewRetrieve[task.StatusDetails](stat).
					Where(status.MatchKeys[task.StatusDetails](m.OntologyID().String())).
					Entry(&taskStatus).
					Exec(ctx, tx)).To(Succeed())
				Expect(taskStatus.Variant).To(Equal(status.VariantDisabled))
				Expect(
					taskStatus.Message,
				).To(Equal("Status Test Task has not been deployed"))
				Expect(taskStatus.Details.Task).To(Equal(m.Key))
			},
		)

		It(
			"Should use the provided status when creating a task",
			func(ctx SpecContext) {
				providedStatus := &task.Status{
					Variant:     status.VariantSuccess,
					Message:     "Custom task status",
					Description: "Task is running",
					Time:        telem.Now(),
					Details: task.StatusDetails{
						Running: true,
					},
				}
				m := &task.Task{
					Type:   testType,
					Rack:   testRack.Key,
					Name:   "Task with custom status",
					Status: providedStatus,
				}
				Expect(w.Create(ctx, m)).To(Succeed())

				var taskStatus task.Status
				Expect(status.NewRetrieve[task.StatusDetails](stat).
					Where(status.MatchKeys[task.StatusDetails](m.OntologyID().String())).
					Entry(&taskStatus).
					Exec(ctx, tx)).To(Succeed())
				Expect(taskStatus.Variant).To(Equal(status.VariantSuccess))
				Expect(taskStatus.Message).To(Equal("Custom task status"))
				Expect(taskStatus.Description).To(Equal("Task is running"))
				// Key should be auto-assigned
				Expect(taskStatus.Key).To(Equal(m.OntologyID().String()))
				// Name should be auto-filled
				Expect(taskStatus.Name).To(Equal(m.Name))
				// Details.Task should be auto-filled
				Expect(taskStatus.Details.Task).To(Equal(m.Key))
				// Provided details should be preserved
				Expect(taskStatus.Details.Running).To(BeTrue())
			},
		)

		It(
			"Should return a validation error if provided status has empty variant",
			func(ctx SpecContext) {
				providedStatus := &task.Status{
					Time:    telem.Now(),
					Message: "Status with no variant",
				}
				m := &task.Task{
					Type:   testType,
					Rack:   testRack.Key,
					Name:   "Task with invalid status",
					Status: providedStatus,
				}
				Expect(
					w.Create(ctx, m),
				).Error().
					To(MatchError(ContainSubstring("variant")))
			},
		)
		It(
			"Should restore a missing status row when the task is re-configured",
			func(ctx SpecContext) {
				t := &task.Task{
					Type: testType,
					Rack: testRack.Key,
					Name: "Self Heal Task",
				}
				Expect(w.Create(ctx, t)).To(Succeed())

				Expect(status.NewWriter[task.StatusDetails](stat, tx).
					Delete(ctx, t.OntologyID().String())).To(Succeed())
				Expect(status.NewRetrieve[task.StatusDetails](stat).
					Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
					Exec(ctx, tx)).To(MatchError(query.ErrNotFound))

				reconfigured := &task.Task{Key: t.Key, Name: t.Name, Type: testType}
				Expect(w.Create(ctx, reconfigured)).To(Succeed())

				var healed task.Status
				Expect(status.NewRetrieve[task.StatusDetails](stat).
					Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
					Entry(&healed).
					Exec(ctx, tx)).To(Succeed())
				Expect(healed.Details.Task).To(Equal(t.Key))
				Expect(healed.Variant).To(Equal(status.VariantWarning))
				Expect(healed.Message).To(Equal("Self Heal Task status unknown"))
			},
		)

		It(
			"Should not clobber a live status row on a no-op re-configure",
			func(ctx SpecContext) {
				t := &task.Task{
					Type: testType,
					Rack: testRack.Key,
					Name: "Live Status Task",
					Status: &task.Status{
						Variant: status.VariantSuccess,
						Message: "Task is running",
						Time:    telem.Now(),
					},
				}
				Expect(w.Create(ctx, t)).To(Succeed())

				reconfigured := &task.Task{Key: t.Key, Name: t.Name, Type: testType}
				Expect(w.Create(ctx, reconfigured)).To(Succeed())

				var preserved task.Status
				Expect(status.NewRetrieve[task.StatusDetails](stat).
					Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
					Entry(&preserved).
					Exec(ctx, tx)).To(Succeed())
				Expect(preserved.Variant).To(Equal(status.VariantSuccess))
				Expect(preserved.Message).To(Equal("Task is running"))
			},
		)

		It(
			"Should return the status it wrote for a new task",
			func(ctx SpecContext) {
				m := &task.Task{
					Type: testType,
					Rack: testRack.Key,
					Name: "Returned Status Task",
				}
				Expect(w.Create(ctx, m)).To(Succeed())
				Expect(m.Status).ToNot(BeNil())
				Expect(m.Status.Variant).To(Equal(status.VariantDisabled))
				Expect(
					m.Status.Message,
				).To(Equal("Returned Status Task has not been deployed"))
				Expect(m.Status.Details.Task).To(Equal(m.Key))
			},
		)

		It(
			"Should return the live status on a re-configure rather than a placeholder",
			func(ctx SpecContext) {
				t := &task.Task{
					Type: testType,
					Rack: testRack.Key,
					Name: "Live Returned Status Task",
					Status: &task.Status{
						Variant: status.VariantSuccess,
						Message: "Task is running",
						Time:    telem.Now(),
						Details: task.StatusDetails{Running: true},
					},
				}
				Expect(w.Create(ctx, t)).To(Succeed())

				reconfigured := &task.Task{Key: t.Key, Name: t.Name, Type: testType}
				Expect(w.Create(ctx, reconfigured)).To(Succeed())
				Expect(reconfigured.Status).ToNot(BeNil())
				Expect(reconfigured.Status.Variant).To(Equal(status.VariantSuccess))
				Expect(reconfigured.Status.Message).To(Equal("Task is running"))
				Expect(reconfigured.Status.Details.Running).To(BeTrue())
			},
		)

		It(
			"Should create a not-deployed status when copying a task",
			func(ctx SpecContext) {
				m := &task.Task{
					Type: testType,
					Rack: testRack.Key,
					Name: "Original Task",
				}
				Expect(w.Create(ctx, m)).To(Succeed())

				copied := MustSucceed(w.Copy(ctx, m.Key, "Copied Task", false))

				var copiedStatus task.Status
				Expect(status.NewRetrieve[task.StatusDetails](stat).
					Where(status.MatchKeys[task.StatusDetails](copied.OntologyID().String())).
					Entry(&copiedStatus).
					Exec(ctx, tx)).To(Succeed())
				Expect(copiedStatus.Variant).To(Equal(status.VariantDisabled))
				Expect(
					copiedStatus.Message,
				).To(Equal("Copied Task has not been deployed"))
				Expect(copiedStatus.Details.Task).To(Equal(copied.Key))
			},
		)
	})

	Describe("Suspect Rack", func() {
		It(
			"Should propagate rack warning status to tasks on that rack",
			func(ctx SpecContext) {
				r := rack.Rack{Name: "suspect rack"}
				Expect(rackService.NewWriter(nil).Create(ctx, &r)).To(Succeed())

				t := &task.Task{
					Type: testType,
					Rack: r.Key,
					Name: "Test Task",
				}
				Expect(svc.NewWriter(nil).Create(ctx, t)).To(Succeed())

				Eventually(func(g Gomega) {
					var taskStatus task.Status
					g.Expect(status.NewRetrieve[task.StatusDetails](stat).
						Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
						Entry(&taskStatus).
						Exec(ctx, nil)).To(Succeed())
					g.Expect(taskStatus.Variant).To(Equal(status.VariantWarning))
					g.Expect(taskStatus.Message).To(ContainSubstring("not running"))
					g.Expect(taskStatus.Details.Task).To(Equal(t.Key))
				}).Should(Succeed())
			},
		)

		It(
			"Should preserve the config hash and rack the Driver reported",
			func(ctx SpecContext) {
				r := rack.Rack{Name: "suspect rack"}
				Expect(rackService.NewWriter(nil).Create(ctx, &r)).To(Succeed())

				t := &task.Task{
					Type: testType,
					Rack: r.Key,
					Name: "Test Task",
					Status: &task.Status{
						Variant: status.VariantSuccess,
						Message: "Task is running",
						Time:    telem.Now(),
						Details: task.StatusDetails{
							Running:    true,
							ConfigHash: "deployed",
							Rack:       r.Key,
						},
					},
				}
				Expect(svc.NewWriter(nil).Create(ctx, t)).To(Succeed())

				Eventually(func(g Gomega) {
					var taskStatus task.Status
					g.Expect(status.NewRetrieve[task.StatusDetails](stat).
						Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
						Entry(&taskStatus).
						Exec(ctx, nil)).To(Succeed())
					g.Expect(taskStatus.Variant).To(Equal(status.VariantWarning))
					g.Expect(taskStatus.Details.Running).To(BeFalse())
					g.Expect(taskStatus.Details.ConfigHash).To(Equal("deployed"))
					g.Expect(taskStatus.Details.Rack).To(Equal(r.Key))
				}).Should(Succeed())
			},
		)
	})

	Describe("Command", func() {
		Describe("String", func() {
			It(
				"Should return a string representation of the command",
				func(ctx SpecContext) {
					k := uuid.New()
					c := &task.Command{
						Key:  "cmd",
						Task: k,
						Type: "doc",
					}
					Expect(
						c.String(),
					).To(Equal("doc (key=cmd, task=" + k.String() + ")"))
				},
			)
		})
	})

	Describe("NewStatusDetails", func() {
		It("Should echo the task's key, config hash, and rack", func() {
			k := uuid.New()
			t := task.Task{
				Key:        k,
				Rack:       testRack.Key,
				ConfigHash: "hash1",
			}
			Expect(task.NewStatusDetails(t, true)).To(Equal(task.StatusDetails{
				Task:       k,
				Running:    true,
				ConfigHash: "hash1",
				Rack:       testRack.Key,
			}))
		})

		It("Should mark the task as not running when running is false", func() {
			d := task.NewStatusDetails(task.Task{Key: uuid.New()}, false)
			Expect(d.Running).To(BeFalse())
		})
	})

	Describe("Observe", func() {
		It("Should notify when a task is created", func(ctx SpecContext) {
			tx := db.OpenTx()
			defer func() { Expect(tx.Close()).To(Succeed()) }()
			w := svc.NewWriter(tx)
			t := &task.Task{
				Rack: testRack.Key,
				Name: "observe-test",
				Type: testType,
			}
			Expect(w.Create(ctx, t)).To(Succeed())
			called := false
			svc.Observe().
				OnChange(func(ctx context.Context, _ gorp.TxReader[task.Key, task.Task]) {
					called = true
				})
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(called).To(BeTrue())
		})
	})
})
