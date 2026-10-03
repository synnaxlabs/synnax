// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v3 "github.com/synnaxlabs/synnax/pkg/service/arc/task/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/task/versions/v4"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migration", func() {
	DescribeTable("Should convert the loop mode to a performance level",
		func(ctx SpecContext, mode v3.ExecutionMode, want v4.Performance) {
			seed := v3.Config{
				ArcKey:        uuid.New(),
				Hash:          "abc123",
				ExecutionMode: mode,
				RtPriority:    80,
				CPUAffinity:   3,
				MemoryLocked:  true,
			}
			seed.Key = uuid.New()
			seed.AutoStart = true
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustSucceed(
				gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v3.Config]{DB: db}),
			)
			Expect(gorp.NewCreate[uuid.UUID, v3.Config]().Entry(&seed).Exec(ctx, db)).
				To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "arc_config",
				Migrations: []migrate.Migration{v4.Migration},
			})).To(Succeed())
			var got v4.Config
			Expect(gorp.NewRetrieve[uuid.UUID, v4.Config]().
				Where(gorp.MatchKeys[uuid.UUID, v4.Config](seed.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.Key).To(Equal(seed.Key))
			Expect(got.AutoStart).To(BeTrue())
			Expect(got.ArcKey).To(Equal(seed.ArcKey))
			Expect(got.Hash).To(Equal("abc123"))
			Expect(got.Performance).To(Equal(want))
		},
		Entry("auto", v3.ExecutionModeAuto, v4.PerformanceAuto),
		Entry("event driven", v3.ExecutionModeEventDriven, v4.PerformanceLow),
		Entry("hybrid", v3.ExecutionModeHybrid, v4.PerformanceMedium),
		Entry("real-time event", v3.ExecutionModeRtEvent, v4.PerformanceMedium),
		Entry("high rate", v3.ExecutionModeHighRate, v4.PerformanceHigh),
		Entry("busy wait", v3.ExecutionModeBusyWait, v4.PerformanceHigh),
	)
})
