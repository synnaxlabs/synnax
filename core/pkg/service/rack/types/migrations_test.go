// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	rackv0 "github.com/synnaxlabs/synnax/pkg/service/rack/types/v0"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migration", func() {
	var (
		db        *gorp.DB
		otg       *ontology.Ontology
		g         *group.Service
		labelSvc  *label.Service
		stat      *status.Service
		searchIdx *search.Index
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx = MustOpen(search.OpenIndex())
		g = MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		labelSvc = MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Search:   searchIdx,
		}))
		stat = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    g,
			Label:    labelSvc,
			Search:   searchIdx,
		}))
	})

	openService := func(ctx context.Context) *rack.Service {
		return MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:           db,
			Ontology:     otg,
			Group:        g,
			HostProvider: mock.NewStaticHostProvider(1),
			Status:       stat,
			Search:       searchIdx,
		}))
	}

	It("Should create unknown statuses for racks missing them", func(ctx SpecContext) {
		r := rackv0.Rack{
			Key:  rackv0.Key(rack.NewKey(1, 50)),
			Name: "rack without status",
		}
		Expect(gorp.NewCreate[rackv0.Key, rackv0.Rack]().
			Entry(&r).
			Exec(ctx, db)).To(Succeed())

		openService(ctx)

		var restoredStatus rack.Status
		Expect(status.NewRetrieve[rack.StatusDetails](stat).
			Where(status.MatchKeys[rack.StatusDetails](rack.Key(r.Key).OntologyID().String())).
			Entry(&restoredStatus).
			Exec(ctx, nil)).To(Succeed())
		Expect(restoredStatus.Variant).To(Equal(status.VariantWarning))
		Expect(restoredStatus.Message).To(Equal("Status unknown"))
		Expect(restoredStatus.Details.Rack).To(Equal(rack.Key(r.Key)))
	})

	It("Should correctly migrate a v1 rack to a v2 rack", func(ctx SpecContext) {
		v1EmbeddedRack := rackv0.Rack{
			Key:  65538,
			Name: "sy_node_1_rack",
		}
		Expect(gorp.NewCreate[rackv0.Key, rackv0.Rack]().
			Entry(&v1EmbeddedRack).
			Exec(ctx, db)).To(Succeed())

		svc := openService(ctx)
		Expect(svc.EmbeddedKey).To(Equal(rack.Key(65538)))
		var embeddedRack rack.Rack
		Expect(svc.NewRetrieve().
			Where(rack.MatchKeys(svc.EmbeddedKey)).
			Entry(&embeddedRack).
			Exec(ctx, db)).To(Succeed())
		Expect(embeddedRack.Embedded).To(BeTrue())
		Expect(embeddedRack.Name).To(Equal("Node 1 Embedded Driver"))
		count := MustSucceed(svc.NewRetrieve().Count(ctx, db))
		Expect(count).To(Equal(1))
	})

	It("Should not match an embedded rack with a mismatched name", func(ctx SpecContext) {
		mismatchedRack := rackv0.Rack{
			Key:      65538,
			Name:     "Some Other Embedded Rack",
			Embedded: true,
		}
		Expect(gorp.NewCreate[rackv0.Key, rackv0.Rack]().
			Entry(&mismatchedRack).
			Exec(ctx, db)).To(Succeed())

		svc := openService(ctx)
		Expect(svc.EmbeddedKey).ToNot(BeEquivalentTo(mismatchedRack.Key))

		var embeddedRack rack.Rack
		Expect(svc.NewRetrieve().
			Where(rack.MatchKeys(svc.EmbeddedKey)).
			Entry(&embeddedRack).
			Exec(ctx, db)).To(Succeed())
		Expect(embeddedRack.Embedded).To(BeTrue())
		Expect(embeddedRack.Name).To(Equal("Node 1 Embedded Driver"))

		count := MustSucceed(svc.NewRetrieve().Count(ctx, db))
		Expect(count).To(Equal(2))
	})

	It("Should reuse an existing v2 embedded rack with the correct name", func(ctx SpecContext) {
		existingRack := rackv0.Rack{
			Key:      65538,
			Name:     "Node 1 Embedded Driver",
			Embedded: true,
		}
		Expect(gorp.NewCreate[rackv0.Key, rackv0.Rack]().
			Entry(&existingRack).
			Exec(ctx, db)).To(Succeed())

		svc := openService(ctx)
		Expect(svc.EmbeddedKey).To(Equal(rack.Key(existingRack.Key)))

		var embeddedRack rack.Rack
		Expect(svc.NewRetrieve().
			Where(rack.MatchKeys(svc.EmbeddedKey)).
			Entry(&embeddedRack).
			Exec(ctx, db)).To(Succeed())
		Expect(embeddedRack.Embedded).To(BeTrue())
		Expect(embeddedRack.Name).To(Equal("Node 1 Embedded Driver"))

		count := MustSucceed(svc.NewRetrieve().Count(ctx, db))
		Expect(count).To(Equal(1))
	})
})
