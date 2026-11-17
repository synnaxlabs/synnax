// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package color_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	xcolor "github.com/synnaxlabs/x/color"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("OpenService", func() {
	It("Should refuse a config with no DB", func(ctx SpecContext) {
		Expect(color.OpenService(ctx, color.ServiceConfig{})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("db: must be non-nil")),
		))
	})
	It("Should refuse a config with no group service", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		Expect(
			color.OpenService(ctx, color.ServiceConfig{DB: db}),
		).Error().
			To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("group: must be non-nil")),
			))
	})
})

var _ = Describe("Changes", func() {
	It("Should return the named color that a commit creates", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		groups := MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		svc := MustOpen(
			color.OpenService(ctx, color.ServiceConfig{DB: db, Group: groups}),
		)
		var names []string
		DeferCleanup(
			db.Observe().OnChange(func(ctx context.Context, c gorp.Commit) {
				for ch := range svc.Changes(c) {
					names = append(names, ch.Value.Name)
				}
			}),
		)
		n := color.Named{Name: "Breaker", Light: MustSucceed(xcolor.Parse("#ff0000"))}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &n)
		})).To(Succeed())
		Expect(names).To(Equal([]string{"Breaker"}))
	})
})
