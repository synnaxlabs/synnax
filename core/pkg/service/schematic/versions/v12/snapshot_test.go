// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v12_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	v11 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v11"
	v12 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v12"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("SnapshotMigration", func() {
	var (
		logs     *observer.ObservedLogs
		snapshot = v11.Schematic{Key: uuid.New(), Name: "Snapshot", Snapshot: true}
		ranged   = v11.Schematic{
			Key:   uuid.New(),
			Name:  "Ranged",
			Range: new(uuid.New()),
		}
		live = v11.Schematic{
			Key:     uuid.New(),
			Name:    "Live",
			Project: new(uuid.New()),
		}
		stored []v11.Schematic
	)
	BeforeEach(func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v11.Key, v11.Schematic]{DB: db}))
		Expect(gorp.NewCreate[v11.Key, v11.Schematic]().
			Entries(&[]v11.Schematic{snapshot, ranged, live}).
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
			Migrations:      []gorp.Migration{v12.SnapshotMigration},
		})).To(Succeed())
		stored = nil
		Expect(gorp.NewRetrieve[v11.Key, v11.Schematic]().
			Entries(&stored).
			Exec(ctx, db)).To(Succeed())
	})

	It("Should delete each stored snapshot and range-owned schematic", func() {
		Expect(stored).To(ConsistOf(HaveField("Key", live.Key)))
	})

	It("Should log a warning for each deleted schematic", func() {
		deleted := func(s v11.Schematic, reason string) map[string]any {
			return map[string]any{
				"type":   "schematic",
				"key":    s.Key.String(),
				"name":   s.Name,
				"reason": reason,
			}
		}
		var got []map[string]any
		for _, e := range logs.FilterMessage("deleted a visualization").All() {
			got = append(got, e.ContextMap())
		}
		Expect(got).To(ConsistOf(
			deleted(snapshot, "snapshot"),
			deleted(ranged, "owned by a range"),
		))
	})
})
