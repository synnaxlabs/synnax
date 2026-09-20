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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	gorptestutil "github.com/synnaxlabs/x/gorp/testutil"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Relationship", func() {
	Describe("GorpKey", func() {
		It("Should return the correct gorp key", func() {
			Expect(v0.Relationship{
				From: v0.ID{Type: "channel", Key: "qux"},
				To:   v0.ID{Type: "device", Key: "baz"},
				Type: v0.RelationshipTypeParentOf,
			}.GorpKey()).To(Equal("channel:qux->parent->device:baz"))
		})
	})
	Describe("SetOptions", func() {
		It("Should return nil", func() {
			Expect(v0.Relationship{}.SetOptions()).To(BeNil())
		})
	})
})

var _ = Describe("RelationshipNormalizeKeys", func() {
	It(
		"Should lift a Relationship row stored under the pre-v0.54 key format",
		func(ctx SpecContext) {
			kvDB := memkv.New()
			db := DeferClose(gorp.Wrap(kvDB, gorp.WithCodec(msgpack.Codec)))
			e := v0.Relationship{
				From: v0.ID{Type: v0.ResourceTypeGroup, Key: "devices"},
				Type: v0.RelationshipTypeParentOf,
				To:   v0.ID{Type: v0.ResourceTypeLabel, Key: "critical"},
			}
			legacy := gorptestutil.SetPreV54Row(
				ctx,
				kvDB,
				"Relationship",
				e.GorpKey(),
				e,
			)
			table := MustOpen(gorp.OpenTable(
				ctx,
				gorp.TableConfig[string, v0.Relationship]{
					DB:         db,
					Migrations: []migrate.Migration{v0.RelationshipNormalizeKeys},
				},
			))
			var res v0.Relationship
			Expect(table.NewRetrieve().
				Where(gorp.MatchKeys[string, v0.Relationship](e.GorpKey())).
				Entry(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(Equal(e))
			Expect(db.Get(ctx, legacy)).Error().To(MatchError(query.ErrNotFound))
		},
	)
})
