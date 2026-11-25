// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt_test

import (
	"context"
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	v3 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v3"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
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
		lookup mqtt.TaskNames
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

	// stage stores a config that the task named name owns, with its plain target staged
	// as v3.WriteMigration stages it.
	stage := func(ctx SpecContext, name string) uuid.UUID {
		GinkgoHelper()
		key := uuid.New()
		Expect(gorp.WrapWriter[uuid.UUID, mqtt.WriteConfig](db).Set(
			ctx,
			mqtt.WriteConfig{Key: key, Targets: []mqtt.WriteTarget{}},
		)).To(Succeed())
		b := MustSucceed(json.Marshal(v3.InlineWrite{
			Targets: []v3.InlineTarget{{
				Key:   "plant/valve/set",
				Topic: "plant/valve/set",
				Qos:   v3.QoSAtLeastOnce,
				Channel: v3.ChannelField{
					Pointer:  "/value",
					JSONType: payload.JSONTypeNumber,
					DataType: telem.Float64T,
				},
			}},
		}))
		Expect(db.Set(ctx, v3.StagedTargetsKVKey(key), b)).To(Succeed())
		names[key] = name
		return key
	}
	run := func(ctx SpecContext) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return mqttSvc.ICDMigration(lookup).Run(ctx, tx, alamos.Instrumentation{})
		})
	}
	// expectStaged asserts whether the plain targets of the config are staged.
	expectStaged := func(key uuid.UUID, staged bool) {
		GinkgoHelper()
		if !staged {
			Expect(db.Get(v3.StagedTargetsKVKey(key))).Error().
				To(MatchError(query.ErrNotFound))
			return
		}
		_, closer := MustSucceed2(db.Get(v3.StagedTargetsKVKey(key)))
		Expect(closer.Close()).To(Succeed())
	}
	read := func(ctx SpecContext, key uuid.UUID) mqtt.WriteConfig {
		GinkgoHelper()
		var cfg mqtt.WriteConfig
		Expect(gorp.NewRetrieve[uuid.UUID, mqtt.WriteConfig]().
			Where(gorp.MatchKeys[uuid.UUID, mqtt.WriteConfig](key)).
			Entry(&cfg).
			Exec(ctx, db)).To(Succeed())
		return cfg
	}

	It("Should move the staged targets of each config into an ICD named after its task",
		func(ctx SpecContext) {
			key := stage(ctx, "Valves")
			Expect(run(ctx)).To(Succeed())
			Expect(asked).To(Equal([][]uuid.UUID{{key}}))
			expectStaged(key, false)
			cfg := read(ctx, key)
			Expect(cfg.Inline).To(BeNil())
			Expect(cfg.Targets).To(HaveLen(1))
			plain, ok := cfg.Targets[0].Variant.(mqtt.PlainWriteTarget)
			Expect(ok).To(BeTrue())
			Expect(plain.Topic).To(Equal("plant/valve/set"))
			var ic icd.ICD
			Expect(icdSvc.NewRetrieve().
				Where(icd.MatchKeys(plain.ICD)).
				Entry(&ic).
				Exec(ctx, db)).To(Succeed())
			Expect(ic.Name).To(Equal("Valves"))
			Expect(ic.Messages).To(HaveLen(1))
			Expect(ic.Messages[0].Name).To(Equal("plant/valve/set"))
		},
	)

	It("Should do nothing without staged targets", func(ctx SpecContext) {
		Expect(run(ctx)).To(Succeed())
		Expect(asked).To(BeEmpty())
	})

	It("Should delete the staged targets of a config that no longer exists", func(
		ctx SpecContext,
	) {
		key := stage(ctx, "Snapshot")
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return mqttSvc.Write.Delete(ctx, tx, key)
		})).To(Succeed())
		Expect(run(ctx)).To(Succeed())
		Expect(asked).To(BeEmpty())
		expectStaged(key, false)
	})

	It("Should keep the staged targets when the task name lookup fails", func(
		ctx SpecContext,
	) {
		key := stage(ctx, "Valves")
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.Delete(ctx, v3.StagedTargetsKVKey(key))).To(Succeed())
		})
		lookup = func(
			context.Context,
			gorp.Tx,
			[]uuid.UUID,
		) (map[uuid.UUID]string, error) {
			return nil, errors.New("no task")
		}
		Expect(run(ctx)).To(MatchError("tasks of MQTT write configs: no task"))
		expectStaged(key, true)
		Expect(read(ctx, key).Targets).To(BeEmpty())
	})
})
