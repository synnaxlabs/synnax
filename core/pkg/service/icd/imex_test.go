// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/versions"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ImEx", func() {
	It("Should never match a legacy file", func() {
		Expect(svc.Match(map[string]any{"messages": []any{}})).To(BeFalse())
	})

	It("Should round trip an ICD through export and import", func(ctx SpecContext) {
		ic := icd.ICD{
			Name: "Exported",
			Messages: []icd.Message{
				canMessage("Status", 0x100, binaryField("rpm", 0, 16)),
			},
		}
		Expect(svc.NewWriter(nil).Create(ctx, &ic)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(svc.NewWriter(nil).Delete(ctx, ic.Key)).To(Succeed())
		})
		env := MustSucceed(svc.Export(ctx, icd.OntologyID(ic.Key)))
		Expect(env.Version).To(Equal(versions.Latest))
		Expect(env.Type).To(Equal("icd"))
		Expect(env.Name).To(Equal("Exported"))
		id := MustSucceed(
			imexSvc.Import(ctx, db, WireRoundTrip(env), imex.ImportOptions{
				Parent: ontology.RootID,
			}),
		)
		Expect(id.Type).To(Equal(ontology.ResourceTypeICD))
		key := MustSucceed(uuid.Parse(id.Key))
		DeferCleanup(func(ctx SpecContext) {
			Expect(svc.NewWriter(nil).Delete(ctx, key)).To(Succeed())
		})
		Expect(key).ToNot(Equal(ic.Key))
		var res icd.ICD
		Expect(svc.NewRetrieve().
			Where(icd.MatchKeys(key)).
			Entry(&res).
			Exec(ctx, nil)).To(Succeed())
		Expect(res.Name).To(Equal("Exported"))
		Expect(res.Hash).To(Equal(ic.Hash))
	})

	It(
		"Should return not found when exporting a missing ICD",
		func(ctx SpecContext) {
			Expect(svc.Export(ctx, icd.OntologyID(uuid.New()))).
				Error().To(MatchError(query.ErrNotFound))
		},
	)
})
