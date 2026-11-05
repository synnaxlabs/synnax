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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v0 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It("Should keep every field of a stored channel with no retention", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Channel]{DB: db}))
		old := v0.Channel{
			Name:        "pt_avg",
			Leaseholder: 1,
			DataType:    telem.Float32T,
			LocalKey:    2,
			LocalIndex:  3,
			Virtual:     true,
			Concurrency: control.ConcurrencyShared,
			Internal:    true,
			Operations:  []v0.Operation{{Type: v0.OperationTypeAvg}},
			Expression:  "return pt",
		}
		Expect(gorp.NewCreate[v0.Key, v0.Channel]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Channel",
			Migrations: []gorp.Migration{v1.Migration},
		})).To(Succeed())
		var got v1.Channel
		Expect(gorp.NewRetrieve[v1.Key, v1.Channel]().
			Where(gorp.MatchKeys[v1.Key, v1.Channel](old.GorpKey())).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got).To(Equal(v1.Channel{
			Name:        "pt_avg",
			Leaseholder: 1,
			DataType:    telem.Float32T,
			LocalKey:    2,
			LocalIndex:  3,
			Virtual:     true,
			Concurrency: control.ConcurrencyShared,
			Internal:    true,
			Operations:  []v1.Operation{{Type: v1.OperationTypeAvg}},
			Expression:  "return pt",
		}))
	})
})
