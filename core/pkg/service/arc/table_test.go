// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("OpenTable", func() {
	It("Should open a table that stores Arcs", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(arc.OpenTable(ctx, arc.TableConfig{DB: db}))
		a := newAutomation(arc.Base{
			Key:    uuid.New(),
			Name:   "stored",
			Source: arc.Source{Variant: arc.TextSource{}},
		})
		Expect(gorp.NewCreate[arc.Key, arc.Arc]().Entry(&a).Exec(ctx, db)).
			To(Succeed())
		var stored arc.Arc
		Expect(gorp.NewRetrieve[arc.Key, arc.Arc]().
			Where(gorp.MatchKeys[arc.Key, arc.Arc](a.Base().Key)).
			Entry(&stored).
			Exec(ctx, db)).To(Succeed())
		Expect(stored.Base().Name).To(Equal("stored"))
	})

	It("Should reject a config with no DB", func(ctx SpecContext) {
		Expect(arc.OpenTable(ctx, arc.TableConfig{})).Error().
			To(MatchError(ContainSubstring("db: must be non-nil")))
	})
})
