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
	"github.com/synnaxlabs/gorp"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	v3 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

type vis struct {
	Key     uuid.UUID
	Project *uuid.UUID
	Range   *uuid.UUID
}

func (v vis) GorpKey() uuid.UUID { return v.Key }

func (vis) SetOptions() []any { return nil }

func parentOf(from, to resource.ID) ontology.Relationship {
	return ontology.Relationship{
		From: from,
		Type: ontology.RelationshipTypeParentOf,
		To:   to,
	}
}

func visID(k uuid.UUID) resource.ID {
	return resource.ID{Type: resource.TypeSchematic, Key: k.String()}
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
		missing  = resource.ID{Type: resource.TypeProject, Key: uuid.New().String()}
		owned    = uuid.New()
		snapshot = uuid.New()
		both     = uuid.New()
		dangling = uuid.New()
		fallback = uuid.New()
		orphan   = uuid.New()
		logID    = resource.ID{Type: resource.TypeLog, Key: uuid.New().String()}
	)

	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		entries := []vis{
			{Key: owned},
			{Key: snapshot},
			{Key: both},
			{Key: dangling},
			{Key: fallback},
			{Key: orphan},
		}
		Expect(gorp.NewCreate[uuid.UUID, vis]().Entries(&entries).Exec(ctx, db)).
			To(Succeed())
		rels := []ontology.Relationship{
			parentOf(projID, visID(owned)),
			parentOf(rngID, visID(snapshot)),
			parentOf(rngID, visID(both)),
			parentOf(projID, visID(both)),
			parentOf(missing, visID(dangling)),
			parentOf(missing, visID(fallback)),
			parentOf(rngID, visID(fallback)),
			parentOf(visID(orphan), logID),
			parentOf(projID, logID),
		}
		Expect(gorp.NewCreate[string, ontology.Relationship]().
			Entries(&rels).
			Exec(ctx, db)).To(Succeed())
		resources := []ontology.Resource{
			{ID: projID},
			{ID: rngID},
			{ID: visID(owned)},
			{ID: visID(orphan)},
		}
		Expect(gorp.NewCreate[string, ontology.Resource]().
			Entries(&resources).
			Exec(ctx, db)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "vis",
			Migrations: []gorp.Migration{
				v3.NewOwnersMigration(v3.OwnersConfig[uuid.UUID, vis]{
					Name:  "owners",
					Type:  resource.TypeSchematic,
					Owned: func(v vis) bool { return v.Project != nil || v.Range != nil },
					SetOwner: func(v *vis, p, r *uuid.UUID) {
						v.Project, v.Range = p, r
					},
				}),
			},
		})).To(Succeed())
	})

	byKey := func(ctx context.Context) map[uuid.UUID]vis {
		GinkgoHelper()
		out := make(map[uuid.UUID]vis)
		for _, v := range retrieveAll[uuid.UUID, vis](ctx, db) {
			out[v.Key] = v
		}
		return out
	}

	DescribeTable("Should set the owner from the parent edge", func(
		ctx SpecContext,
		key *uuid.UUID,
		project *uuid.UUID,
		rng *uuid.UUID,
	) {
		Expect(byKey(ctx)).To(HaveKeyWithValue(*key, vis{
			Key:     *key,
			Project: project,
			Range:   rng,
		}))
	},
		Entry("a visualization under a project", &owned, &proj, nil),
		Entry("a visualization under a range", &snapshot, nil, &rng),
		Entry("a visualization under both a project and a range", &both, &proj, nil),
		Entry("a visualization under a missing project and a range", &fallback, nil,
			&rng),
	)

	It("Should delete a visualization with no stored owner", func(ctx SpecContext) {
		Expect(byKey(ctx)).ToNot(HaveKey(orphan))
		Expect(byKey(ctx)).ToNot(HaveKey(dangling))
	})

	It("Should delete the ontology resource of a deleted visualization", func(
		ctx SpecContext,
	) {
		Expect(retrieveAll[string, ontology.Resource](ctx, db)).To(ConsistOf(
			ontology.Resource{ID: projID},
			ontology.Resource{ID: rngID},
			ontology.Resource{ID: visID(owned)},
		))
	})

	It(
		"Should delete the converted edges and the edges of deleted visualizations",
		func(ctx SpecContext) {
			Expect(retrieveAll[string, ontology.Relationship](ctx, db)).To(ConsistOf(
				parentOf(projID, logID),
			))
		},
	)
})
