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

var _ = Describe("Resource", func() {
	Describe("GorpKey", func() {
		It("Should return the resource ID formatted as type:key", func() {
			r := v0.Resource{ID: v0.ID{Type: "channel", Key: "qux"}}
			Expect(r.GorpKey()).To(Equal("channel:qux"))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v0.Resource{}.SetOptions()).To(BeNil())
		})
	})
})

var _ = Describe("ResourceNormalizeKeys", func() {
	It(
		"Should lift a Resource row stored under the pre-v0.54 key format",
		func(ctx SpecContext) {
			kvDB := memkv.New()
			db := DeferClose(gorp.Wrap(kvDB, gorp.WithCodec(msgpack.Codec)))
			e := v0.Resource{
				ID:   v0.ID{Type: v0.ResourceTypeLabel, Key: "critical"},
				Name: "Critical",
			}
			legacy := gorptestutil.SetPreV54Row(
				ctx,
				kvDB,
				"Resource",
				e.GorpKey(),
				e,
			)
			table := MustOpen(gorp.OpenTable(
				ctx,
				gorp.TableConfig[string, v0.Resource]{
					DB:         db,
					Migrations: []migrate.Migration{v0.ResourceNormalizeKeys},
				},
			))
			var res v0.Resource
			Expect(table.NewRetrieve().
				Where(gorp.MatchKeys[string, v0.Resource](e.GorpKey())).
				Entry(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(Equal(e))
			Expect(db.Get(ctx, legacy)).Error().To(MatchError(query.ErrNotFound))
		},
	)
})
