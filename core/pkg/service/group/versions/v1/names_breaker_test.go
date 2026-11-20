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
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	v1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("NameRuleMigration breakers", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })
	create := func(ctx SpecContext, groups ...v1.Group) {
		GinkgoHelper()
		Expect(gorp.NewCreate[v1.Key, v1.Group]().
			Entries(&groups).
			Exec(ctx, db)).To(Succeed())
	}
	run := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return v1.NameRuleMigration.Run(ctx, tx, alamos.Instrumentation{})
		})).To(Succeed())
	}
	stored := func(ctx SpecContext) map[v1.Key]v1.Group {
		GinkgoHelper()
		var all []v1.Group
		Expect(gorp.NewRetrieve[v1.Key, v1.Group]().
			Entries(&all).
			Exec(ctx, db)).To(Succeed())
		out := make(map[v1.Key]v1.Group, len(all))
		for _, g := range all {
			out[g.Key] = g
		}
		return out
	}
	legal := func(name string) bool {
		return validate.IsPathSegment(name) && validate.IsNormalName(name)
	}

	It("Should rename a long run of combining marks to a legal name", func(
		ctx SpecContext,
	) {
		g := v1.Group{
			Key:   uuid.New(),
			Name:  "a" + strings.Repeat("\u0301", 31),
			Scope: resource.TypeChannel,
		}
		create(ctx, g)
		run(ctx)
		name := stored(ctx)[g.Key].Name
		Expect(legal(name)).To(BeTrue(), "%+q", name)
	})
})
