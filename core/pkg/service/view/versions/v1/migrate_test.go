// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	rangerv1 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v1"
	statusv1 "github.com/synnaxlabs/synnax/pkg/service/status/versions/v1"
	v0 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v1"
	"github.com/synnaxlabs/x/encoding/msgpack"
	filter "github.com/synnaxlabs/x/filter/versions/v0"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	var (
		db  *gorp.DB
		otg *ontology.Ontology
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.View]{DB: db}))
	})

	create := func(ctx SpecContext, views ...v0.View) {
		GinkgoHelper()
		Expect(gorp.NewCreate[v0.Key, v0.View]().Entries(&views).Exec(ctx, db)).
			To(Succeed())
		ids := make([]ontology.ID, len(views))
		for i, v := range views {
			ids[i] = v.OntologyID()
		}
		w := otg.NewWriter(db)
		Expect(w.DefineResources(ctx, ontology.RootID)).To(Succeed())
		Expect(w.DefineResources(ctx, ids...)).To(Succeed())
		Expect(w.DefineRelationships(
			ctx,
			ontology.RootID,
			ontology.RelationshipTypeParentOf,
			ids...,
		)).To(Succeed())
	}

	run := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "View",
			Migrations: []gorp.Migration{v1.NewMigration(otg)},
		})).To(Succeed())
	}

	retrieve := func(ctx SpecContext, key uuid.UUID) (v1.View, error) {
		var v v1.View
		err := gorp.NewRetrieve[v1.Key, v1.View]().
			Where(gorp.MatchKeys[v1.Key, v1.View](key)).
			Entry(&v).Exec(ctx, db)
		return v, err
	}

	It("Should turn the query of range and status views into a where", func(
		ctx SpecContext,
	) {
		label := uuid.New()
		ranges := v0.View{
			Key:   uuid.New(),
			Name:  "Ranges",
			Type:  "range",
			Query: msgpack.EncodedJSON{"has_labels": []any{label.String()}},
		}
		statuses := v0.View{
			Key:  uuid.New(),
			Name: "Statuses",
			Type: "status",
			Query: msgpack.EncodedJSON{
				"has_labels": []any{label.String()},
				"variants":   []any{"error", "warning"},
			},
		}
		empty := v0.View{Key: uuid.New(), Name: "Empty", Type: "range"}
		emptyStatuses := v0.View{
			Key:   uuid.New(),
			Name:  "Empty statuses",
			Type:  "status",
			Query: msgpack.EncodedJSON{"has_labels": []any{}, "variants": []any{}},
		}
		create(ctx, ranges, statuses, empty, emptyStatuses)
		run(ctx)
		Expect(retrieve(ctx, ranges.Key)).To(Equal(v1.View{Variant: v1.RangeView{
			Base:  v1.Base{Key: ranges.Key, Name: "Ranges"},
			Where: &rangerv1.Where{Labels: []uuid.UUID{label}},
		}}))
		Expect(retrieve(ctx, statuses.Key)).To(Equal(v1.View{Variant: v1.StatusView{
			Base: v1.Base{Key: statuses.Key, Name: "Statuses"},
			Where: &statusv1.Where{
				Labels: []uuid.UUID{label},
				Variant: &filter.Set[statusv1.Variant]{In: []statusv1.Variant{
					statusv1.VariantError,
					statusv1.VariantWarning,
				}},
			},
		}}))
		Expect(retrieve(ctx, empty.Key)).To(Equal(v1.View{Variant: v1.RangeView{
			Base: v1.Base{Key: empty.Key, Name: "Empty"},
		}}))
		Expect(retrieve(ctx, emptyStatuses.Key)).To(Equal(v1.View{
			Variant: v1.StatusView{
				Base: v1.Base{Key: emptyStatuses.Key, Name: "Empty statuses"},
			},
		}))
	})

	It(
		"Should delete views it cannot hold and their ontology resources",
		func(ctx SpecContext) {
			kept := v0.View{Key: uuid.New(), Name: "Ranges", Type: "range"}
			dropped := v0.View{
				Key:   uuid.New(),
				Name:  "Plot",
				Type:  "lineplot",
				Query: msgpack.EncodedJSON{"channels": []any{"a"}},
			}
			undecodable := v0.View{
				Key:   uuid.New(),
				Name:  "Team",
				Type:  "range",
				Query: msgpack.EncodedJSON{"has_labels": []any{"team-a"}},
			}
			badVariant := v0.View{
				Key:   uuid.New(),
				Name:  "Bogus",
				Type:  "status",
				Query: msgpack.EncodedJSON{"variants": []any{"bogus"}},
			}
			undecodableStatus := v0.View{
				Key:   uuid.New(),
				Name:  "Scalar",
				Type:  "status",
				Query: msgpack.EncodedJSON{"variants": "error"},
			}
			create(ctx, kept, dropped, undecodable, badVariant, undecodableStatus)
			run(ctx)
			Expect(retrieve(ctx, dropped.Key)).Error().
				To(MatchError(query.ErrNotFound))
			Expect(retrieve(ctx, undecodable.Key)).Error().
				To(MatchError(query.ErrNotFound))
			Expect(retrieve(ctx, badVariant.Key)).Error().
				To(MatchError(query.ErrNotFound))
			Expect(retrieve(ctx, undecodableStatus.Key)).Error().
				To(MatchError(query.ErrNotFound))
			Expect(retrieve(ctx, kept.Key)).Error().ToNot(HaveOccurred())
			Expect(gorp.NewRetrieve[string, ontology.Resource]().
				Where(gorp.MatchKeys[string, ontology.Resource](
					dropped.OntologyID().String(),
				)).
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
			var rels []ontology.Relationship
			Expect(gorp.NewRetrieve[string, ontology.Relationship]().
				Entries(&rels).Exec(ctx, db)).To(Succeed())
			Expect(rels).To(ConsistOf(ontology.Relationship{
				From: ontology.RootID,
				Type: ontology.RelationshipTypeParentOf,
				To:   kept.OntologyID(),
			}))
		},
	)
})
