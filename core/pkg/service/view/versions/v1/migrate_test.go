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
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	v0 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v1"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var rootID = resource.ID{Type: resource.TypeBuiltin, Key: "root"}

var _ = Describe("Migration", func() {
	var db *gorp.DB
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.View]{DB: db}))
	})

	create := func(ctx SpecContext, views ...v0.View) {
		GinkgoHelper()
		Expect(gorp.NewCreate[v0.Key, v0.View]().Entries(&views).Exec(ctx, db)).
			To(Succeed())
		ids := make([]resource.ID, len(views))
		for i, v := range views {
			ids[i] = v.ResourceID()
		}
		Expect(ontology.DefineResources(ctx, db, append(ids, rootID)...)).
			To(Succeed())
		rels := make([]ontology.Relationship, len(ids))
		for i, id := range ids {
			rels[i] = ontology.Relationship{
				From: rootID,
				Type: ontology.RelationshipTypeParentOf,
				To:   id,
			}
		}
		Expect(gorp.WrapWriter[string, ontology.Relationship](db).Set(ctx, rels...)).
			To(Succeed())
	}

	run := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "View",
			Migrations: []gorp.Migration{v1.Migration},
		})).To(Succeed())
	}

	retrieve := func(ctx SpecContext, key uuid.UUID) (v1.View, error) {
		var v v1.View
		err := gorp.NewRetrieve[v1.Key, v1.View]().
			Where(gorp.MatchKeys[v1.Key, v1.View](key)).
			Entry(&v).Exec(ctx, db)
		return v, err
	}

	It("Should type the query of range and status views", func(ctx SpecContext) {
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
		create(ctx, ranges, statuses, empty)
		run(ctx)
		Expect(retrieve(ctx, ranges.Key)).To(Equal(v1.View{Variant: v1.RangeView{
			Base:  v1.Base{Key: ranges.Key, Name: "Ranges"},
			Query: v1.RangeQuery{HasLabels: []uuid.UUID{label}},
		}}))
		Expect(retrieve(ctx, statuses.Key)).To(Equal(v1.View{Variant: v1.StatusView{
			Base: v1.Base{Key: statuses.Key, Name: "Statuses"},
			Query: v1.StatusQuery{
				HasLabels: []uuid.UUID{label},
				Variants:  []status.Variant{status.VariantError, status.VariantWarning},
			},
		}}))
		Expect(retrieve(ctx, empty.Key)).To(Equal(v1.View{Variant: v1.RangeView{
			Base: v1.Base{Key: empty.Key, Name: "Empty"},
		}}))
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
					dropped.ResourceID().String(),
				)).
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
			var rels []ontology.Relationship
			Expect(gorp.NewRetrieve[string, ontology.Relationship]().
				Entries(&rels).Exec(ctx, db)).To(Succeed())
			Expect(rels).To(ConsistOf(ontology.Relationship{
				From: rootID,
				Type: ontology.RelationshipTypeParentOf,
				To:   kept.ResourceID(),
			}))
		},
	)
})
