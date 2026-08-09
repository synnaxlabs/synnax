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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/task/common"
	v2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
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
		db  *gorp.DB
		otg *ontology.Ontology
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
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

	run := func(ctx SpecContext, stores ...common.ConfigStore) {
		GinkgoHelper()
		configs := MustSucceed(common.NewConfigRegistry(stores...))
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

	openArcStore := func(ctx SpecContext) common.ConfigStore {
		GinkgoHelper()
		return MustOpen(common.OpenConfigService[arctask.Config](
			ctx,
			common.ConfigServiceConfig{
				DB:       db,
				Ontology: otg,
				Type:     "arc_task",
				Version:  1,
			},
		))
	}

	It(
		"Should normalize the config into its record store and parent the record",
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
			Expect(migrated.Type).To(Equal("arc_task"))
			Expect(migrated.Config).To(BeEmpty())
			Expect(migrated.ConfigHash).To(HaveLen(16))

			taskID := ontology.ID{
				Type: ontology.ResourceTypeTask,
				Key:  key.String(),
			}
			parents := MustSucceed(otg.RetrieveParents(nil, taskID))
			ids := parents[taskID]
			Expect(ids).To(HaveLen(1))
			Expect(ids[0].Type).To(Equal(ontology.ResourceType("arc_task")))
			record := MustSucceed(store.Read(
				ctx, nil, MustSucceed(uuid.Parse(ids[0].Key)),
			))
			Expect(record).To(HaveKeyWithValue("arc_key", arcKey.String()))
			Expect(record).To(HaveKeyWithValue("hash", "abc123"))
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
				Type:   "arc_task",
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
	)
})
