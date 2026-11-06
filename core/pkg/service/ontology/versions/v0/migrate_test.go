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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	gorptestutil "github.com/synnaxlabs/gorp/testutil"
	v0 "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migrate", func() {
	It(
		"Should move legacy resources and relationships to the current key and codec",
		func(ctx SpecContext) {
			kvDB := memkv.New()
			db := DeferClose(gorp.Wrap(kvDB))
			res := v0.Resource{
				ID: resource.ID{Type: resource.TypeLabel, Key: "critical"},
			}
			rel := v0.Relationship{
				From: resource.ID{Type: resource.TypeGroup, Key: "sensors"},
				Type: v0.RelationshipTypeParentOf,
				To:   res.ID,
			}
			legacyRes := gorptestutil.SetPreV54Row(
				ctx,
				kvDB,
				"Resource",
				res.GorpKey(),
				res,
			)
			legacyRel := gorptestutil.SetPreV54Row(
				ctx,
				kvDB,
				"Relationship",
				rel.GorpKey(),
				rel,
			)
			Expect(v0.Migrate(ctx, db, alamos.Instrumentation{})).To(Succeed())
			var resources []v0.Resource
			Expect(gorp.NewRetrieve[string, v0.Resource]().
				Entries(&resources).
				Exec(ctx, db)).To(Succeed())
			Expect(resources).To(ConsistOf(res))
			Expect(v0.RetrieveRelationships(
				ctx,
				db,
				func(v0.Relationship) bool { return true },
			)).To(ConsistOf(rel))
			Expect(db.Get(ctx, legacyRes)).Error().To(MatchError(query.ErrNotFound))
			Expect(db.Get(ctx, legacyRel)).Error().To(MatchError(query.ErrNotFound))
		},
	)
})
