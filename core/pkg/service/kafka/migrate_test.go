// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka_test

import (
	"context"
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	v1 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v1"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ICDMigration", func() {
	var (
		// names holds the task names that the lookup returns.
		names map[uuid.UUID]string
		// asked holds the keys of each lookup call.
		asked  [][]uuid.UUID
		lookup kafka.TaskNames
	)
	BeforeEach(func() {
		names, asked = map[uuid.UUID]string{}, nil
		lookup = func(
			_ context.Context,
			_ gorp.Tx,
			keys []uuid.UUID,
		) (map[uuid.UUID]string, error) {
			asked = append(asked, keys)
			return names, nil
		}
	})

	// stage stores a config that the task named name owns, with its records staged as
	// v1.ReadMigration stages them.
	stage := func(ctx SpecContext, name string) uuid.UUID {
		GinkgoHelper()
		key := uuid.New()
		Expect(gorp.WrapWriter[uuid.UUID, kafka.ReadConfig](db).Set(
			ctx,
			kafka.ReadConfig{Key: key, Topic: "plant", Records: []kafka.ReadRecord{}},
		)).To(Succeed())
		b := MustSucceed(json.Marshal(v1.InlineRead{Records: []v1.InlineRecord{{
			Key:       "r1",
			RecordKey: "pump",
			Fields: []v1.ReadField{{
				Key:      "level",
				Name:     "level",
				Pointer:  "/level",
				DataType: telem.Float64T,
			}},
		}}}))
		Expect(db.Set(ctx, v1.StagedRecordsKVKey(key), b)).To(Succeed())
		names[key] = name
		return key
	}
	run := func(ctx SpecContext) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return kafkaSvc.ICDMigration(lookup).Run(ctx, tx, alamos.Instrumentation{})
		})
	}
	// expectStaged asserts whether the records of the config are staged.
	expectStaged := func(key uuid.UUID, staged bool) {
		GinkgoHelper()
		if !staged {
			Expect(db.Get(v1.StagedRecordsKVKey(key))).Error().
				To(MatchError(query.ErrNotFound))
			return
		}
		_, closer := MustSucceed2(db.Get(v1.StagedRecordsKVKey(key)))
		Expect(closer.Close()).To(Succeed())
	}
	read := func(ctx SpecContext, key uuid.UUID) kafka.ReadConfig {
		GinkgoHelper()
		var cfg kafka.ReadConfig
		Expect(gorp.NewRetrieve[uuid.UUID, kafka.ReadConfig]().
			Where(gorp.MatchKeys[uuid.UUID, kafka.ReadConfig](key)).
			Entry(&cfg).
			Exec(ctx, db)).To(Succeed())
		return cfg
	}

	It("Should move the staged records of each config into an ICD named after its task",
		func(ctx SpecContext) {
			key := stage(ctx, "Pump station")
			Expect(run(ctx)).To(Succeed())
			Expect(asked).To(Equal([][]uuid.UUID{{key}}))
			expectStaged(key, false)
			cfg := read(ctx, key)
			Expect(cfg.Inline).To(BeNil())
			Expect(cfg.Records).To(HaveLen(1))
			Expect(cfg.Records[0].RecordKey).To(Equal("pump"))
			var ic icd.ICD
			Expect(icdSvc.NewRetrieve().
				Where(icd.MatchKeys(cfg.Records[0].ICD)).
				Entry(&ic).
				Exec(ctx, db)).To(Succeed())
			Expect(ic.Name).To(Equal("Pump station"))
			Expect(ic.Messages).To(HaveLen(1))
			Expect(ic.Messages[0].Name).To(Equal("pump"))
		},
	)

	It("Should do nothing without staged records", func(ctx SpecContext) {
		Expect(run(ctx)).To(Succeed())
		Expect(asked).To(BeEmpty())
	})

	It("Should delete the staged records of a config that no longer exists", func(
		ctx SpecContext,
	) {
		key := stage(ctx, "Snapshot")
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return kafkaSvc.Read.Delete(ctx, tx, key)
		})).To(Succeed())
		Expect(run(ctx)).To(Succeed())
		Expect(asked).To(BeEmpty())
		expectStaged(key, false)
	})

	It("Should keep the staged records when the task name lookup fails", func(
		ctx SpecContext,
	) {
		key := stage(ctx, "Pump station")
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.Delete(ctx, v1.StagedRecordsKVKey(key))).To(Succeed())
		})
		lookup = func(
			context.Context,
			gorp.Tx,
			[]uuid.UUID,
		) (map[uuid.UUID]string, error) {
			return nil, errors.New("no task")
		}
		Expect(run(ctx)).To(MatchError("tasks of Kafka read configs: no task"))
		expectStaged(key, true)
		Expect(read(ctx, key).Records).To(BeEmpty())
	})
})
