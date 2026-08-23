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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	v0 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	gorptestutil "github.com/synnaxlabs/x/gorp/testutil"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("View", func() {
	Describe("GorpKey", func() {
		It("Should return the view's key", func() {
			k := uuid.New()
			Expect(v0.View{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v0.View{}.SetOptions()).To(BeNil())
		})
	})
	Describe("OntologyID", func() {
		It("Should return the view ontology identifier", func() {
			k := uuid.New()
			Expect(v0.View{Key: k}.OntologyID()).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeView, Key: k.String(),
			}))
		})
	})
})

var _ = Describe("NormalizeKeys", func() {
	It(
		"Should lift a View row stored under the pre-v0.54 key format",
		func(ctx SpecContext) {
			kvDB := memkv.New()
			db := DeferClose(gorp.Wrap(kvDB, gorp.WithCodec(msgpack.Codec)))
			e := v0.View{Key: uuid.New(), Name: "Recent", Type: "range"}
			legacy := gorptestutil.SetPreV54Row(
				ctx,
				kvDB,
				"View",
				e.GorpKey(),
				e,
			)
			table := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.View]{
				DB:         db,
				Migrations: []migrate.Migration{v0.NormalizeKeys},
			}))
			var res v0.View
			Expect(table.NewRetrieve().
				Where(gorp.MatchKeys[v0.Key, v0.View](e.GorpKey())).
				Entry(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(Equal(e))
			Expect(db.Get(ctx, legacy)).Error().To(MatchError(query.ErrNotFound))
		},
	)
})
