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
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

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
		svc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Rack:     rackService,
			Status:   stat,
			Search:   searchIdx,
			ImEx:     imex.NewService(),
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
	Describe("Task", func() {
		It("Should lease a racked task to the rack's node", func(ctx SpecContext) {
			rk := rack.NewKey(node.Key(1), 1)
			t := task.Task{Key: uuid.New(), Rack: rk}
			Expect(t.SetOptions()).To(Equal([]any{rk.Node()}))
		})
		It("Should use the default lease for a rackless draft", func(ctx SpecContext) {
			t := task.Task{Key: uuid.New()}
			Expect(t.SetOptions()).To(BeNil())
		})
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
				Key:  key,
				Rack: testRack.Key,
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).To(Equal(key))
		})
		It("Should create a rackless draft", func(ctx SpecContext) {
			m := &task.Task{Name: "Draft Task"}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).ToNot(Equal(uuid.Nil))
			Expect(m.Rack.IsZero()).To(BeTrue())
		})
	})

	Describe("ConfigHash", func() {
		create := func(ctx context.Context, config msgpack.EncodedJSON) string {
			t := &task.Task{Rack: testRack.Key, Name: "Test Task", Config: config}
			Expect(w.Create(ctx, t)).To(Succeed())
			return t.ConfigHash
		}
		It("Should assign a hash on create", func(ctx SpecContext) {
			Expect(create(ctx, msgpack.EncodedJSON{"rate": 10})).ToNot(BeEmpty())
		})
		It(
			"Should hash equal configs equally regardless of key order",
			func(ctx SpecContext) {
				first := create(ctx, msgpack.EncodedJSON{"rate": 10, "port": "COM1"})
				second := create(ctx, msgpack.EncodedJSON{"port": "COM1", "rate": 10})
				Expect(first).To(Equal(second))
			},
		)
		It("Should hash differing configs differently", func(ctx SpecContext) {
			first := create(ctx, msgpack.EncodedJSON{"rate": 10})
			second := create(ctx, msgpack.EncodedJSON{"rate": 20})
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
				Expect(create(ctx, msgpack.EncodedJSON{"rate": 50})).
					To(Equal(create(ctx, msgpack.EncodedJSON{"rate": 50.0})))
			},
		)
		It(
			"Should restore the original hash when an edit is undone",
			func(ctx SpecContext) {
				t := &task.Task{
					Rack:   testRack.Key,
					Name:   "Test Task",
					Config: msgpack.EncodedJSON{"rate": 10},
				}
				Expect(w.Create(ctx, t)).To(Succeed())
				original := t.ConfigHash
				t.Config = msgpack.EncodedJSON{"rate": 20}
				Expect(w.Create(ctx, t)).To(Succeed())
				Expect(t.ConfigHash).ToNot(Equal(original))
				t.Config = msgpack.EncodedJSON{"rate": 10}
				Expect(w.Create(ctx, t)).To(Succeed())
				Expect(t.ConfigHash).To(Equal(original))
			},
		)
		It("Should ignore a client-provided hash", func(ctx SpecContext) {
			t := &task.Task{
				Rack:       testRack.Key,
				Name:       "Test Task",
				Config:     msgpack.EncodedJSON{"rate": 10},
				ConfigHash: "deadbeefdeadbeef",
			}
			Expect(w.Create(ctx, t)).To(Succeed())
			Expect(t.ConfigHash).ToNot(Equal("deadbeefdeadbeef"))
			Expect(t.ConfigHash).To(Equal(create(ctx, msgpack.EncodedJSON{"rate": 10})))
		})
		// An unhashable config must not store an empty hash, which a driver would echo
		// back as a match and silently disable drift detection for the task.
		It("Should reject a config that cannot be hashed", func(ctx SpecContext) {
			t := &task.Task{
				Rack:   testRack.Key,
				Name:   "Test Task",
				Config: msgpack.EncodedJSON{"rate": math.NaN()},
			}
			Expect(
				w.Create(ctx, t),
			).To(MatchError(ContainSubstring("hash task config")))
		})
		It("Should persist the hash alongside the task", func(ctx SpecContext) {
			t := &task.Task{
				Rack:   testRack.Key,
				Name:   "Test Task",
				Config: msgpack.EncodedJSON{"rate": 10},
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
					Rack:   testRack.Key,
					Name:   "Test Task",
					Config: msgpack.EncodedJSON{"rate": 10},
				}
				Expect(w.Create(ctx, t)).To(Succeed())
				snap := MustSucceed(w.Copy(ctx, t.Key, "Snapshot", true))
				Expect(snap.ConfigHash).To(Equal(t.ConfigHash))
				edited := snap
				edited.Config = msgpack.EncodedJSON{"rate": 999}
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

	Describe("CreateMany", func() {
		It("Should create multiple tasks", func(ctx SpecContext) {
			tasks := []task.Task{
				{
					Rack: testRack.Key,
					Name: "Task 1",
				},
				{
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
			Expect(res).To(Equal(*m))
		})

		It("Should filter tasks by snapshot status", func(ctx SpecContext) {
			regular := &task.Task{
				Rack: testRack.Key,
				Name: "Regular Task",
			}
			Expect(w.Create(ctx, regular)).To(Succeed())
			snapshot := &task.Task{
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
				Rack:     testRack.Key,
				Name:     "Snapshot Task 1",
				Snapshot: true,
			}
			Expect(w.Create(ctx, snapshot1)).To(Succeed())
			snapshot2 := &task.Task{
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
				Rack: testRack.Key,
				Name: "Regular Task 2",
			}
			Expect(w.Create(ctx, regular)).To(Succeed())
			internal := &task.Task{
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
				Rack:     testRack.Key,
				Name:     "Internal Task 1",
				Internal: true,
			}
			Expect(w.Create(ctx, internal1)).To(Succeed())
			internal2 := &task.Task{
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
	})

	Describe("Status", func() {
		It(
			"Should create an unknown status when creating a task",
			func(ctx SpecContext) {
				m := &task.Task{
					Rack: testRack.Key,
					Name: "Status Test Task",
				}
				Expect(w.Create(ctx, m)).To(Succeed())

				var taskStatus task.Status
				Expect(status.NewRetrieve[task.StatusDetails](stat).
					Where(status.MatchKeys[task.StatusDetails](m.OntologyID().String())).
					Entry(&taskStatus).
					Exec(ctx, tx)).To(Succeed())
				Expect(taskStatus.Variant).To(Equal(status.VariantWarning))
				Expect(taskStatus.Message).To(Equal("Status Test Task status unknown"))
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
					Rack: testRack.Key,
					Name: "Self Heal Task",
				}
				Expect(w.Create(ctx, t)).To(Succeed())

				Expect(status.NewWriter[task.StatusDetails](stat, tx).
					Delete(ctx, t.OntologyID().String())).To(Succeed())
				Expect(status.NewRetrieve[task.StatusDetails](stat).
					Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
					Exec(ctx, tx)).To(MatchError(query.ErrNotFound))

				reconfigured := &task.Task{Key: t.Key, Name: t.Name}
				Expect(w.Create(ctx, reconfigured)).To(Succeed())

				var healed task.Status
				Expect(status.NewRetrieve[task.StatusDetails](stat).
					Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
					Entry(&healed).
					Exec(ctx, tx)).To(Succeed())
				Expect(healed.Details.Task).To(Equal(t.Key))
			},
		)

		It(
			"Should not clobber a live status row on a no-op re-configure",
			func(ctx SpecContext) {
				t := &task.Task{
					Rack: testRack.Key,
					Name: "Live Status Task",
					Status: &task.Status{
						Variant: status.VariantSuccess,
						Message: "Task is running",
						Time:    telem.Now(),
					},
				}
				Expect(w.Create(ctx, t)).To(Succeed())

				reconfigured := &task.Task{Key: t.Key, Name: t.Name}
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
			"Should create an unknown status when copying a task",
			func(ctx SpecContext) {
				m := &task.Task{
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
				Expect(copiedStatus.Variant).To(Equal(status.VariantWarning))
				Expect(copiedStatus.Message).To(Equal("Copied Task status unknown"))
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

	Describe("Observe", func() {
		It("Should notify when a task is created", func(ctx SpecContext) {
			tx := db.OpenTx()
			defer func() { Expect(tx.Close()).To(Succeed()) }()
			w := svc.NewWriter(tx)
			t := &task.Task{
				Rack: testRack.Key,
				Name: "observe-test",
				Type: "test",
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
