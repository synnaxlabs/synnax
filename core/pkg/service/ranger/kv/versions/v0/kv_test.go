// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"encoding/hex"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/ranger/kv/versions/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	gorptestutil "github.com/synnaxlabs/x/gorp/testutil"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Pair", func() {
	Describe("GorpKey", func() {
		It("Should join the range key and pair key", func() {
			rng := uuid.New()
			Expect(v0.Pair{Range: rng, Key: "temperature"}.GorpKey()).
				To(Equal(rng.String() + "<--->" + "temperature"))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v0.Pair{}.SetOptions()).To(BeNil())
		})
	})

	Describe("CustomTypeName", func() {
		It("Should return the KVPair gorp type name", func() {
			Expect(v0.Pair{}.CustomTypeName()).To(Equal("KVPair"))
		})
	})
})

var _ = Describe("NormalizeKeys", func() {
	It(
		"Should lift a Pair row stored under the pre-v0.54 key format",
		func(ctx SpecContext) {
			kvDB := memkv.New()
			db := DeferClose(gorp.Wrap(kvDB, gorp.WithCodec(msgpack.Codec)))
			e := v0.Pair{Range: uuid.New(), Key: "operator", Value: "ada"}
			legacy := gorptestutil.SetPreV54Row(
				ctx,
				kvDB,
				"Pair",
				e.GorpKey(),
				e,
			)
			table := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, v0.Pair]{
				DB:         db,
				Migrations: []migrate.Migration{v0.NormalizeKeys},
			}))
			var res v0.Pair
			Expect(table.NewRetrieve().
				Where(gorp.MatchKeys[string, v0.Pair](e.GorpKey())).
				Entry(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(Equal(e))
			Expect(db.Get(ctx, legacy)).Error().To(MatchError(query.ErrNotFound))
		},
	)
})

// v0.51 and earlier stored pairs under the type name "KVPair". These are two rows a
// v0.49.5 Core wrote for one range.
var (
	kvPairRange = uuid.MustParse("1e706952-94fe-48be-a8cd-fc8d2a61b7db")
	kvPairRows  = map[string]string{
		"a64b5650616972d92a31653730363935322d393466652d343862652d613863642d666338" +
			"6432613631623764623c2d2d2d3e6b": "83a572616e6765c4101e70695294fe48bea8cd" +
			"fc8d2a61b7dba36b6579a16ba576616c7565a5616c706861",
		"a64b5650616972d92e31653730363935322d393466652d343862652d613863642d666338" +
			"6432613631623764623c2d2d2d3e656d707479": "83a572616e6765c4101e70695294fe" +
			"48bea8cdfc8d2a61b7dba36b6579a5656d707479a576616c7565a0",
	}
)

var _ = Describe("RecoverKVPairKeys", func() {
	var (
		kvDB kv.DB
		db   *gorp.DB
	)
	BeforeEach(func(ctx SpecContext) {
		kvDB = memkv.New()
		db = DeferClose(gorp.Wrap(kvDB))
		for k, v := range kvPairRows {
			Expect(kvDB.Set(
				ctx,
				MustSucceed(hex.DecodeString(k)),
				MustSucceed(hex.DecodeString(v)),
			)).To(Succeed())
		}
	})
	open := func(
		ctx SpecContext,
		migrations ...migrate.Migration,
	) *gorp.Table[string, v0.Pair] {
		GinkgoHelper()
		return MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, v0.Pair]{
			DB:         db,
			Migrations: migrations,
		}))
	}
	retrieve := func(ctx SpecContext, table *gorp.Table[string, v0.Pair]) []v0.Pair {
		GinkgoHelper()
		var res []v0.Pair
		Expect(table.NewRetrieve().Entries(&res).Exec(ctx, db)).To(Succeed())
		return res
	}
	expectLegacyRowsDeleted := func(ctx SpecContext) {
		GinkgoHelper()
		for k := range kvPairRows {
			Expect(kvDB.Get(ctx, MustSucceed(hex.DecodeString(k)))).Error().
				To(MatchError(query.ErrNotFound))
		}
	}
	DescribeTable("Should recover the rows a v0.51 Core stored",
		func(ctx SpecContext, preMigrated bool) {
			if preMigrated {
				open(ctx, v0.NormalizeKeys, v0.Migration)
			}
			table := open(ctx, v0.NormalizeKeys, v0.Migration, v0.RecoverKVPairKeys)
			Expect(retrieve(ctx, table)).To(ConsistOf(
				v0.Pair{Range: kvPairRange, Key: "k", Value: "alpha"},
				v0.Pair{Range: kvPairRange, Key: "empty", Value: ""},
			))
			expectLegacyRowsDeleted(ctx)
		},
		Entry("when upgrading straight to the recovery", false),
		Entry("when the store already ran the other migrations", true),
	)
	It("Should keep a newer value written under the same key", func(ctx SpecContext) {
		table := open(ctx, v0.NormalizeKeys, v0.Migration)
		newer := v0.Pair{Range: kvPairRange, Key: "k", Value: "newer"}
		Expect(table.NewCreate().Entry(&newer).Exec(ctx, db)).To(Succeed())
		table = open(ctx, v0.NormalizeKeys, v0.Migration, v0.RecoverKVPairKeys)
		Expect(retrieve(ctx, table)).To(ConsistOf(
			newer,
			v0.Pair{Range: kvPairRange, Key: "empty", Value: ""},
		))
		expectLegacyRowsDeleted(ctx)
	})
})
