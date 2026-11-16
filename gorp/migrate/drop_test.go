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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	. "github.com/synnaxlabs/gorp/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/kv"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
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
				DB:         db,
				Namespace:  "composition",
				Migrations: []gorp.Migration{migrate.Drop("drop", "droppedEntry")},
			})).To(Succeed())
			Expect(keysNaming("droppedEntry")).To(BeEmpty())
		})

	It("Should keep the entries of other tables", func(ctx SpecContext) {
		Expect(gorp.NewCreate[int32, entryV1]().
			Entry(&entryV1{ID: 1, Data: "kept"}).
			Exec(ctx, db)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "composition",
			Migrations: []gorp.Migration{migrate.Drop("drop", "droppedEntry")},
		})).To(Succeed())
		var res entryV1
		Expect(gorp.NewRetrieve[int32, entryV1]().
			Where(gorp.MatchKeys[int32, entryV1](1)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		Expect(res.Data).To(Equal("kept"))
	})
})
