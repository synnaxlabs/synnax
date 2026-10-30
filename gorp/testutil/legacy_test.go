// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	. "github.com/synnaxlabs/gorp/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("SetPreV54Row", func() {
	It(
		"Should write the row at msgpack(typeName) + msgpack(key)",
		func(ctx SpecContext) {
			kvDB := DeferClose(memkv.New())
			e := entry{ID: 7, Data: "old"}
			key := SetPreV54Row(ctx, kvDB, "legacyEntry", e.ID, e)

			prefix := MustSucceed(msgpack.Codec.Encode(ctx, "legacyEntry"))
			encodedKey := MustSucceed(msgpack.Codec.Encode(ctx, e.ID))
			Expect(key).To(Equal(append(prefix, encodedKey...)))

			raw, closer := MustSucceed2(kvDB.Get(ctx, key))
			var stored entry
			Expect(msgpack.Codec.Decode(ctx, raw, &stored)).To(Succeed())
			Expect(closer.Close()).To(Succeed())
			Expect(stored).To(Equal(e))
		},
	)

	It(
		"Should write a row NormalizeKeys lifts to the current prefix",
		func(ctx SpecContext) {
			kvDB := memkv.New()
			db := DeferClose(gorp.Wrap(kvDB, gorp.WithCodec(msgpack.Codec)))
			e := entry{ID: 7, Data: "old"}
			SetPreV54Row(ctx, kvDB, "entry", e.ID, e)
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{
				DB: db,
				Migrations: []gorp.Migration{
					migrate.NormalizeKeys[int32, entry]("entry"),
				},
			}))
			r := gorp.WrapReader[int32, entry](db)
			Expect(r.Get(ctx, e.ID)).To(Equal(e))
		},
	)
})
