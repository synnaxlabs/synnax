// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	schematic "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v11"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func parentOf(from, to resource.ID) ontology.Relationship {
	return ontology.Relationship{
		From: from,
		Type: ontology.RelationshipTypeParentOf,
		To:   to,
	}
}

func schematicID(k uuid.UUID) resource.ID {
	return resource.ID{Type: resource.TypeSchematic, Key: k.String()}
}

func create[K gorp.Key, E gorp.Entry[K]](ctx context.Context, db *gorp.DB, es ...E) {
	GinkgoHelper()
	Expect(gorp.NewCreate[K, E]().Entries(&es).Exec(ctx, db)).To(Succeed())
}

func retrieveAll[K gorp.Key, E gorp.Entry[K]](ctx context.Context, db *gorp.DB) []E {
	GinkgoHelper()
	var out []E
	Expect(gorp.NewRetrieve[K, E]().Entries(&out).Exec(ctx, db)).
		To(Or(Succeed(), MatchError(query.ErrNotFound)))
	return out
}

var _ = Describe("OwnersMigration", func() {
	var (
		db       *gorp.DB
		proj     = uuid.New()
		rng      = uuid.New()
		projID   = resource.ID{Type: resource.TypeProject, Key: proj.String()}
		rngID    = resource.ID{Type: resource.TypeRange, Key: rng.String()}
		owned    = uuid.New()
		snapshot = uuid.New()
		both     = uuid.New()
		orphan   = uuid.New()
		logKey   = uuid.New()
		logID    = resource.ID{Type: resource.TypeLog, Key: logKey.String()}
	)

	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		create[schematic.Key](
			ctx,
			db,
			schematic.Schematic{Key: owned, Name: "owned"},
			schematic.Schematic{Key: snapshot, Name: "snapshot", Snapshot: true},
			schematic.Schematic{Key: both, Name: "both"},
			schematic.Schematic{Key: orphan, Name: "orphan"},
		)
		create[string](
			ctx,
			db,
			parentOf(projID, schematicID(owned)),
			parentOf(rngID, schematicID(snapshot)),
			parentOf(rngID, schematicID(both)),
			parentOf(projID, schematicID(both)),
			parentOf(schematicID(orphan), logID),
			parentOf(projID, logID),
		)
		create[string](
			ctx,
			db,
			ontology.Resource{ID: schematicID(owned)},
			ontology.Resource{ID: schematicID(orphan)},
		)
		tx := db.OpenTx()
		Expect(schematic.OwnersMigration.Run(ctx, tx, alamos.Instrumentation{})).
			To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())
		Expect(tx.Close()).To(Succeed())
	})

	byKey := func(ctx context.Context) map[uuid.UUID]schematic.Schematic {
		GinkgoHelper()
		out := make(map[uuid.UUID]schematic.Schematic)
		for _, s := range retrieveAll[schematic.Key, schematic.Schematic](ctx, db) {
			out[s.Key] = s
		}
		return out
	}

	It("Should set the project of a schematic under a project", func(ctx SpecContext) {
		s := byKey(ctx)[owned]
		Expect(s.Project).To(HaveValue(Equal(proj)))
		Expect(s.Range).To(BeNil())
	})

	It("Should set the range of a snapshot under a range", func(ctx SpecContext) {
		s := byKey(ctx)[snapshot]
		Expect(s.Range).To(HaveValue(Equal(rng)))
		Expect(s.Project).To(BeNil())
	})

	It("Should prefer the project when both own a schematic", func(ctx SpecContext) {
		s := byKey(ctx)[both]
		Expect(s.Project).To(HaveValue(Equal(proj)))
		Expect(s.Range).To(BeNil())
	})

	It("Should delete a schematic with no owner", func(ctx SpecContext) {
		Expect(byKey(ctx)).ToNot(HaveKey(orphan))
		Expect(retrieveAll[string, ontology.Resource](ctx, db)).To(ConsistOf(
			ontology.Resource{ID: schematicID(owned)},
		))
	})

	It("Should delete the converted edges and the edges of deleted schematics", func(
		ctx SpecContext,
	) {
		Expect(retrieveAll[string, ontology.Relationship](ctx, db)).To(ConsistOf(
			parentOf(projID, logID),
		))
	})
})
