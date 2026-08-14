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
	"encoding/json"
	"fmt"
	"math"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	racktask "github.com/synnaxlabs/synnax/pkg/service/rack/task"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	v2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
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
		memDB kv.DB
		db    *gorp.DB
		otg   *ontology.Ontology
	)
	BeforeEach(func(ctx SpecContext) {
		memDB = memkv.New()
		db = DeferClose(gorp.Wrap(memDB))
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	})

	seed := func(ctx SpecContext, t v2.Task) {
		GinkgoHelper()
		Expect(gorp.NewCreate[v2.Key, v2.Task]().Entry(&t).Exec(ctx, db)).
			To(Succeed())
		Expect(otg.NewWriter(nil).DefineResources(ctx, ontology.ID{
			Type: ontology.ResourceTypeTask,
			Key:  t.Key.String(),
		})).To(Succeed())
	}

	run := func(ctx SpecContext, stores ...config.Store) {
		GinkgoHelper()
		configs := MustSucceed(config.NewRegistry(stores...))
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "Task",
			Migrations: []migrate.Migration{
				v3.NewMigration(v3.MigrationConfig{
					Ontology: otg,
					Configs:  configs,
				}),
			},
		})).To(Succeed())
	}

	openArcStore := func(ctx SpecContext) config.Store {
		GinkgoHelper()
		return MustOpen(config.OpenService(
			ctx,
			config.ServiceConfig[arctask.Config]{
				DB:                 db,
				Type:               "arc",
				Version:            1,
				SetEntryKey:        (*arctask.Config).SetKey,
				ApplyEntryDefaults: (*arctask.Config).ApplyDefaults,
				ValidateEntry:      (*arctask.Config).Validate,
			},
		))
	}

	It(
		"Should normalize the config into its record store under the task's key",
		func(ctx SpecContext) {
			store := openArcStore(ctx)
			key, arcKey := uuid.New(), uuid.New()
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

			// The hash must follow the frozen rule — xxhash64 of the JSON
			// encoding of the canonical record without its key — or drivers
			// see phantom config drift after the upgrade.
			content := make(map[string]any, len(record))
			for k, v := range record {
				if k != "key" {
					content[k] = v
				}
			}
			b := MustSucceed(json.Marshal(content))
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

	It("Should quarantine a task of an unknown type", func(ctx SpecContext) {
		key := uuid.New()
		seed(ctx, v2.Task{
			Key:    key,
			Name:   "Bogus Task",
			Type:   "bogus",
			Config: msgpack.EncodedJSON{"anything": true},
		})
		run(ctx, openArcStore(ctx))

		Expect(gorp.NewRetrieve[v2.Key, v2.Task]().
			Where(gorp.MatchKeys[v2.Key, v2.Task](key)).
			Exists(ctx, db)).To(BeFalse())
		Expect(otg.NewRetrieve().WhereIDs(ontology.ID{
			Type: ontology.ResourceTypeTask,
			Key:  key.String(),
		}).Exists(ctx, nil)).To(BeFalse())
		staged, closer := MustSucceed2(db.Get(ctx, v3.QuarantineKVKey(key)))
		Expect(string(staged)).To(ContainSubstring("Bogus Task"))
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
			Expect(otg.NewWriter(nil).DefineResources(ctx, ontology.ID{
				Type: ontology.ResourceTypeTask,
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
