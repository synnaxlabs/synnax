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
	"fmt"
	"math"
	"slices"
	"sync"
	"sync/atomic"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/labjack"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

const testType = pagerduty.AlertTaskType

var _ = Describe("Task", Ordered, func() {
	var (
		db          *gorp.DB
		svc         *task.Service
		w           task.Writer
		tx          gorp.Tx
		rackService *rack.Service
		testRack    *rack.Rack
		stat        *status.Service
		configs     config.Registry
		testArc     arc.Key
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB: db,
		}))
		stat = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB: db,
		}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
			DB: db,
		}))
		rackService = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:                  db,
			HostKey:             1,
			Status:              stat,
			HealthCheckInterval: 10 * telem.Millisecond,
			DriverRole: MustSucceed(
				racktestutil.CreateDriverRole(ctx, db, roleSvc),
			),
		}))
		pd := MustOpen(pagerduty.OpenService(ctx, pagerduty.ServiceConfig{DB: db}))
		MustOpen(arc.OpenTable(ctx, arc.TableConfig{DB: db}))
		a := arc.Arc{
			Variant: arc.AutomationArc{
				Base: arc.Base{
					Key:    uuid.New(),
					Name:   "Test Arc",
					Source: arc.Source{Variant: arc.TextSource{}},
				},
			},
		}
		Expect(gorp.NewCreate[arc.Key, arc.Arc]().Entry(&a).Exec(ctx, db)).
			To(Succeed())
		testArc = a.Base().Key
		at := MustOpen(arctask.OpenService(ctx, arctask.ServiceConfig{DB: db}))
		lj := MustOpen(labjack.OpenService(ctx, labjack.ServiceConfig{DB: db}))
		configs = MustSucceed(config.NewRegistry(
			slices.Concat(pd.Stores(), at.Stores(), lj.Stores())...,
		))
		svc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
			DB:      db,
			Rack:    rackService,
			Status:  stat,
			ImEx:    imex.NewService(),
			Configs: configs,
		}))
		testRack = &rack.Rack{Name: "Test Rack"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return rackService.NewWriter(tx).Create(ctx, testRack)
		})).To(Succeed())
		for _, k := range []status.Key{"st-1", "st-2", "st-a", "st-b"} {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return stat.NewWriter(tx).Set(ctx, &status.Status[any]{
					Key:     k,
					Name:    string(k),
					Variant: status.VariantInfo,
					Time:    telem.Now(),
				})
			})).To(Succeed())
		}
	})
	openTx := func() {
		tx = DeferClose(db.OpenTx())
		w = svc.NewWriter(tx)
	}
	type frame = actions.Scoped[task.Key, task.Action]
	collect := func() *[]frame {
		GinkgoHelper()
		var frames []frame
		DeferCleanup(svc.OnAction(func(_ context.Context, f frame) {
			frames = append(frames, f)
		}))
		return &frames
	}
	Describe("CommandChannelKey", func() {
		It(
			"Should return zero when no channel service is configured",
			func(ctx SpecContext) {
				Expect(svc.CommandChannelKey()).To(Equal(channel.Key(0)))
			},
		)
	})
	Describe("StatusDetails msgpack round-trip", func() {
		It("Should round-trip the details", func(ctx SpecContext) {
			original := task.StatusDetails{
				Rack:    testRack.Key,
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
			openTx()
			m := &task.Task{
				Type: testType,
				Rack: testRack.Key,
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).ToNot(Equal(uuid.Nil()))
			Expect(m.Name).To(Equal("Test Task"))
		})
		It("Should preserve a client-provided key", func(ctx SpecContext) {
			openTx()
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
			openTx()
			m := &task.Task{Type: testType, Name: "Draft Task"}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Key).ToNot(Equal(uuid.Nil()))
			Expect(m.Rack.IsZero()).To(BeTrue())
		})
		It("Should reject a task with an empty name", func(ctx SpecContext) {
			openTx()
			m := &task.Task{Type: testType, Rack: testRack.Key}
			Expect(w.Create(ctx, m)).To(MatchError(ContainSubstring("name: required")))
		})
		It(
			"Should broadcast a create with the dispatch key after the commit",
			func(ctx SpecContext) {
				frames := collect()
				createTx := db.OpenTx()
				defer func() { Expect(createTx.Close()).To(Succeed()) }()
				m := &task.Task{
					Type:   testType,
					Rack:   testRack.Key,
					Name:   "Broadcast",
					Config: msgpack.EncodedJSON{"routing_key": "rk-1"},
				}
				Expect(svc.NewWriter(createTx).
					WithDispatchKey("dk-create").
					Create(ctx, m)).To(Succeed())
				Expect(*frames).To(BeEmpty())
				Expect(createTx.Commit(ctx)).To(Succeed())
				Expect(*frames).To(HaveLen(1))
				f := (*frames)[0]
				Expect(f.Key).To(Equal(m.Key))
				Expect(f.DispatchKey).To(Equal("dk-create"))
				Expect(f.Actions).To(Equal([]task.Action{
					task.NewCreateAction(task.CreatePayload{Task: *m}),
				}))
			},
		)
		It(
			"Should not broadcast when the transaction closes without a commit",
			func(ctx SpecContext) {
				frames := collect()
				discardTx := db.OpenTx()
				Expect(svc.NewWriter(discardTx).Create(ctx, &task.Task{
					Type: testType,
					Rack: testRack.Key,
					Name: "Discarded",
				})).To(Succeed())
				Expect(discardTx.Close()).To(Succeed())
				Expect(*frames).To(BeEmpty())
			},
		)
	})

	Describe("ConfigHash", func() {
		BeforeEach(openTx)
		create := func(ctx context.Context, config msgpack.EncodedJSON) string {
			GinkgoHelper()
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
		// A row deleted and re-added carries a fresh key, which must not read as a
		// config change and ask the user to redeploy.
		It("Should ignore the keys of nested rows", func(ctx SpecContext) {
			alert := func(key string) map[string]any {
				return map[string]any{"key": key, "status": "st-1"}
			}
			first := create(ctx, msgpack.EncodedJSON{
				"routing_key": "rk-1", "alerts": []any{alert("a-1")},
			})
			second := create(ctx, msgpack.EncodedJSON{
				"routing_key": "rk-1", "alerts": []any{alert("a-2")},
			})
			Expect(first).To(Equal(second))
		})
		It("Should hash nested rows that differ in content differently", func(
			ctx SpecContext,
		) {
			alert := func(status string) map[string]any {
				return map[string]any{"key": "a-1", "status": status}
			}
			first := create(ctx, msgpack.EncodedJSON{"alerts": []any{alert("st-1")}})
			second := create(ctx, msgpack.EncodedJSON{"alerts": []any{alert("st-2")}})
			Expect(first).ToNot(Equal(second))
		})
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
				createRead := func(config msgpack.EncodedJSON) string {
					GinkgoHelper()
					t := &task.Task{
						Type:   "labjack_read",
						Rack:   testRack.Key,
						Name:   "LabJack Read",
						Config: config,
					}
					Expect(w.Create(ctx, t)).To(Succeed())
					return t.ConfigHash
				}
				Expect(createRead(msgpack.EncodedJSON{"sample_rate": 50})).
					To(Equal(createRead(msgpack.EncodedJSON{"sample_rate": 50.0})))
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
	})

	Describe("Config storage", func() {
		BeforeEach(openTx)
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
				copied := MustSucceed(w.Copy(ctx, t.Key, "Copied"))
				Expect(copied.Config).To(
					HaveKeyWithValue("routing_key", "rk-copy"),
				)
				Expect(copied.Key).ToNot(Equal(t.Key))
				store := MustBeOk(configs.Store(testType))
				data := MustSucceed(store.Read(ctx, tx, copied.Key))
				Expect(data).To(HaveKeyWithValue("routing_key", "rk-copy"))
			},
		)
	})

	Describe("CreateMany", func() {
		BeforeEach(openTx)
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
			openTx()
			m := &task.Task{
				Type: testType,
				Rack: testRack.Key,
				Name: "Test Task",
			}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(m.Name).To(Equal("Test Task"))
			t := MustSucceed(w.Copy(ctx, m.Key, "Copied Task"))
			Expect(t.Key).ToNot(Equal(uuid.Nil()))
			Expect(t.Key).ToNot(Equal(m.Key))
		})

		It("Should reject a copy with an empty name", func(ctx SpecContext) {
			openTx()
			m := &task.Task{Type: testType, Rack: testRack.Key, Name: "Test Task"}
			Expect(w.Create(ctx, m)).To(Succeed())
			Expect(w.Copy(ctx, m.Key, "")).Error().
				To(MatchError(ContainSubstring("name: required")))
		})

		It(
			"Should broadcast a create of the copy after the commit",
			func(ctx SpecContext) {
				m := &task.Task{
					Type:   testType,
					Rack:   testRack.Key,
					Name:   "Original",
					Config: msgpack.EncodedJSON{"routing_key": "rk-1"},
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, m)
				})).To(Succeed())
				openTx()
				frames := collect()
				copied := MustSucceed(w.WithDispatchKey("dk-copy").
					Copy(ctx, m.Key, "Copied"))
				Expect(*frames).To(BeEmpty())
				Expect(tx.Commit(ctx)).To(Succeed())
				DeferCleanup(func(ctx SpecContext) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Delete(ctx, m.Key, true)
					})).To(Succeed())
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Delete(ctx, copied.Key, true)
					})).To(Succeed())
				})
				Expect(*frames).To(HaveLen(1))
				f := (*frames)[0]
				Expect(f.Key).To(Equal(copied.Key))
				Expect(f.DispatchKey).To(Equal("dk-copy"))
				Expect(f.Actions).To(Equal([]task.Action{
					task.NewCreateAction(task.CreatePayload{Task: copied}),
				}))
				Expect(copied.Config).To(HaveKeyWithValue("routing_key", "rk-1"))
			},
		)
	})

	Describe("Dispatch", func() {
		// Each dispatch commits in a transaction of its own, so its tasks commit too
		// and are deleted after each spec.
		createCommitted := func(ctx context.Context, t *task.Task) {
			GinkgoHelper()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, t)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Delete(ctx, t.Key, true)
				})).To(Succeed())
			})
		}
		var alerts *task.Task
		BeforeEach(func(ctx SpecContext) {
			alerts = &task.Task{
				Type: testType,
				Rack: testRack.Key,
				Name: "Alerts",
				Config: msgpack.EncodedJSON{
					"routing_key": "rk-1",
					"alerts":      []any{map[string]any{"key": "a", "status": "st-a"}},
				},
			}
			createCommitted(ctx, alerts)
		})
		set := func(path []string, value any) task.Action {
			return task.NewSetAction(task.SetPayload{Path: path, Value: value})
		}
		dispatch := func(
			ctx context.Context,
			key task.Key,
			dispatchKey string,
			acts []task.Action,
		) ([]task.Action, error) {
			var added []task.Action
			if err := db.WithTx(ctx, func(tx gorp.Tx) (err error) {
				added, err = svc.NewWriter(tx).
					WithDispatchKey(dispatchKey).
					Apply(ctx, key, acts...)
				return err
			}); err != nil {
				return nil, err
			}
			return added, nil
		}
		retrieve := func(ctx context.Context, key task.Key) task.Task {
			GinkgoHelper()
			var res task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchKeys(key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			return res
		}
		expectUnchanged := func(ctx context.Context) {
			GinkgoHelper()
			res := retrieve(ctx, alerts.Key)
			Expect(res.Name).To(Equal(alerts.Name))
			Expect(res.Config).To(Equal(alerts.Config))
			Expect(res.ConfigHash).To(Equal(alerts.ConfigHash))
		}

		It("Should store the new config and its hash", func(ctx SpecContext) {
			MustSucceed(dispatch(ctx, alerts.Key, "dk", []task.Action{
				set([]string{"config", "routing_key"}, "rk-2"),
			}))
			res := retrieve(ctx, alerts.Key)
			Expect(res.Config["routing_key"]).To(Equal("rk-2"))
			Expect(res.ConfigHash).ToNot(Equal(alerts.ConfigHash))
		})

		It("Should hash the config as a create of the same config does", func(
			ctx SpecContext,
		) {
			MustSucceed(dispatch(ctx, alerts.Key, "dk", []task.Action{
				set([]string{"config", "routing_key"}, "rk-2"),
			}))
			same := &task.Task{
				Type: testType,
				Rack: testRack.Key,
				Name: "Same",
				Config: msgpack.EncodedJSON{
					"routing_key": "rk-2",
					"alerts":      []any{map[string]any{"key": "a", "status": "st-a"}},
				},
			}
			createCommitted(ctx, same)
			Expect(same.ConfigHash).To(Equal(retrieve(ctx, alerts.Key).ConfigHash))
		})

		It("Should apply list actions to the config", func(ctx SpecContext) {
			MustSucceed(dispatch(ctx, alerts.Key, "dk", []task.Action{
				task.NewInsertAction(task.InsertPayload{
					Path:  []string{"config", "alerts"},
					Value: map[string]any{"key": "b", "status": "st-b"},
					After: new("a"),
				}),
				task.NewMoveAction(task.MovePayload{
					Path: []string{"config", "alerts"},
					Key:  "b",
				}),
				task.NewRemoveAction(task.RemovePayload{
					Path: []string{"config", "alerts"},
					Key:  "a",
				}),
			}))
			res := retrieve(ctx, alerts.Key)
			Expect(res.Config["alerts"]).To(HaveExactElements(
				HaveKeyWithValue("key", "b"),
			))
		})

		It("Should set a field of a list entry by its key", func(ctx SpecContext) {
			MustSucceed(dispatch(ctx, alerts.Key, "dk", []task.Action{
				set([]string{"config", "alerts", "a", "status"}, "st-b"),
			}))
			Expect(retrieve(ctx, alerts.Key).Config["alerts"]).To(HaveExactElements(
				SatisfyAll(
					HaveKeyWithValue("key", "a"),
					HaveKeyWithValue("status", "st-b"),
				),
			))
		})

		It("Should do nothing for a path through a missing list entry", func(
			ctx SpecContext,
		) {
			computed := MustSucceed(dispatch(ctx, alerts.Key, "dk", []task.Action{
				set([]string{"config", "alerts", "z", "status"}, "st-b"),
			}))
			Expect(computed).To(Equal([]task.Action{
				set([]string{"config_hash"}, alerts.ConfigHash),
			}))
			expectUnchanged(ctx)
		})

		It("Should leave the config and its hash alone for a rename", func(
			ctx SpecContext,
		) {
			computed := MustSucceed(dispatch(ctx, alerts.Key, "dk", []task.Action{
				set([]string{"name"}, "Renamed"),
			}))
			Expect(computed).To(BeEmpty())
			res := retrieve(ctx, alerts.Key)
			Expect(res.Config).To(Equal(alerts.Config))
			Expect(res.ConfigHash).To(Equal(alerts.ConfigHash))
		})

		It("Should rename the task and its status", func(ctx SpecContext) {
			MustSucceed(dispatch(ctx, alerts.Key, "dk", []task.Action{
				set([]string{"name"}, "Renamed"),
			}))
			Expect(retrieve(ctx, alerts.Key).Name).To(Equal("Renamed"))
			var renamed task.Status
			Expect(stat.NewRetrieve[task.StatusDetails]().
				Where(status.MatchOwners[task.StatusDetails](alerts.ResourceID())).
				Entry(&renamed).
				Exec(ctx, db)).To(Succeed())
			Expect(renamed.Name).To(Equal("Renamed"))
		})

		It("Should rename a task that has no status", func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return stat.NewWriter(tx).DeleteOwned(ctx, alerts.ResourceID())
			})).
				To(Succeed())
			MustSucceed(dispatch(ctx, alerts.Key, "dk", []task.Action{
				set([]string{"name"}, "Renamed"),
			}))
			Expect(retrieve(ctx, alerts.Key).Name).To(Equal("Renamed"))
		})

		It("Should move the task to another rack", func(ctx SpecContext) {
			other := &rack.Rack{Name: "Other Rack"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return rackService.NewWriter(tx).Create(ctx, other)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return rackService.NewWriter(tx).Delete(ctx, nil, other.Key)
				})).To(Succeed())
			})
			MustSucceed(dispatch(ctx, alerts.Key, "dk", []task.Action{
				set([]string{"rack"}, other.Key),
			}))
			Expect(retrieve(ctx, alerts.Key).Rack).To(Equal(other.Key))
		})

		It("Should apply none of the actions when one fails", func(ctx SpecContext) {
			Expect(dispatch(ctx, alerts.Key, "dk", []task.Action{
				set([]string{"config", "routing_key"}, "rk-2"),
				set([]string{"type"}, "arc"),
			})).Error().To(MatchError(validate.ErrValidation))
			expectUnchanged(ctx)
		})

		It("Should reject a member the config does not declare", func(
			ctx SpecContext,
		) {
			Expect(dispatch(ctx, alerts.Key, "dk", []task.Action{
				set([]string{"config", "routng_key"}, "rk-2"),
			})).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("the path names no editable field")),
			))
			expectUnchanged(ctx)
		})

		It("Should reject a config that fails validation", func(ctx SpecContext) {
			arcTask := &task.Task{
				Type:   arctask.Type,
				Rack:   testRack.Key,
				Name:   "Arc",
				Config: msgpack.EncodedJSON{"arc_key": testArc.String()},
			}
			createCommitted(ctx, arcTask)
			Expect(dispatch(ctx, arcTask.Key, "dk", []task.Action{
				set([]string{"config", "performance"}, "BOGUS"),
			})).Error().To(MatchError(ContainSubstring("invalid performance: BOGUS")))
		})

		It("Should reject an empty name", func(ctx SpecContext) {
			Expect(dispatch(ctx, alerts.Key, "dk", []task.Action{
				set([]string{"name"}, ""),
			})).Error().To(MatchError(ContainSubstring("name: required")))
			expectUnchanged(ctx)
		})

		It("Should reject a create", func(ctx SpecContext) {
			overwritten := *alerts
			overwritten.Name = "Overwritten"
			Expect(dispatch(ctx, alerts.Key, "dk", []task.Action{
				task.NewCreateAction(task.CreatePayload{Task: overwritten}),
			})).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("only the Core dispatches a task create")),
			))
			expectUnchanged(ctx)
		})

		It("Should return not found for a missing task", func(ctx SpecContext) {
			Expect(dispatch(ctx, uuid.New(), "dk", []task.Action{
				set([]string{"name"}, "Renamed"),
			})).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should broadcast and return a set of the new config hash", func(
			ctx SpecContext,
		) {
			frames := collect()
			acts := []task.Action{set([]string{"config", "routing_key"}, "rk-2")}
			computed := MustSucceed(dispatch(ctx, alerts.Key, "dk-1", acts))
			hash := retrieve(ctx, alerts.Key).ConfigHash
			Expect(
				computed,
			).To(Equal([]task.Action{set([]string{"config_hash"}, hash)}))
			Expect(*frames).To(HaveLen(1))
			f := (*frames)[0]
			Expect(f.Key).To(Equal(alerts.Key))
			Expect(f.DispatchKey).To(Equal("dk-1"))
			Expect(f.Seq).To(BeNumerically(">", 0))
			Expect(f.Actions).To(Equal(append(acts, computed...)))
		})

		It(
			"Should broadcast and return the stored config when it fills a default",
			func(
				ctx SpecContext,
			) {
				arcTask := &task.Task{
					Type: arctask.Type,
					Rack: testRack.Key,
					Name: "Arc",
					Config: msgpack.EncodedJSON{
						"arc_key":     testArc.String(),
						"performance": "high",
					},
				}
				createCommitted(ctx, arcTask)
				frames := collect()
				acts := []task.Action{set([]string{"config", "performance"}, "")}
				computed := MustSucceed(dispatch(ctx, arcTask.Key, "dk", acts))
				stored := retrieve(ctx, arcTask.Key)
				Expect(stored.Config).To(HaveKeyWithValue("performance", "auto"))
				Expect(computed).To(Equal([]task.Action{
					set([]string{"config"}, stored.Config),
					set([]string{"config_hash"}, stored.ConfigHash),
				}))
				Expect(*frames).To(HaveLen(1))
				Expect((*frames)[0].Actions).To(Equal(append(acts, computed...)))
			},
		)

		It("Should not broadcast a rejected dispatch", func(ctx SpecContext) {
			frames := collect()
			Expect(dispatch(ctx, alerts.Key, "dk", []task.Action{
				set([]string{"type"}, "arc"),
			})).Error().To(MatchError(validate.ErrValidation))
			Expect(*frames).To(BeEmpty())
		})

		It("Should hash and broadcast a config that a status delete detaches", func(
			ctx SpecContext,
		) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return stat.NewWriter(tx).Set(ctx, &status.Status[any]{
					Key:     "st-detach",
					Name:    "st-detach",
					Variant: status.VariantInfo,
					Time:    telem.Now(),
				})
			})).To(Succeed())
			detached := &task.Task{
				Type: testType,
				Rack: testRack.Key,
				Name: "Detached",
				Config: msgpack.EncodedJSON{
					"routing_key": "rk-1",
					"alerts": []any{
						map[string]any{"key": "a", "status": "st-detach"},
					},
				},
			}
			createCommitted(ctx, detached)
			frames := collect()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return stat.NewWriter(tx).Delete(ctx, "st-detach")
			})).To(Succeed())
			stored := retrieve(ctx, detached.Key)
			Expect(stored.Config["alerts"]).To(HaveExactElements(
				HaveKeyWithValue("status", ""),
			))
			twin := &task.Task{
				Type:   testType,
				Rack:   testRack.Key,
				Name:   "Twin",
				Config: stored.Config,
			}
			Expect(svc.NewWriter(DeferClose(db.OpenTx())).Create(ctx, twin)).
				To(Succeed())
			Expect(stored.ConfigHash).To(Equal(twin.ConfigHash))
			Expect(stored.ConfigHash).ToNot(Equal(detached.ConfigHash))
			Expect(*frames).To(HaveLen(1))
			Expect((*frames)[0].Key).To(Equal(detached.Key))
			Expect((*frames)[0].Actions).To(Equal([]task.Action{
				set([]string{"config"}, stored.Config),
				set([]string{"config_hash"}, stored.ConfigHash),
			}))
		})
	})

	It("Should retrieve a task and its config through a snapshot", func(
		ctx SpecContext,
	) {
		t := &task.Task{
			Type:   testType,
			Rack:   testRack.Key,
			Name:   "Snapshot Task",
			Config: msgpack.EncodedJSON{"routing_key": "rk-snapshot"},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, t)
		})).To(Succeed())
		var res task.Task
		Expect(db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
			return svc.NewRetrieve().Where(task.MatchKeys(t.Key)).Entry(&res).
				Exec(ctx, snap)
		})).To(Succeed())
		Expect(res.Config).To(HaveKeyWithValue("routing_key", "rk-snapshot"))
	})

	Describe("Retrieve", func() {
		BeforeEach(openTx)
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

		It("Should resume strictly past the key", func(ctx SpecContext) {
			keys := make([]task.Key, 3)
			for i := range keys {
				t := &task.Task{Type: testType, Rack: testRack.Key, Name: "Paged"}
				Expect(w.Create(ctx, t)).To(Succeed())
				keys[i] = t.Key
			}
			// Each key's cursor drops it and the keys before it in storage order.
			counts := lo.Map(keys, func(k task.Key, _ int) int {
				var res []task.Task
				Expect(svc.NewRetrieve().
					Where(task.MatchKeys(keys...)).
					After(k).
					Entries(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).NotTo(ContainElement(HaveField("Key", k)))
				return len(res)
			})
			Expect(counts).To(ConsistOf(0, 1, 2))
		})

		It(
			"Should fail to filter by Arc on a service that cannot match Arcs",
			func(ctx SpecContext) {
				Expect(svc.NewRetrieve().
					Where(task.MatchArcs(testArc)).
					Entries(&[]task.Task{}).
					Exec(ctx, tx)).
					To(MatchError("this task service cannot filter tasks by Arc"))
			},
		)

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
		BeforeEach(openTx)
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
				Expect(stat.NewRetrieve[any]().
					Where(status.MatchOwners[any](m.ResourceID())).
					Entry(new(status.Status[any])).
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

	Describe("Rack", func() {
		BeforeEach(openTx)
		createRack := func(ctx context.Context) rack.Rack {
			GinkgoHelper()
			r := rack.Rack{Name: "Other Rack"}
			Expect(rackService.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			return r
		}
		exists := func(ctx context.Context, key task.Key) bool {
			GinkgoHelper()
			return MustSucceed(svc.NewRetrieve().
				Where(task.MatchKeys(key)).
				Exists(ctx, tx))
		}

		It("Should delete the internal tasks of a deleted rack", func(ctx SpecContext) {
			r := createRack(ctx)
			internal := &task.Task{
				Type:     testType,
				Rack:     r.Key,
				Name:     "Internal Scan Task",
				Internal: true,
			}
			elsewhere := &task.Task{Type: testType, Rack: testRack.Key, Name: "Kept"}
			Expect(w.Create(ctx, internal)).To(Succeed())
			Expect(w.Create(ctx, elsewhere)).To(Succeed())
			Expect(w.DeleteRacks(ctx, r.Key)).To(Succeed())
			Expect(exists(ctx, internal.Key)).To(BeFalse())
			Expect(exists(ctx, elsewhere.Key)).To(BeTrue())
		})

		It(
			"Should delete the config records of the tasks of a deleted rack",
			func(ctx SpecContext) {
				r := createRack(ctx)
				t := &task.Task{
					Type:     testType,
					Rack:     r.Key,
					Name:     "Internal Scan Task",
					Internal: true,
				}
				Expect(w.Create(ctx, t)).To(Succeed())
				store := MustBeOk(configs.Store(testType))
				MustSucceed(store.Read(ctx, tx, t.Key))
				Expect(w.DeleteRacks(ctx, r.Key)).To(Succeed())
				Expect(store.Read(ctx, tx, t.Key)).Error().
					To(MatchError(query.ErrNotFound))
			},
		)

		It(
			"Should refuse to delete a rack that holds a task that is not internal",
			func(ctx SpecContext) {
				r := createRack(ctx)
				t := &task.Task{Type: testType, Rack: r.Key, Name: "User Task"}
				Expect(w.Create(ctx, t)).To(Succeed())
				Expect(w.DeleteRacks(ctx, r.Key)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						"cannot delete rack Other Rack because it still holds tasks",
					)),
				))
				Expect(exists(ctx, t.Key)).To(BeTrue())
				Expect(rackService.NewRetrieve().
					Where(rack.MatchKeys(r.Key)).
					Exists(ctx, tx)).To(BeTrue())
			},
		)

		It("Should refuse a task on a rack that does not exist", func(ctx SpecContext) {
			t := &task.Task{Type: testType, Rack: rack.NewKey(1, 999), Name: "Lost"}
			Expect(w.Create(ctx, t)).To(MatchError(fmt.Sprintf(
				"rack: rack %d does not exist: validation error",
				rack.NewKey(1, 999),
			)))
		})

		It("Should store a draft on rack zero", func(ctx SpecContext) {
			t := &task.Task{Type: testType, Name: "Draft"}
			Expect(w.Create(ctx, t)).To(Succeed())
			Expect(exists(ctx, t.Key)).To(BeTrue())
		})

		It("Should find the drafts with rack zero", func(ctx SpecContext) {
			r := createRack(ctx)
			draft := &task.Task{Type: testType, Name: "Draft"}
			onRack := &task.Task{Type: testType, Rack: r.Key, Name: "On Rack"}
			Expect(w.Create(ctx, draft)).To(Succeed())
			Expect(w.Create(ctx, onRack)).To(Succeed())
			keysOn := func(racks ...rack.Key) []task.Key {
				GinkgoHelper()
				var res []task.Task
				Expect(svc.NewRetrieve().
					Where(task.MatchRacks(racks...)).
					Entries(&res).
					Exec(ctx, tx)).To(Succeed())
				return lo.Map(res, func(t task.Task, _ int) task.Key { return t.Key })
			}
			Expect(keysOn(0)).To(And(
				ContainElement(draft.Key),
				Not(ContainElement(onRack.Key)),
			))
			Expect(keysOn(0, r.Key)).To(ContainElements(draft.Key, onRack.Key))
		})

		It("Should find the tasks on a rack", func(ctx SpecContext) {
			r := createRack(ctx)
			t := &task.Task{Type: testType, Rack: r.Key, Name: "On Rack"}
			Expect(w.Create(ctx, t)).To(Succeed())
			var res []task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchRacks(r.Key)).
				Entries(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(HaveLen(1))
			Expect(res[0].Key).To(Equal(t.Key))
		})
	})

	Describe("Status", func() {
		BeforeEach(openTx)
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
				Expect(stat.NewRetrieve[task.StatusDetails]().
					Where(status.MatchOwners[task.StatusDetails](m.ResourceID())).
					Entry(&taskStatus).
					Exec(ctx, tx)).To(Succeed())
				Expect(taskStatus.Variant).To(Equal(status.VariantDisabled))
				Expect(
					taskStatus.Message,
				).To(Equal("Task not deployed"))
				Expect(taskStatus.Owner).To(HaveValue(Equal(m.ResourceID())))
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
				Expect(stat.NewRetrieve[task.StatusDetails]().
					Where(status.MatchOwners[task.StatusDetails](m.ResourceID())).
					Entry(&taskStatus).
					Exec(ctx, tx)).To(Succeed())
				Expect(taskStatus.Variant).To(Equal(status.VariantSuccess))
				Expect(taskStatus.Message).To(Equal("Custom task status"))
				Expect(taskStatus.Description).To(Equal("Task is running"))
				Expect(taskStatus.Owner).To(HaveValue(Equal(m.ResourceID())))
				// Name should be auto-filled
				Expect(taskStatus.Name).To(Equal(m.Name))
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

				sw := stat.NewWriter(tx)
				Expect(sw.DeleteOwned(ctx, t.ResourceID())).
					To(Succeed())
				Expect(stat.NewRetrieve[any]().
					Where(status.MatchOwners[any](t.ResourceID())).
					Entry(new(status.Status[any])).
					Exec(ctx, tx)).To(MatchError(query.ErrNotFound))

				reconfigured := &task.Task{Key: t.Key, Name: t.Name, Type: testType}
				Expect(w.Create(ctx, reconfigured)).To(Succeed())

				var healed task.Status
				Expect(stat.NewRetrieve[task.StatusDetails]().
					Where(status.MatchOwners[task.StatusDetails](t.ResourceID())).
					Entry(&healed).
					Exec(ctx, tx)).To(Succeed())
				Expect(healed.Owner).To(HaveValue(Equal(t.ResourceID())))
				Expect(healed.Variant).To(Equal(status.VariantWarning))
				Expect(healed.Message).To(Equal("Task status unknown"))
			},
		)

		It(
			"Should not write a provided status over the status of another task",
			func(ctx SpecContext) {
				a := &task.Task{Type: testType, Rack: testRack.Key, Name: "A"}
				Expect(w.Create(ctx, a)).To(Succeed())
				b := &task.Task{
					Type: testType,
					Rack: testRack.Key,
					Name: "B",
					Status: &task.Status{
						Key:     a.Status.Key,
						Variant: status.VariantSuccess,
						Message: "forged",
						Time:    telem.Now(),
					},
				}
				Expect(w.Create(ctx, b)).To(Succeed())
				var aStatus, bStatus task.Status
				Expect(stat.NewRetrieve[task.StatusDetails]().
					Where(status.MatchOwners[task.StatusDetails](a.ResourceID())).
					Entry(&aStatus).
					Exec(ctx, tx)).To(Succeed())
				Expect(aStatus.Message).To(Equal("Task not deployed"))
				Expect(stat.NewRetrieve[task.StatusDetails]().
					Where(status.MatchOwners[task.StatusDetails](b.ResourceID())).
					Entry(&bStatus).
					Exec(ctx, tx)).To(Succeed())
				Expect(bStatus.Message).To(Equal("forged"))
				Expect(bStatus.Key).ToNot(Equal(aStatus.Key))
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
				Expect(stat.NewRetrieve[task.StatusDetails]().
					Where(status.MatchOwners[task.StatusDetails](t.ResourceID())).
					Entry(&preserved).
					Exec(ctx, tx)).To(Succeed())
				Expect(preserved.Variant).To(Equal(status.VariantSuccess))
				Expect(preserved.Message).To(Equal("Task is running"))
			},
		)

		It(
			"Should rename a live status row when a re-configure renames the task",
			func(ctx SpecContext) {
				t := &task.Task{
					Type: testType,
					Rack: testRack.Key,
					Name: "Before Task",
					Status: &task.Status{
						Variant: status.VariantSuccess,
						Message: "Task is running",
						Time:    telem.Now(),
					},
				}
				Expect(w.Create(ctx, t)).To(Succeed())

				reconfigured := &task.Task{
					Key:  t.Key,
					Name: "After Task",
					Type: testType,
				}
				Expect(w.Create(ctx, reconfigured)).To(Succeed())
				Expect(reconfigured.Status.Name).To(Equal("After Task"))

				var renamed task.Status
				Expect(stat.NewRetrieve[task.StatusDetails]().
					Where(status.MatchOwners[task.StatusDetails](t.ResourceID())).
					Entry(&renamed).
					Exec(ctx, tx)).To(Succeed())
				Expect(renamed.Name).To(Equal("After Task"))
				Expect(renamed.Message).To(Equal("Task is running"))
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
				).To(Equal("Task not deployed"))
				Expect(m.Status.Owner).To(HaveValue(Equal(m.ResourceID())))
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

				copied := MustSucceed(w.Copy(ctx, m.Key, "Copied Task"))

				var copiedStatus task.Status
				Expect(stat.NewRetrieve[task.StatusDetails]().
					Where(status.MatchOwners[task.StatusDetails](copied.ResourceID())).
					Entry(&copiedStatus).
					Exec(ctx, tx)).To(Succeed())
				Expect(copiedStatus.Variant).To(Equal(status.VariantDisabled))
				Expect(
					copiedStatus.Message,
				).To(Equal("Task not deployed"))
				Expect(copiedStatus.Owner).To(HaveValue(Equal(copied.ResourceID())))
			},
		)
	})

	Describe("Suspect Rack", func() {
		retrieveStatus := func(ctx context.Context, t *task.Task) task.Status {
			GinkgoHelper()
			var got task.Status
			Expect(stat.NewRetrieve[task.StatusDetails]().
				Where(status.MatchOwners[task.StatusDetails](t.ResourceID())).
				Entry(&got).
				Exec(ctx, db)).To(Succeed())
			return got
		}

		It(
			"Should copy the rack's problem onto a running task",
			func(ctx SpecContext) {
				// The rack and its tasks commit together, so the rack cannot go silent
				// before the tasks exist.
				createTx := db.OpenTx()
				defer func() { Expect(createTx.Close()).To(Succeed()) }()
				r := rack.Rack{Name: "suspect rack"}
				Expect(rackService.NewWriter(createTx).Create(ctx, &r)).To(Succeed())
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
						},
					},
				}
				Expect(svc.NewWriter(createTx).Create(ctx, t)).To(Succeed())
				Expect(createTx.Commit(ctx)).To(Succeed())

				Eventually(func(g Gomega) {
					var taskStatus task.Status
					g.Expect(stat.NewRetrieve[task.StatusDetails]().
						Where(status.MatchOwners[task.StatusDetails](t.ResourceID())).
						Entry(&taskStatus).
						Exec(ctx, db)).To(Succeed())
					g.Expect(taskStatus.Message).To(Equal("Task stopped responding"))
					g.Expect(taskStatus.Variant).To(Equal(status.VariantWarning))
					g.Expect(taskStatus.Name).To(Equal("Test Task"))
					g.Expect(taskStatus.Description).To(Equal(
						"The Core lost contact with the Driver on suspect rack " +
							"while the task was running.",
					))
					g.Expect(taskStatus.Owner).To(HaveValue(Equal(t.ResourceID())))
					g.Expect(taskStatus.Details).To(Equal(task.StatusDetails{
						Running:    false,
						ConfigHash: "deployed",
						Rack:       r.Key,
					}))
				}).Should(Succeed())
			},
		)

		It(
			"Should not copy the rack's problem onto a task that is not running",
			func(ctx SpecContext) {
				// The rack and its tasks commit together, so the rack cannot go silent
				// before the tasks exist.
				createTx := db.OpenTx()
				defer func() { Expect(createTx.Close()).To(Succeed()) }()
				r := rack.Rack{Name: "suspect rack"}
				Expect(rackService.NewWriter(createTx).Create(ctx, &r)).To(Succeed())
				running := &task.Task{
					Type: testType,
					Rack: r.Key,
					Name: "Running Task",
					Status: &task.Status{
						Variant: status.VariantSuccess,
						Message: "Task is running",
						Time:    telem.Now(),
						Details: task.StatusDetails{Running: true},
					},
				}
				Expect(svc.NewWriter(createTx).Create(ctx, running)).To(Succeed())
				stopped := &task.Task{
					Type: testType,
					Rack: r.Key,
					Name: "Stopped Task",
					Status: &task.Status{
						Variant: status.VariantSuccess,
						Message: "Task stopped",
						Time:    telem.Now(),
					},
				}
				Expect(svc.NewWriter(createTx).Create(ctx, stopped)).To(Succeed())
				undeployed := &task.Task{Type: testType, Rack: r.Key, Name: "New Task"}
				Expect(svc.NewWriter(createTx).Create(ctx, undeployed)).To(Succeed())
				Expect(createTx.Commit(ctx)).To(Succeed())

				By("Waiting for the copy onto the running task, written in one batch")
				Eventually(func(g Gomega) {
					var taskStatus task.Status
					g.Expect(stat.NewRetrieve[task.StatusDetails]().
						Where(status.MatchOwners[task.StatusDetails](
							running.ResourceID(),
						)).
						Entry(&taskStatus).
						Exec(ctx, db)).To(Succeed())
					g.Expect(taskStatus.Message).To(Equal("Task stopped responding"))
				}).Should(Succeed())

				stoppedStatus := retrieveStatus(ctx, stopped)
				Expect(stoppedStatus.Variant).To(Equal(status.VariantSuccess))
				Expect(stoppedStatus.Message).To(Equal("Task stopped"))
				undeployedStatus := retrieveStatus(ctx, undeployed)
				Expect(undeployedStatus.Variant).To(Equal(status.VariantDisabled))
				Expect(undeployedStatus.Message).To(Equal("Task not deployed"))
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
		It("Should echo the task's config hash and rack", func() {
			t := task.Task{
				Key:        uuid.New(),
				Rack:       testRack.Key,
				ConfigHash: "hash1",
			}
			Expect(task.NewStatusDetails(t, true)).To(Equal(task.StatusDetails{
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

	Describe("Search", func() {
		BeforeEach(openTx)
		create := func(ctx SpecContext, name string) resource.ID {
			GinkgoHelper()
			t := task.Task{Type: testType, Rack: testRack.Key, Name: name}
			Expect(w.Create(ctx, &t)).To(Succeed())
			return task.ResourceID(t.Key)
		}

		It(
			"Should return the matching tasks, closest match first",
			func(ctx SpecContext) {
				a := create(ctx, "turbopump test")
				b := create(ctx, "turbopump")
				q := search.Query[resource.ID]{Term: "turbopump"}
				Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
					To(Equal([]resource.ID{b, a}))
				q.Limit = 1
				Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
					To(Equal([]resource.ID{b}))
			},
		)

		It("Should match a task by its type", func(ctx SpecContext) {
			id := create(ctx, "igniter")
			q := search.Query[resource.ID]{Term: testType}
			Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
				To(ContainElement(id))
		})

		It(
			"Should not match a staged task outside its transaction",
			func(ctx SpecContext) {
				create(ctx, "injector")
				q := search.Query[resource.ID]{Term: "injector"}
				Expect(
					MustSucceed(svc.Resources().Search(ctx, db, q)).Keys,
				).To(BeEmpty())
			},
		)
	})
})

var _ = Describe("MatchArcs", func() {
	It(
		"Should filter tasks by Arc through the configured matcher",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			pd := MustOpen(pagerduty.OpenService(ctx, pagerduty.ServiceConfig{DB: db}))
			var (
				want  task.Key
				asked []arc.Key
			)
			tasks := MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
				DB:     db,
				Stores: pd.Stores(),
				MatchArcs: func(arcs ...arc.Key) task.Filter {
					asked = append(asked, arcs...)
					return task.MatchKeys(want)
				},
			}))
			create := func() task.Key {
				GinkgoHelper()
				return MustSucceed(tasks.Create(ctx, pd.Alert, msgpack.EncodedJSON{
					"routing_key": "rk",
				}))
			}
			want = create()
			create()
			arcKey := uuid.New()
			var res []task.Task
			Expect(tasks.NewRetrieve().
				Where(task.MatchArcs(arcKey)).
				Entries(&res).
				Exec(ctx, db)).To(Succeed())
			Expect(lo.Map(res, func(t task.Task, _ int) task.Key { return t.Key })).
				To(Equal([]task.Key{want}))
			Expect(asked).To(Equal([]arc.Key{arcKey}))
		},
	)
})

var _ = Describe("Writer in concurrent transactions", func() {
	var (
		db       *gorp.DB
		svc      *task.Service
		testRack rack.Rack
		hashes   map[string]string
	)
	routingKey := func(i int) string { return fmt.Sprintf("rk-%d", i) }
	newTask := func(key task.Key, i int) task.Task {
		return task.Task{
			Key:    key,
			Type:   testType,
			Rack:   testRack.Key,
			Name:   "Concurrent",
			Config: msgpack.EncodedJSON{"routing_key": routingKey(i)},
		}
	}
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB: db,
		}))
		stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB: db,
		}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
			DB: db,
		}))
		racks := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:         db,
			HostKey:    1,
			Status:     stat,
			DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roleSvc)),
		}))
		pd := MustOpen(pagerduty.OpenService(ctx, pagerduty.ServiceConfig{DB: db}))
		svc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
			DB:      db,
			Rack:    racks,
			Status:  stat,
			ImEx:    imex.NewService(),
			Configs: MustSucceed(config.NewRegistry(pd.Stores()...)),
		}))
		testRack = rack.Rack{Name: "Test Rack"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return racks.NewWriter(tx).Create(ctx, &testRack)
		})).To(Succeed())
		hashes = make(map[string]string)
		for i := range 4 {
			t := newTask(uuid.New(), i)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &t)
			})).To(Succeed())
			hashes[routingKey(i)] = t.ConfigHash
		}
	})
	expectStoredHash := func(ctx context.Context, key task.Key) {
		GinkgoHelper()
		var t task.Task
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewRetrieve().Where(task.MatchKeys(key)).Entry(&t).Exec(ctx, tx)
		})).To(Succeed())
		Expect(t.ConfigHash).To(Equal(hashes[t.Config["routing_key"].(string)]))
	}
	createUntil := func(ctx context.Context, done *atomic.Bool, key task.Key) func() {
		return func() {
			defer GinkgoRecover()
			for i := 0; !done.Load(); i++ {
				t := newTask(key, i%len(hashes))
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &t)
				})).To(Succeed())
			}
		}
	}

	It(
		"Should store the hash of the stored config during concurrent creates",
		func(ctx SpecContext) {
			t := newTask(uuid.New(), 0)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &t)
			})).To(Succeed())
			var (
				wg   sync.WaitGroup
				done atomic.Bool
			)
			for range 4 {
				wg.Go(createUntil(ctx, &done, t.Key))
			}
			wg.Go(func() {
				defer GinkgoRecover()
				defer done.Store(true)
				for range 200 {
					expectStoredHash(ctx, t.Key)
				}
			})
			wg.Wait()
			expectStoredHash(ctx, t.Key)
		},
	)

	It(
		"Should copy the config that matches the copied hash during concurrent creates",
		func(ctx SpecContext) {
			src := newTask(uuid.New(), 0)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &src)
			})).To(Succeed())
			var (
				wg   sync.WaitGroup
				done atomic.Bool
			)
			for range 2 {
				wg.Go(createUntil(ctx, &done, src.Key))
			}
			wg.Go(func() {
				defer GinkgoRecover()
				defer done.Store(true)
				for range 50 {
					var c task.Task
					Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
						c, err = svc.NewWriter(tx).
							Copy(ctx, src.Key, "Copy")
						return err
					})).To(Succeed())
					expectStoredHash(ctx, c.Key)
				}
			})
			wg.Wait()
		},
	)
})
