// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package config_test

import (
	"context"
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// stagedConfig is a config that the StagedMigration specs move staged values into.
type stagedConfig struct {
	Key   uuid.UUID
	Task  string
	Value string
}

func (c stagedConfig) GorpKey() uuid.UUID { return c.Key }

var _ = Describe("StagedMigration", func() {
	const prefix = "sy_config_test_staged/"
	var (
		sdb *gorp.DB
		// names holds the task names that lookup returns.
		names map[uuid.UUID]string
		// asked holds the keys of each lookup call.
		asked   [][]uuid.UUID
		lookup  config.TaskNames
		moveErr error
	)
	BeforeEach(func() {
		sdb = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(sdb.Run)
		names, asked, moveErr = map[uuid.UUID]string{}, nil, nil
		lookup = func(
			_ context.Context,
			_ gorp.Tx,
			keys []uuid.UUID,
		) (map[uuid.UUID]string, error) {
			asked = append(asked, keys)
			return names, nil
		}
	})

	stageValue := func(ctx SpecContext, key uuid.UUID, value string) {
		GinkgoHelper()
		b := MustSucceed(json.Marshal(value))
		Expect(sdb.Set(ctx, []byte(prefix+key.String()), b)).To(Succeed())
	}
	// stage stores a config that the task named task owns, and stages value for it.
	stage := func(ctx SpecContext, task, value string) uuid.UUID {
		GinkgoHelper()
		key := uuid.New()
		Expect(gorp.WrapWriter[uuid.UUID, stagedConfig](sdb).Set(
			ctx,
			stagedConfig{Key: key},
		)).To(Succeed())
		stageValue(ctx, key, value)
		names[key] = task
		return key
	}
	run := func(ctx SpecContext) error {
		return sdb.WithTx(ctx, func(tx gorp.Tx) error {
			return config.StagedMigration(
				"staged_test",
				prefix,
				lookup,
				func(
					_ context.Context,
					_ gorp.Tx,
					_ alamos.Instrumentation,
					task string,
					cfg *stagedConfig,
					staged string,
				) error {
					cfg.Task, cfg.Value = task, staged
					return moveErr
				},
			).Run(ctx, tx, alamos.Instrumentation{})
		})
	}
	isStaged := func(key uuid.UUID) bool {
		GinkgoHelper()
		_, closer, err := sdb.Get([]byte(prefix + key.String()))
		if errors.Is(err, query.ErrNotFound) {
			return false
		}
		Expect(err).ToNot(HaveOccurred())
		Expect(closer.Close()).To(Succeed())
		return true
	}
	read := func(ctx SpecContext, key uuid.UUID) stagedConfig {
		GinkgoHelper()
		var cfg stagedConfig
		Expect(gorp.NewRetrieve[uuid.UUID, stagedConfig]().
			Where(gorp.MatchKeys[uuid.UUID, stagedConfig](key)).
			Entry(&cfg).
			Exec(ctx, sdb)).To(Succeed())
		return cfg
	}

	It("Should move each staged value into its config and delete the value", func(
		ctx SpecContext,
	) {
		pump := stage(ctx, "Pump station", "pump")
		valve := stage(ctx, "Valve", "valve")
		Expect(run(ctx)).To(Succeed())
		Expect(asked).To(HaveLen(1))
		Expect(asked[0]).To(ConsistOf(pump, valve))
		Expect(read(ctx, pump)).To(Equal(stagedConfig{
			Key:   pump,
			Task:  "Pump station",
			Value: "pump",
		}))
		Expect(read(ctx, valve)).To(Equal(stagedConfig{
			Key:   valve,
			Task:  "Valve",
			Value: "valve",
		}))
		Expect(isStaged(pump)).To(BeFalse())
		Expect(isStaged(valve)).To(BeFalse())
	})

	It("Should do nothing without staged values", func(ctx SpecContext) {
		Expect(run(ctx)).To(Succeed())
		Expect(asked).To(BeEmpty())
	})

	It("Should delete the staged value of a config that no longer exists", func(
		ctx SpecContext,
	) {
		key := uuid.New()
		stageValue(ctx, key, "gone")
		Expect(run(ctx)).To(Succeed())
		Expect(asked).To(BeEmpty())
		Expect(isStaged(key)).To(BeFalse())
	})

	It("Should keep the staged values when the task name lookup fails", func(
		ctx SpecContext,
	) {
		key := stage(ctx, "Pump station", "pump")
		lookup = func(
			context.Context,
			gorp.Tx,
			[]uuid.UUID,
		) (map[uuid.UUID]string, error) {
			return nil, errors.New("no task")
		}
		Expect(run(ctx)).To(MatchError("task names: no task"))
		Expect(isStaged(key)).To(BeTrue())
		Expect(read(ctx, key)).To(Equal(stagedConfig{Key: key}))
	})

	It("Should keep the staged values when a move fails", func(ctx SpecContext) {
		key := stage(ctx, "Pump station", "pump")
		moveErr = errors.New("bad value")
		Expect(run(ctx)).To(MatchError("config " + key.String() + ": bad value"))
		Expect(isStaged(key)).To(BeTrue())
		Expect(read(ctx, key)).To(Equal(stagedConfig{Key: key}))
	})
})
