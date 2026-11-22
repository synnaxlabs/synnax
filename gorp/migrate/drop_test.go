// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	. "github.com/synnaxlabs/gorp/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/kv"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// droppedEntry is the entry type of the table that Drop deletes.
type droppedEntry struct {
	ID   int32
	Data string
}

func (d droppedEntry) GorpKey() int32 { return d.ID }

func (droppedEntry) CustomTypeName() string { return "droppedEntry" }

var _ = Describe("Drop", func() {
	var (
		kvs kv.DB
		db  *gorp.DB
	)
	BeforeEach(func() {
		kvs = memkv.New()
		db = DeferClose(gorp.Wrap(kvs, gorp.WithCodec(msgpack.Codec)))
	})

	keysNaming := func(name string) [][]byte {
		GinkgoHelper()
		itr := MustSucceed(kvs.OpenIterator(kv.IteratorOptions{}))
		defer func() { Expect(itr.Close()).To(Succeed()) }()
		var keys [][]byte
		for itr.First(); itr.Valid(); itr.Next() {
			if bytes.Contains(itr.Key(), []byte(name)) {
				keys = append(keys, bytes.Clone(itr.Key()))
			}
		}
		return keys
	}

	It("Should delete every entry and the migration record of the table",
		func(ctx SpecContext) {
			table := MustSucceed(
				gorp.OpenTable(ctx, gorp.TableConfig[int32, droppedEntry]{
					DB: db,
					Migrations: []gorp.Migration{
						migrate.Codec[int32, droppedEntry]("codec"),
					},
				}),
			)
			Expect(table.Close()).To(Succeed())
			Expect(gorp.NewCreate[int32, droppedEntry]().
				Entries(&[]droppedEntry{{ID: 1}, {ID: 2}}).
				Exec(ctx, db)).To(Succeed())
			SetPreV54Row(ctx, kvs, "droppedEntry", int32(3), droppedEntry{ID: 3})
			Expect(keysNaming("droppedEntry")).To(HaveLen(4))
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:        db,
				Namespace: "composition",
				Migrations: []gorp.Migration{
					migrate.Drop[int32]("drop", "droppedEntry"),
				},
			})).To(Succeed())
			Expect(keysNaming("droppedEntry")).To(BeEmpty())
		})

	It("Should log a warning with the key of each entry it deletes",
		func(ctx SpecContext) {
			Expect(gorp.NewCreate[int32, droppedEntry]().
				Entries(&[]droppedEntry{{ID: 1}, {ID: 2}}).
				Exec(ctx, db)).To(Succeed())
			SetPreV54Row(ctx, kvs, "droppedEntry", int32(3), droppedEntry{ID: 3})
			core, logs := observer.New(zap.WarnLevel)
			ins := alamos.New("gorp", alamos.WithLogger(MustSucceed(
				alamos.NewLogger(alamos.LoggerConfig{ZapLogger: zap.New(core)}),
			)))
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				Instrumentation: ins,
				DB:              db,
				Namespace:       "composition",
				Migrations: []gorp.Migration{
					migrate.Drop[int32]("drop", "droppedEntry"),
				},
			})).To(Succeed())
			deleted := func(key string) map[string]any {
				return map[string]any{"type": "droppedEntry", "key": key}
			}
			logged := logs.FilterMessage("deleted an entry").All()
			got := make([]map[string]any, len(logged))
			for i, e := range logged {
				got[i] = e.ContextMap()
			}
			Expect(got).To(ConsistOf(deleted("1"), deleted("2"), deleted("3")))
		})

	It("Should keep the entries of other tables", func(ctx SpecContext) {
		Expect(gorp.NewCreate[int32, entryV1]().
			Entry(&entryV1{ID: 1, Data: "kept"}).
			Exec(ctx, db)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "composition",
			Migrations: []gorp.Migration{migrate.Drop[int32]("drop", "droppedEntry")},
		})).To(Succeed())
		var res entryV1
		Expect(gorp.NewRetrieve[int32, entryV1]().
			Where(gorp.MatchKeys[int32, entryV1](1)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		Expect(res.Data).To(Equal("kept"))
	})
})
