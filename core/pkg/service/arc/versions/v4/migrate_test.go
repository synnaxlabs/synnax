// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	graph "github.com/synnaxlabs/arc/graph/versions/v1"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	v3 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

const (
	oldText = "sensor -> scale{factor=2.0} -> out"
	newText = "sensor -> scale(factor=2.0) -> out"
)

var _ = Describe("MigrateArc", func() {
	It("Should rewrite the text of a stored Arc into a fresh document", func(
		ctx SpecContext,
	) {
		seed := v3.Arc{
			Key:  uuid.New(),
			Name: "stored",
			Mode: v3.ModeText,
			Text: text.Text{Doc: text.Create(oldText)},
		}
		got := migrateFromV3(ctx, seed)
		Expect(got.Key).To(Equal(seed.Key))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Mode).To(Equal(seed.Mode))
		Expect(got.Text.Materialize().Raw).To(Equal(newText))
		Expect(got.Text.Raw).To(BeEmpty())
	})

	It("Should seed the document of an exported Arc from its raw text", func(
		ctx SpecContext,
	) {
		got := MustSucceed(v4.MigrateArc(ctx, v3.Arc{
			Name: "exported",
			Mode: v3.ModeText,
			Text: text.Text{Raw: oldText},
		}))
		Expect(got.Text.Materialize().Raw).To(Equal(newText))
		Expect(got.Text.Raw).To(BeEmpty())
	})

	It("Should leave an Arc with no text untouched", func(ctx SpecContext) {
		got := MustSucceed(v4.MigrateArc(ctx, v3.Arc{Name: "empty", Mode: v3.ModeText}))
		Expect(got.Text.Doc.Inserts).To(BeEmpty())
		Expect(got.Text.Raw).To(BeEmpty())
	})

	It("Should rewrite the text of a graph-mode Arc and keep its graph", func(
		ctx SpecContext,
	) {
		seed := v3.Arc{
			Key:   uuid.New(),
			Name:  "graph",
			Mode:  v3.ModeGraph,
			Graph: graph.Graph{Nodes: graph.Nodes{{Key: "n1"}}},
			Text:  text.Text{Doc: text.Create(oldText)},
		}
		got := migrateFromV3(ctx, seed)
		Expect(got.Mode).To(Equal(v3.ModeGraph))
		Expect(got.Graph).To(Equal(seed.Graph))
		Expect(got.Text.Materialize().Raw).To(Equal(newText))
	})
})

// migrateFromV3 runs the v4 migration over a gorp-seeded v3 Arc and returns the
// migrated current Arc.
func migrateFromV3(ctx SpecContext, seed v3.Arc) v4.Arc {
	GinkgoHelper()
	db := DeferClose(gorp.Wrap(memkv.New()))
	MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v3.Key, v3.Arc]{DB: db}))
	Expect(gorp.NewCreate[v3.Key, v3.Arc]().Entry(&seed).Exec(ctx, db)).To(Succeed())
	Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
		DB:         db,
		Namespace:  "Arc",
		Migrations: []gorp.Migration{v4.Migration},
	})).To(Succeed())
	var got v4.Arc
	Expect(gorp.NewRetrieve[v4.Key, v4.Arc]().
		Where(gorp.MatchKeys[v4.Key, v4.Arc](seed.Key)).
		Entry(&got).Exec(ctx, db)).To(Succeed())
	return got
}
