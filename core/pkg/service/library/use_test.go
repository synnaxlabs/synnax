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
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func define(ctx context.Context, s *namedService, name string) ontology.ID {
	GinkgoHelper()
	key := uuid.New().String()
	s.names[key] = name
	id := ontology.ID{Type: s.t, Key: key}
	Expect(otg.NewWriter(tx).DefineResources(ctx, id)).To(Succeed())
	return id
}

var _ = Describe("Use", func() {
	It("Should record that a resource uses a document", func(c SpecContext) {
		doc, user := define(c, docs, "Bus"), define(c, tasks, "Reader")
		Expect(library.Use(c, otg.NewWriter(tx), user, &doc)).To(Succeed())
		users := MustSucceed(library.Users(c, tx, otg, doc))
		Expect(ontology.ResourceIDs(users)).To(ConsistOf(user))
	})

	It("Should replace the document a resource used before", func(c SpecContext) {
		first, second := define(c, docs, "First"), define(c, docs, "Second")
		user := define(c, tasks, "Reader")
		w := otg.NewWriter(tx)
		Expect(library.Use(c, w, user, &first)).To(Succeed())
		Expect(library.Use(c, w, user, &second)).To(Succeed())
		Expect(MustSucceed(library.Users(c, tx, otg, first))).To(BeEmpty())
		Expect(ontology.ResourceIDs(MustSucceed(library.Users(c, tx, otg, second)))).
			To(ConsistOf(user))
	})

	It("Should remove the use when the resource uses no document", func(c SpecContext) {
		doc, user := define(c, docs, "Bus"), define(c, tasks, "Reader")
		w := otg.NewWriter(tx)
		Expect(library.Use(c, w, user, &doc)).To(Succeed())
		Expect(library.Use(c, w, user, nil)).To(Succeed())
		Expect(MustSucceed(library.Users(c, tx, otg, doc))).To(BeEmpty())
	})

	It("Should find no users of a document with no resource", func(c SpecContext) {
		missing := ontology.ID{Type: ontology.ResourceTypeICD, Key: uuid.New().String()}
		Expect(MustSucceed(library.Users(c, tx, otg, missing))).To(BeEmpty())
	})
})

var _ = Describe("CheckUnused", func() {
	It("Should accept a document nothing uses", func(c SpecContext) {
		Expect(library.CheckUnused(c, tx, otg, define(c, docs, "Bus"))).To(Succeed())
	})

	It("Should name every resource that uses the document", func(c SpecContext) {
		doc := define(c, docs, "Bus")
		w := otg.NewWriter(tx)
		for _, name := range []string{"Reader", "Writer"} {
			Expect(library.Use(c, w, define(c, tasks, name), &doc)).To(Succeed())
		}
		Expect(library.CheckUnused(c, tx, otg, doc)).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("Bus is used by ")),
			MatchError(ContainSubstring("Reader")),
			MatchError(ContainSubstring("Writer")),
		))
	})
})
