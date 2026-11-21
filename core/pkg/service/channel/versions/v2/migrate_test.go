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
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	v1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v2"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("Migration", func() {
	It("Should keep its key, which the applied set holds forever", func() {
		Expect(v2.Migration.Key()).To(Equal("v63_drop_internal"))
	})

	It("Should drop the internal flag and log each channel that held it", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		stored := []v1.Channel{
			{
				Name:        "sy_task_cmd",
				Leaseholder: node.KeyFree,
				LocalKey:    1,
				DataType:    telem.JSONT,
				Virtual:     true,
				Internal:    true,
			},
			{Name: "pressure", Leaseholder: 1, LocalKey: 2, DataType: telem.Float32T},
		}
		Expect(gorp.WrapWriter[v1.Key, v1.Channel](db).Set(ctx, stored...)).
			To(Succeed())
		core, logs := observer.New(zapcore.WarnLevel)
		logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
			DB:              db,
			Namespace:       "channel",
			Migrations:      []gorp.Migration{v2.Migration},
		})).To(Succeed())
		var migrated []v2.Channel
		Expect(gorp.NewRetrieve[v2.Key, v2.Channel]().
			Entries(&migrated).
			Exec(ctx, db)).To(Succeed())
		Expect(migrated).To(ConsistOf(
			v2.Channel{
				Name:        "sy_task_cmd",
				Leaseholder: node.KeyFree,
				LocalKey:    1,
				DataType:    telem.JSONT,
				Virtual:     true,
			},
			v2.Channel{
				Name:        "pressure",
				Leaseholder: 1,
				LocalKey:    2,
				DataType:    telem.Float32T,
			},
		))
		dropped := logs.FilterMessage("dropped the internal flag of a channel").All()
		Expect(dropped).To(HaveLen(1))
		Expect(dropped[0].ContextMap()).To(HaveKeyWithValue("name", "sy_task_cmd"))
		Expect(dropped[0].ContextMap()).To(HaveKeyWithValue(
			"key",
			stored[0].Key().String(),
		))
	})
})

var _ = Describe("StageCalculations", func() {
	It("Should keep its key, which the applied set holds forever", func() {
		Expect(v2.StageCalculations.Key()).To(Equal("v63_stage_calculations"))
	})

	It("Should stage the expression and operations of each channel that has one", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		ops := []v1.Operation{{Type: v1.OperationTypeAvg, Duration: telem.Second}}
		stored := []v1.Channel{
			{
				Name:        "doubled",
				Leaseholder: node.KeyFree,
				LocalKey:    1,
				DataType:    telem.Float32T,
				Virtual:     true,
				Expression:  "return pressure * 2",
				Operations:  ops,
			},
			{Name: "pressure", Leaseholder: 1, LocalKey: 2, DataType: telem.Float32T},
		}
		Expect(gorp.WrapWriter[v1.Key, v1.Channel](db).Set(ctx, stored...)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: alamos.New("test"),
			DB:              db,
			Namespace:       "channel",
			Migrations: []gorp.Migration{
				v2.StageCalculations,
				v2.Migration,
			},
		})).To(Succeed())
		blob, closer, err := db.Get(
			[]byte(v2.CalculationKVPrefix + stored[0].Key().String()),
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(closer.Close()).To(Succeed())
		var staged v2.StagedCalculation
		Expect(json.Unmarshal(blob, &staged)).To(Succeed())
		Expect(staged).To(Equal(v2.StagedCalculation{
			Expression: "return pressure * 2",
			Operations: ops,
		}))
		_, _, err = db.Get([]byte(v2.CalculationKVPrefix + stored[1].Key().String()))
		Expect(err).To(MatchError(query.ErrNotFound))
	})
})
