// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil provides test helpers for gorp migration testing.
// It simplifies the common pattern of seeding old entries, running migrations
// via OpenTable, and asserting on the migrated results.
package testutil

import (
	"context"
	stdbinary "encoding/binary"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/types"
)

// Result wraps the outcome of a migration test, providing convenient
// methods for querying migrated data and checking migration metadata.
type Result[K gorp.Key, E gorp.Entry[K]] struct {
	Table *gorp.Table[K, E]
	DB    *gorp.DB
}

// Close releases the underlying in-memory database.
func (r *Result[K, E]) Close() error {
	return r.DB.Close()
}

// Entries retrieves all entries from the migrated table.
func (r *Result[K, E]) Entries(ctx context.Context) []E {
	var entries []E
	gomega.ExpectWithOffset(1,
		r.Table.NewRetrieve().Entries(&entries).Exec(ctx, r.DB),
	).To(gomega.Succeed())
	return entries
}

// Entry retrieves a single entry by key from the migrated table.
func (r *Result[K, E]) Entry(ctx context.Context, key K) E {
	var entry E
	gomega.ExpectWithOffset(1,
		r.Table.NewRetrieve().WhereKeys(key).Entry(&entry).Exec(ctx, r.DB),
	).To(gomega.Succeed())
	return entry
}

// EntryCount returns the number of entries in the migrated table.
func (r *Result[K, E]) EntryCount(ctx context.Context) int {
	count, err := r.Table.NewRetrieve().Count(ctx, r.DB)
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
	return count
}

// Version reads the migration version counter for the entry type.
func (r *Result[K, E]) Version(ctx context.Context) uint16 {
	versionKey := []byte("__gorp_migration__//" + types.Name[E]())
	b, closer, err := r.DB.Get(ctx, versionKey)
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
	gomega.ExpectWithOffset(1, closer.Close()).To(gomega.Succeed())
	gomega.ExpectWithOffset(1, b).To(gomega.HaveLen(2))
	return stdbinary.BigEndian.Uint16(b)
}

// SeedAndMigrate creates an in-memory DB, seeds old entries encoded with the
// default msgpack codec under the final type's key prefix, runs the provided
// migrations through OpenTable, and returns a Result for querying the migrated
// data. The seedCodec parameter is optional (nil uses msgpack). The finalCodec
// is the codec the table will use after migration (nil uses msgpack).
func SeedAndMigrate[K gorp.Key, Old gorp.Entry[K], New gorp.Entry[K]](
	ctx context.Context,
	seed []Old,
	seedCodec gorp.Codec[Old],
	migrations []gorp.Migration,
	finalCodec gorp.Codec[New],
) *Result[K, New] {
	db := gorp.Wrap(memkv.New())
	for _, entry := range seed {
		var (
			data []byte
			err  error
		)
		if seedCodec != nil {
			data, err = seedCodec.Marshal(ctx, entry)
		} else {
			data, err = db.Encode(ctx, entry)
		}
		gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
		key := gorp.EncodeKey[K, New](entry.GorpKey())
		gomega.ExpectWithOffset(1, db.Set(ctx, key, data)).To(gomega.Succeed())
	}
	tbl, err := gorp.OpenTable[K, New](ctx, gorp.TableConfig[New]{
		DB:         db,
		Codec:      finalCodec,
		Migrations: migrations,
	})
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
	return &Result[K, New]{Table: tbl, DB: db}
}

// RunAutoPost runs the auto-migrate function followed by the post-migrate
// function on the provided input, returning the transformed output. Either
// auto or post may be nil but not both.
func RunAutoPost[I, O any](
	ctx context.Context,
	input I,
	auto gorp.AutoMigrateFunc[I, O],
	post gorp.PostMigrateFunc[I, O],
) O {
	var result O
	if auto != nil {
		var err error
		result, err = auto(ctx, input)
		gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
	}
	if post != nil {
		gomega.ExpectWithOffset(1, post(ctx, &result, input)).To(gomega.Succeed())
	}
	return result
}
