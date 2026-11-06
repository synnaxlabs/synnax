// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"encoding/json/v2"
	"fmt"
	"math"
	"uuid"

	"github.com/cespare/xxhash/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	racktask "github.com/synnaxlabs/synnax/pkg/service/rack/task"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	v2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("QuarantineKVKey", func() {
	It("Should scope the task key under the quarantine prefix", func() {
		key := uuid.New()
		Expect(string(v3.QuarantineKVKey(key))).To(
			Equal(v3.QuarantineKVPrefix + key.String()),
		)
	})
})

var _ = Describe("NewMigration", func() {
	var (
		memDB     kv.DB
		db        *gorp.DB
		otg       *ontology.Ontology
		statusSvc *status.Service
	)
	BeforeEach(func(ctx SpecContext) {
		memDB = memkv.New()
		db = DeferClose(gorp.Wrap(memDB))
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
		}))
		statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
	})

	seed := func(ctx SpecContext, t v2.Task) {
		GinkgoHelper()
		Expect(gorp.NewCreate[v2.Key, v2.Task]().Entry(&t).Exec(ctx, db)).
			To(Succeed())
		Expect(otg.NewWriter(nil).DefineResources(ctx, resource.ID{
			Type: resource.TypeTask,
			Key:  t.Key.String(),
		})).To(Succeed())
	}

	// run migrates the stored tasks as the task table does when it opens.
	run := func(ctx SpecContext, stores ...config.Store) {
		GinkgoHelper()
		configs := MustSucceed(config.NewRegistry(stores...))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v2.Key, v2.Task]{
			DB:   db,
			Name: string(resource.TypeTask),
			Migrations: []gorp.Migration{
				v3.NewMigration(v3.MigrationConfig{
					Status:   statusSvc,
					Ontology: otg,
					Configs:  configs,
				}),
			},
		}))
	}

	// openArcStore opens the Arc table and the Arc task config store, in the order the
	// Core opens them.
	openArcStore := func(ctx SpecContext) config.Store {
		GinkgoHelper()
		MustOpen(arc.OpenTable(ctx, arc.TableConfig{DB: db}))
		return MustOpen(arctask.OpenService(ctx, arctask.ServiceConfig{DB: db})).Config
	}

	createArc := func(ctx SpecContext) arc.Key {
		GinkgoHelper()
		a := arc.Arc{Key: uuid.New(), Name: "Stored Arc"}
		Expect(gorp.NewCreate[arc.Key, arc.Arc]().Entry(&a).Exec(ctx, db)).
			To(Succeed())
		return a.Key
	}

	It(
		"Should normalize the config into its record store under the task's key",
		func(ctx SpecContext) {
			store := openArcStore(ctx)
			key, arcKey := uuid.New(), createArc(ctx)
			seed(ctx, v2.Task{
				Key:  key,
				Name: "Stored Arc Task",
				Type: "arc",
				Config: msgpack.EncodedJSON{
					"arcKey": arcKey.String(),
					"hash":   "abc123",
				},
			})
			run(ctx, store)

			var migrated v2.Task
			Expect(gorp.NewRetrieve[v2.Key, v2.Task]().
				Where(gorp.MatchKeys[v2.Key, v2.Task](key)).
				Entry(&migrated).Exec(ctx, db)).To(Succeed())
			Expect(migrated.Type).To(Equal("arc"))
			Expect(migrated.Config).To(BeEmpty())

			record := MustSucceed(store.Read(ctx, nil, key))
			Expect(record).To(HaveKeyWithValue("arc_key", arcKey.String()))
			Expect(record).To(HaveKeyWithValue("hash", "abc123"))

			// The hash must follow the frozen rule — xxhash64 of the JSON encoding of
			// the canonical record without its key — or drivers see phantom config
			// drift after the upgrade. Deterministic holds the member order the frozen
			// rule was defined against.
			content := make(map[string]any, len(record))
			for k, v := range record {
				if k != "key" {
					content[k] = v
				}
			}
			b := MustSucceed(json.Marshal(content, json.Deterministic(true)))
			Expect(migrated.ConfigHash).To(
				Equal(fmt.Sprintf("%016x", xxhash.Sum64(b))),
			)
		},
	)

	It(
		"Should rename a released driver's rack status task onto its store",
		func(ctx SpecContext) {
			key := uuid.New()
			seed(ctx, v2.Task{
				Key:      key,
				Name:     "Rack Status",
				Type:     "Rack Status",
				Internal: true,
			})
			rt := MustOpen(racktask.OpenService(ctx, racktask.ServiceConfig{DB: db}))
			run(ctx, rt.Stores()...)

			var migrated v2.Task
			Expect(gorp.NewRetrieve[v2.Key, v2.Task]().
				Where(gorp.MatchKeys[v2.Key, v2.Task](key)).
				Entry(&migrated).Exec(ctx, db)).To(Succeed())
			Expect(migrated.Type).To(Equal("rack_status"))
			Expect(migrated.ConfigHash).To(HaveLen(16))
			record := MustSucceed(rt.Status.Read(ctx, nil, key))
			Expect(record).To(HaveKeyWithValue("key", key.String()))
		},
	)

	It(
		"Should retire the stale Rack State twin a 0.49 driver upgrade left behind",
		func(ctx SpecContext) {
			stale, live := uuid.New(), uuid.New()
			seed(ctx, v2.Task{
				Key:      stale,
				Name:     "Rack State",
				Type:     "Rack State",
				Internal: true,
			})
			seed(ctx, v2.Task{
				Key:      live,
				Name:     "Rack Status",
				Type:     "Rack Status",
				Internal: true,
			})
			rt := MustOpen(racktask.OpenService(ctx, racktask.ServiceConfig{DB: db}))
			run(ctx, rt.Stores()...)

			Expect(gorp.NewRetrieve[v2.Key, v2.Task]().
				Where(gorp.MatchKeys[v2.Key, v2.Task](stale)).
				Exists(ctx, db)).To(BeFalse())
			var migrated v2.Task
			Expect(gorp.NewRetrieve[v2.Key, v2.Task]().
				Where(gorp.MatchKeys[v2.Key, v2.Task](live)).
				Entry(&migrated).Exec(ctx, db)).To(Succeed())
			Expect(migrated.Type).To(Equal("rack_status"))
		},
	)

	It("Should store a record for every task it converts", func(ctx SpecContext) {
		store := openArcStore(ctx)
		rt := MustOpen(racktask.OpenService(ctx, racktask.ServiceConfig{DB: db}))
		arcKeys := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
		for i, k := range arcKeys {
			seed(ctx, v2.Task{
				Key:    k,
				Name:   fmt.Sprintf("Arc Task %d", i),
				Type:   "arc",
				Config: msgpack.EncodedJSON{"arcKey": createArc(ctx).String()},
			})
		}
		statusKey := uuid.New()
		seed(ctx, v2.Task{
			Key:      statusKey,
			Name:     "Rack Status",
			Type:     "Rack Status",
			Internal: true,
		})
		run(ctx, append(rt.Stores(), store)...)

		var migrated []v2.Task
		Expect(gorp.NewRetrieve[v2.Key, v2.Task]().
			Entries(&migrated).Exec(ctx, db)).To(Succeed())
		Expect(migrated).To(HaveLen(4))
		for _, k := range arcKeys {
			Expect(store.Read(ctx, nil, k)).To(HaveKeyWithValue("key", k.String()))
		}
		Expect(rt.Status.Read(ctx, nil, statusKey)).
			To(HaveKeyWithValue("key", statusKey.String()))
	})

	It("Should quarantine a task of an unknown type", func(ctx SpecContext) {
		key := uuid.New()
		seed(ctx, v2.Task{
			Key:    key,
			Name:   "Bogus Task",
			Type:   "bogus",
			Config: msgpack.EncodedJSON{"anything": true},
		})
		taskID := resource.ID{Type: resource.TypeTask, Key: key.String()}
		Expect(statusSvc.NewMigrationWriter(nil).Set(ctx, status.Status[any]{
			Key:     uuid.New().String(),
			Owner:   &taskID,
			Name:    "Bogus Task",
			Variant: status.VariantSuccess,
			Time:    telem.Now(),
		})).To(Succeed())
		run(ctx, openArcStore(ctx))

		Expect(statusSvc.NewRetrieve[any]().
			Where(status.MatchOwners[any](taskID)).
			Entry(new(status.Status[any])).
			Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
		Expect(gorp.NewRetrieve[v2.Key, v2.Task]().
			Where(gorp.MatchKeys[v2.Key, v2.Task](key)).
			Exists(ctx, db)).To(BeFalse())
		Expect(otg.NewRetrieve().WhereIDs(resource.ID{
			Type: resource.TypeTask,
			Key:  key.String(),
		}).Exists(ctx, nil)).To(BeFalse())
		staged, closer := MustSucceed2(db.Get(ctx, v3.QuarantineKVKey(key)))
		Expect(string(staged)).To(ContainSubstring("Bogus Task"))
		Expect(closer.Close()).To(Succeed())
	})

	It("Should quarantine a task whose Arc does not exist", func(ctx SpecContext) {
		store := openArcStore(ctx)
		key, missing := uuid.New(), uuid.New()
		seed(ctx, v2.Task{
			Key:    key,
			Name:   "Orphaned Arc Task",
			Type:   "arc",
			Config: msgpack.EncodedJSON{"arcKey": missing.String()},
		})
		run(ctx, store)

		Expect(gorp.NewRetrieve[v2.Key, v2.Task]().
			Where(gorp.MatchKeys[v2.Key, v2.Task](key)).
			Exists(ctx, db)).To(BeFalse())
		Expect(store.Read(ctx, nil, key)).Error().To(MatchError(query.ErrNotFound))
		staged, closer := MustSucceed2(db.Get(ctx, v3.QuarantineKVKey(key)))
		Expect(string(staged)).To(ContainSubstring("Orphaned Arc Task"))
		Expect(closer.Close()).To(Succeed())
	})

	It(
		"Should quarantine a task whose config fails to decode",
		func(ctx SpecContext) {
			key := uuid.New()
			seed(ctx, v2.Task{
				Key:    key,
				Name:   "Broken Arc Task",
				Type:   "arc",
				Config: msgpack.EncodedJSON{"arc_key": 123},
			})
			run(ctx, openArcStore(ctx))

			Expect(gorp.NewRetrieve[v2.Key, v2.Task]().
				Where(gorp.MatchKeys[v2.Key, v2.Task](key)).
				Exists(ctx, db)).To(BeFalse())
			staged, closer := MustSucceed2(db.Get(ctx, v3.QuarantineKVKey(key)))
			Expect(string(staged)).To(ContainSubstring("Broken Arc Task"))
			Expect(closer.Close()).To(Succeed())
		},
	)

	It(
		"Should stage a config JSON cannot encode without aborting boot",
		func(ctx SpecContext) {
			key := uuid.New()
			// Released cores encoded rows with msgpack, which accepts NaN; the
			// current orc codec rejects it, so the row seeds through a
			// msgpack-codec wrap of the same KV to mimic legacy on-disk data.
			legacyDB := gorp.Wrap(memDB, gorp.WithCodec(msgpack.Codec))
			t := v2.Task{
				Key:    key,
				Name:   "NaN Task",
				Type:   "bogus",
				Config: msgpack.EncodedJSON{"value": math.NaN()},
			}
			Expect(gorp.NewCreate[v2.Key, v2.Task]().
				Entry(&t).Exec(ctx, legacyDB)).To(Succeed())
			Expect(otg.NewWriter(nil).DefineResources(ctx, resource.ID{
				Type: resource.TypeTask,
				Key:  key.String(),
			})).To(Succeed())
			run(ctx, openArcStore(ctx))

			Expect(gorp.NewRetrieve[v2.Key, v2.Task]().
				Where(gorp.MatchKeys[v2.Key, v2.Task](key)).
				Exists(ctx, db)).To(BeFalse())
			staged, closer := MustSucceed2(db.Get(ctx, v3.QuarantineKVKey(key)))
			Expect(string(staged)).To(ContainSubstring("NaN Task"))
			// The config is dropped from the staged row: JSON cannot carry it.
			Expect(string(staged)).ToNot(ContainSubstring("value"))
			Expect(closer.Close()).To(Succeed())
		},
	)

	DescribeTable("Should remove tasks of retired types",
		func(ctx SpecContext, retiredType string) {
			key := uuid.New()
			seed(ctx, v2.Task{
				Key:    key,
				Name:   "Retired Task",
				Type:   retiredType,
				Config: msgpack.EncodedJSON{},
			})
			run(ctx, openArcStore(ctx))

			Expect(gorp.NewRetrieve[v2.Key, v2.Task]().
				Where(gorp.MatchKeys[v2.Key, v2.Task](key)).
				Exists(ctx, db)).To(BeFalse())
			staged, closer := MustSucceed2(db.Get(ctx, v3.QuarantineKVKey(key)))
			Expect(string(staged)).To(ContainSubstring("Retired Task"))
			Expect(closer.Close()).To(Succeed())
		},
		Entry("sequence", "sequence"),
		Entry("heartbeat", "heartbeat"),
		Entry("opcScanner", "opcScanner"),
		Entry("Rack State", "Rack State"),
	)
})
