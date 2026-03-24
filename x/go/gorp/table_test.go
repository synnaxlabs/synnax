// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/types"
)

var _ = Describe("Table", func() {
	var (
		ctx context.Context
		kvs kv.DB
		db  *gorp.DB
	)
	BeforeEach(func() {
		ctx = context.Background()
		kvs = memkv.New()
		db = gorp.Wrap(kvs)
	})
	AfterEach(func() { Expect(db.Close()).To(Succeed()) })

	Describe("OpenTable", func() {
		It("Should open a table on an empty database", func() {
			table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[entry]{DB: db}))
			Expect(table.Close()).To(Succeed())
		})

		It("Should be idempotent when called multiple times", func() {
			e := entry{ID: 1, Data: "data"}
			Expect(gorp.NewCreate[int32, entry]().Entry(&e).Exec(ctx, db)).To(Succeed())

			table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[entry]{DB: db}))
			Expect(table.Close()).To(Succeed())

			table = MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[entry]{DB: db}))
			Expect(table.Close()).To(Succeed())

			var res entry
			Expect(gorp.NewRetrieve[int32, entry]().
				WhereKeys(1).Entry(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(Equal(e))
		})

		It("Should preserve entries after re-encoding", func() {
			entries := []entry{
				{ID: 1, Data: "one"},
				{ID: 2, Data: "two"},
				{ID: 3, Data: "three"},
			}
			Expect(gorp.NewCreate[int32, entry]().
				Entries(&entries).Exec(ctx, db)).To(Succeed())

			table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[entry]{DB: db}))
			Expect(table.Close()).To(Succeed())

			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				WhereKeys(1, 2, 3).
				Entries(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(Equal(entries))
		})

		It("Should work with uint64 keys", func() {
			entries := []uint64Entry{
				{ID: 1, Data: "one"},
				{ID: 999999999, Data: "big"},
			}
			Expect(gorp.NewCreate[uint64, uint64Entry]().
				Entries(&entries).Exec(ctx, db)).To(Succeed())

			table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[uint64Entry]{DB: db}))
			Expect(table.Close()).To(Succeed())

			var res []uint64Entry
			Expect(gorp.NewRetrieve[uint64, uint64Entry]().
				WhereKeys(1, 999999999).
				Entries(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(HaveLen(2))
		})

		It("Should work with string keys", func() {
			entries := []stringEntry{
				{ID: "alpha", Data: "first"},
				{ID: "beta", Data: "second"},
			}
			Expect(gorp.NewCreate[string, stringEntry]().
				Entries(&entries).Exec(ctx, db)).To(Succeed())

			table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[stringEntry]{DB: db}))
			Expect(table.Close()).To(Succeed())

			var res []stringEntry
			Expect(gorp.NewRetrieve[string, stringEntry]().
				WhereKeys("alpha", "beta").
				Entries(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(Equal(entries))
		})
	})

	Describe("MigrateOldPrefixKeys", func() {
		writeOldFormatEntry := func(codec *binary.MsgPackCodec, e entry) {
			typeName := types.Name[entry]()
			oldPrefix := MustSucceed(codec.Encode(ctx, typeName))
			encodedValue := MustSucceed(codec.Encode(ctx, e))
			// The key suffix doesn't matter for migration because
			// migrateOldPrefixKeys decodes the VALUE to reconstruct the
			// entry. We use an arbitrary suffix to differentiate keys.
			encodedKey := MustSucceed(codec.Encode(ctx, e.ID))
			fullKey := make([]byte, len(oldPrefix)+len(encodedKey))
			copy(fullKey, oldPrefix)
			copy(fullKey[len(oldPrefix):], encodedKey)
			Expect(kvs.Set(ctx, fullKey, encodedValue)).To(Succeed())
		}

		It("Should migrate entries stored under old codec-based prefix", func() {
			codec := &binary.MsgPackCodec{}
			writeOldFormatEntry(codec, entry{ID: 42, Data: "old format"})

			table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[entry]{DB: db}))
			Expect(table.Close()).To(Succeed())

			var res entry
			Expect(gorp.NewRetrieve[int32, entry]().
				WhereKeys(42).Entry(&res).Exec(ctx, db)).To(Succeed())
			Expect(res.Data).To(Equal("old format"))
		})

		It("Should remove entries from the old prefix after migration", func() {
			codec := &binary.MsgPackCodec{}
			writeOldFormatEntry(codec, entry{ID: 7, Data: "migrate me"})

			oldPrefix := MustSucceed(codec.Encode(ctx, types.Name[entry]()))
			iter := MustSucceed(kvs.OpenIterator(kv.IterPrefix(oldPrefix)))
			iter.First()
			Expect(iter.Valid()).To(BeTrue())
			Expect(iter.Close()).To(Succeed())

			table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[entry]{DB: db}))
			Expect(table.Close()).To(Succeed())

			iter = MustSucceed(kvs.OpenIterator(kv.IterPrefix(oldPrefix)))
			iter.First()
			Expect(iter.Valid()).To(BeFalse())
			Expect(iter.Close()).To(Succeed())
		})

		It("Should handle migration with multiple old-format entries", func() {
			codec := &binary.MsgPackCodec{}
			for i := range 5 {
				writeOldFormatEntry(codec, entry{ID: int32(i), Data: "old"})
			}

			table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[entry]{DB: db}))
			Expect(table.Close()).To(Succeed())

			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(HaveLen(5))
		})

		It("Should not duplicate entries already stored under the new prefix", func() {
			e := entry{ID: 10, Data: "already new"}
			Expect(gorp.NewCreate[int32, entry]().Entry(&e).Exec(ctx, db)).To(Succeed())

			table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[entry]{DB: db}))
			Expect(table.Close()).To(Succeed())

			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(HaveLen(1))
			Expect(res[0]).To(Equal(e))
		})

		It("Should migrate old-format entries while preserving new-format entries", func() {
			codec := &binary.MsgPackCodec{}
			writeOldFormatEntry(codec, entry{ID: 1, Data: "old"})

			newEntry := entry{ID: 2, Data: "new"}
			Expect(gorp.NewCreate[int32, entry]().
				Entry(&newEntry).Exec(ctx, db)).To(Succeed())

			table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[entry]{DB: db}))
			Expect(table.Close()).To(Succeed())

			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				WhereKeys(1, 2).
				Entries(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(HaveLen(2))
		})
	})
})
