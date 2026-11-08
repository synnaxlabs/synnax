// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Resources", func() {
	var (
		db  *gorp.DB
		svc *status.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
		}))
		svc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
	})
	set := func(ctx SpecContext, key, name string) status.Status[any] {
		GinkgoHelper()
		s := status.Status[any]{
			Key:     key,
			Name:    name,
			Variant: status.VariantInfo,
			Time:    telem.Now(),
		}
		Expect(svc.NewWriter(nil).Set(ctx, &s)).To(Succeed())
		return s
	}

	It("Should name each status by its name", func(ctx SpecContext) {
		s := set(ctx, "resource-status", "Pump status")
		Expect(svc.Resources().Retrieve(ctx, db, []resource.ID{s.ResourceID()})).
			To(Equal([]resource.Resource{{ID: s.ResourceID(), Name: s.Name}}))
	})

	It("Should leave out an ID that names no status", func(ctx SpecContext) {
		s := set(ctx, "resource-status", "Pump status")
		missing := status.ResourceID("missing-status")
		Expect(svc.Resources().Retrieve(
			ctx,
			db,
			[]resource.ID{missing, s.ResourceID()},
		)).To(Equal([]resource.Resource{{ID: s.ResourceID(), Name: s.Name}}))
	})

	It("Should list each status by its name", func(ctx SpecContext) {
		s := set(ctx, "resource-status", "Pump status")
		Expect(svc.Resources().List(ctx, db, 10, nil)).
			To(Equal([]resource.Resource{{ID: s.ResourceID(), Name: s.Name}}))
	})
})
