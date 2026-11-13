// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v11_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	v10 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v10"
	v11 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v11"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("SnapshotMigration", func() {
	var (
		logs     *observer.ObservedLogs
		snapshot = v10.Schematic{Key: uuid.New(), Name: "Snapshot", Snapshot: true}
		live     = v10.Schematic{Key: uuid.New(), Name: "Live"}
		stored   []v10.Schematic
	)
	BeforeEach(func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v10.Key, v10.Schematic]{DB: db}))
		Expect(gorp.NewCreate[v10.Key, v10.Schematic]().
			Entries(&[]v10.Schematic{snapshot, live}).
			Exec(ctx, db)).To(Succeed())
		var core zapcore.Core
		core, logs = observer.New(zapcore.WarnLevel)
		logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
			DB:              db,
			Namespace:       "Schematic",
			Migrations:      []gorp.Migration{v11.SnapshotMigration},
		})).To(Succeed())
		stored = nil
		Expect(gorp.NewRetrieve[v10.Key, v10.Schematic]().
			Entries(&stored).
			Exec(ctx, db)).To(Succeed())
	})

	It("Should delete each stored snapshot", func() {
		Expect(stored).To(ConsistOf(HaveField("Key", live.Key)))
	})

	It("Should log a warning for each deleted snapshot", func() {
		Expect(logs.FilterMessage("deleted a schematic snapshot").All()).To(ConsistOf(
			WithTransform(
				func(e observer.LoggedEntry) map[string]any { return e.ContextMap() },
				Equal(map[string]any{"key": snapshot.Key.String(), "name": "Snapshot"}),
			),
		))
	})
})
