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
	v3 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v3"
	ranger "github.com/synnaxlabs/synnax/pkg/service/ranger/versions"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	color "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

type vis struct {
	Key     uuid.UUID
	Name    string
	Project *uuid.UUID
	Range   *uuid.UUID
}

func (v vis) GorpKey() uuid.UUID { return v.Key }

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
		logs     *observer.ObservedLogs
		proj     = uuid.New()
		rng      = uuid.New()
		projID   = resource.ID{Type: resource.TypeProject, Key: proj.String()}
		rngID    = resource.ID{Type: resource.TypeRange, Key: rng.String()}
		missing  = resource.ID{Type: resource.TypeProject, Key: uuid.New().String()}
		gone     = resource.ID{Type: resource.TypeRange, Key: uuid.New().String()}
		owned    = uuid.New()
		snapshot = uuid.New()
		both     = uuid.New()
		dangling = uuid.New()
		fallback = uuid.New()
		lost     = uuid.New()
		orphan   = uuid.New()
		logID    = resource.ID{Type: resource.TypeLog, Key: uuid.New().String()}
	)

	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		entries := []vis{
			{Key: owned},
			{Key: snapshot},
			{Key: both},
			{Key: dangling, Name: "Dangling"},
			{Key: fallback},
			{Key: lost, Name: "Lost"},
			{Key: orphan, Name: "Orphan"},
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
			parentOf(gone, visID(lost)),
			parentOf(visID(orphan), logID),
			parentOf(projID, logID),
		}
		Expect(gorp.NewCreate[string, ontology.Relationship]().
			Entries(&rels).
			Exec(ctx, db)).To(Succeed())
		resources := []ontology.Resource{{ID: visID(owned)}, {ID: visID(orphan)}}
		Expect(gorp.NewCreate[string, ontology.Resource]().
			Entries(&resources).
			Exec(ctx, db)).To(Succeed())
		Expect(gorp.NewCreate[v3.Key, v3.Project]().
			Entry(&v3.Project{Key: proj, Name: "Project"}).
			Exec(ctx, db)).To(Succeed())
		Expect(gorp.NewCreate[ranger.Key, ranger.Range]().
			Entry(&ranger.Range{Key: rng, Name: "Range", Color: &color.Color{R: 1, A: 128}}).
			Exec(ctx, db)).To(Succeed())
		var core zapcore.Core
		core, logs = observer.New(zapcore.WarnLevel)
		logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
			DB:              db,
			Namespace:       "vis",
			Migrations: []gorp.Migration{
				v3.NewOwnersMigration(v3.OwnersConfig[uuid.UUID, vis]{
					Name:   "owners",
					Type:   resource.TypeSchematic,
					Owned:  func(v vis) bool { return v.Project != nil || v.Range != nil },
					NameOf: func(v vis) string { return v.Name },
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
		Expect(byKey(ctx)).ToNot(HaveKey(lost))
	})

	It("Should log a warning for each deleted visualization", func(ctx SpecContext) {
		deleted := func(k uuid.UUID, name string) map[string]any {
			return map[string]any{
				"type": string(resource.TypeSchematic),
				"key":  k.String(),
				"name": name,
			}
		}
		var got []map[string]any
		for _, e := range logs.FilterMessage(
			"deleted a visualization with no project or range",
		).All() {
			got = append(got, e.ContextMap())
		}
		Expect(got).To(ConsistOf(
			deleted(orphan, "Orphan"),
			deleted(dangling, "Dangling"),
			deleted(lost, "Lost"),
		))
	})

	It("Should delete the ontology resource of a deleted visualization", func(
		ctx SpecContext,
	) {
		Expect(retrieveAll[string, ontology.Resource](ctx, db)).To(ConsistOf(
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
