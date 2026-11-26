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
	"github.com/synnaxlabs/gorp"
	v1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("NameRuleMigration breaker", func() {
	It("Should give a legal name to a folder whose 30 marks end in a Hangul vowel",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			key := uuid.MustParse("00000000-0000-4000-8000-000000000001")
			Expect(gorp.WrapWriter[v1.Key, v1.Group](db).Set(ctx, v1.Group{
				Key:   key,
				Name:  "Stand" + strings.Repeat("\u0316", 30) + "\u1161",
				Scope: resource.TypeChannel,
			})).To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "group",
				Migrations: []gorp.Migration{v1.NameRuleMigration},
			})).To(Succeed())
			var g v1.Group
			Expect(gorp.NewRetrieve[v1.Key, v1.Group]().
				Where(gorp.MatchKeys[v1.Key, v1.Group](key)).
				Entry(&g).
				Exec(ctx, db)).To(Succeed())
			Expect(validate.IsPathSegment(g.Name)).To(BeTrue(), "new name %+q", g.Name)
		})
})
