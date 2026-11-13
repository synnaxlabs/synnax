// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/color"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	var (
		db  *gorp.DB
		svc *label.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		svc = MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB: db,
		}))
	})

	It("Should name each label by its name", func(ctx SpecContext) {
		l := label.Label{Name: "Nominal", Color: MustSucceed(color.Parse("#00FF00"))}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &l)
		})).To(Succeed())
		Expect(svc.Resources().Retrieve(ctx, db, []resource.ID{l.ResourceID()})).
			To(Equal([]resource.Resource{{ID: l.ResourceID(), Name: l.Name}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			db,
			[]resource.ID{{Type: resource.TypeLabel, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid label key "pump"`)),
		))
	})

	It("Should leave out an ID that names no label", func(ctx SpecContext) {
		l := label.Label{Name: "Nominal", Color: MustSucceed(color.Parse("#00FF00"))}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &l)
		})).To(Succeed())
		missing := resource.ID{Type: resource.TypeLabel, Key: uuid.New().String()}
		Expect(svc.Resources().Retrieve(
			ctx,
			db,
			[]resource.ID{missing, l.ResourceID()},
		)).To(Equal([]resource.Resource{{ID: l.ResourceID(), Name: l.Name}}))
	})

	It("Should list each label by its name", func(ctx SpecContext) {
		l := label.Label{Name: "Nominal", Color: MustSucceed(color.Parse("#00FF00"))}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &l)
		})).To(Succeed())
		Expect(svc.Resources().List(ctx, db, 10, nil)).
			To(Equal([]resource.Resource{{ID: l.ResourceID(), Name: l.Name}}))
	})
})
