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
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/types/v0"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Legacy mappings", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })

	writeMapping := func(ctx SpecContext, mappings []v0.LegacyUserMapping) {
		raw := MustSucceed(json.Marshal(mappings))
		Expect(db.Set(ctx, []byte(v0.LegacyMappingKVKey), raw)).To(Succeed())
	}

	Describe("ReadLegacyMappings", func() {
		It("Should return nil when no mapping is stored", func(ctx SpecContext) {
			Expect(v0.ReadLegacyMappings(ctx, db)).To(BeNil())
		})
		It("Should read the persisted mapping", func(ctx SpecContext) {
			id := ontology.ID{Type: ontology.ResourceTypeUser, Key: "u1"}
			writeMapping(ctx, []v0.LegacyUserMapping{{UserOntologyID: id}})
			got := MustSucceed(v0.ReadLegacyMappings(ctx, db))
			Expect(got).To(HaveLen(1))
			Expect(got[0].UserOntologyID).To(Equal(id))
		})
	})

	Describe("DeleteLegacyMappings", func() {
		It("Should remove the persisted mapping", func(ctx SpecContext) {
			writeMapping(ctx, []v0.LegacyUserMapping{{
				UserOntologyID: ontology.ID{Type: ontology.ResourceTypeUser, Key: "u1"},
			}})
			Expect(v0.DeleteLegacyMappings(ctx, db)).To(Succeed())
			Expect(v0.ReadLegacyMappings(ctx, db)).To(BeNil())
		})
	})
})
