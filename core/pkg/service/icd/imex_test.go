// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package library_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/synnax/pkg/service/library/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ImEx", func() {
	It("Should never match a legacy file", func() {
		Expect(svc.Match(map[string]any{"entries": []any{}})).To(BeFalse())
	})

	It("Should round trip a library through export and import", func(ctx SpecContext) {
		lib := library.Library{
			Name: "Exported",
			Entries: []library.Entry{
				enumEntry("State", library.EnumValue{Value: 1, Name: "On"}),
				messageEntry(canMessage("Status", 0x100, binaryField("rpm", 0, 16))),
			},
		}
		Expect(svc.NewWriter(nil).Create(ctx, &lib)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(svc.NewWriter(nil).Delete(ctx, lib.Key)).To(Succeed())
		})
		env := MustSucceed(svc.Export(ctx, library.OntologyID(lib.Key)))
		Expect(env.Version).To(Equal(versions.Latest))
		Expect(env.Type).To(Equal("library"))
		Expect(env.Name).To(Equal("Exported"))
		id := MustSucceed(
			imexSvc.Import(ctx, db, WireRoundTrip(env), imex.ImportOptions{
				Parent: ontology.RootID,
			}),
		)
		Expect(id.Type).To(Equal(ontology.ResourceTypeLibrary))
		key := MustSucceed(uuid.Parse(id.Key))
		DeferCleanup(func(ctx SpecContext) {
			Expect(svc.NewWriter(nil).Delete(ctx, key)).To(Succeed())
		})
		Expect(key).ToNot(Equal(lib.Key))
		var res library.Library
		Expect(svc.NewRetrieve().
			Where(library.MatchKeys(key)).
			Entry(&res).
			Exec(ctx, nil)).To(Succeed())
		Expect(res.Name).To(Equal("Exported"))
		Expect(library.Hash(res)).To(Equal(MustSucceed(library.Hash(lib))))
	})

	It(
		"Should return not found when exporting a missing library",
		func(ctx SpecContext) {
			Expect(svc.Export(ctx, library.OntologyID(uuid.New()))).
				Error().To(MatchError(query.ErrNotFound))
		},
	)
})
