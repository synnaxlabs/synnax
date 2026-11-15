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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	v3 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("NewDeleteMigration", func() {
	var (
		logs    *observer.ObservedLogs
		kept    = vis{Key: uuid.New(), Name: "Kept"}
		deleted = vis{Key: uuid.New(), Name: "Deleted"}
		stored  []vis
	)
	BeforeEach(func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		Expect(gorp.NewCreate[uuid.UUID, vis]().
			Entries(&[]vis{kept, deleted}).
			Exec(ctx, db)).To(Succeed())
		var core zapcore.Core
		core, logs = observer.New(zapcore.WarnLevel)
		logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
			DB:              db,
			Namespace:       "vis",
			Migrations: []gorp.Migration{
				v3.NewDeleteMigration(v3.DeleteConfig[uuid.UUID, vis]{
					Name:   "delete",
					Type:   resource.TypeSchematic,
					NameOf: func(v vis) string { return v.Name },
					Reason: func(v vis) string {
						if v.Key == deleted.Key {
							return "test reason"
						}
						return ""
					},
				}),
			},
		})).To(Succeed())
		stored = retrieveAll[uuid.UUID, vis](ctx, db)
	})

	It("Should delete each visualization with a reason", func() {
		Expect(stored).To(ConsistOf(kept))
	})

	It("Should log a warning with the reason for each deletion", func() {
		Expect(logs.FilterMessage("deleted a visualization").All()).To(ConsistOf(
			WithTransform(
				func(e observer.LoggedEntry) map[string]any { return e.ContextMap() },
				Equal(map[string]any{
					"type":   string(resource.TypeSchematic),
					"key":    deleted.Key.String(),
					"name":   "Deleted",
					"reason": "test reason",
				}),
			),
		))
	})
})
