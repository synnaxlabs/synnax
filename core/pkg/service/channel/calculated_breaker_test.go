// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

const unresolved = "only the channel API can match the calculated line"

var _ = Describe("Calculated line breaker", Ordered, func() {
	var (
		svc *channel.Service
		db  *gorp.DB
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		cfg, _ := serviceConfig(ctx, testutil.NewNode(ctx), db)
		svc = MustOpen(channel.OpenService(ctx, cfg))
	})

	shapes := map[string]channel.Where{
		"top level": {Calculated: new(true)},
		"not of not": {
			Not: &channel.Where{Not: &channel.Where{Calculated: new(false)}},
		},
		"or of lines": {
			Or: []channel.Where{
				{Internal: &filter.Bool{Eq: new(true)}},
				{Calculated: new(true)},
			},
		},
		"and with an empty key set": {And: []channel.Where{
			{Key: &filter.Set[channel.Key]{In: []channel.Key{}}},
			{Calculated: new(true)},
		}},
		"beside an empty key set": {
			Key:        &filter.Set[channel.Key]{In: []channel.Key{}},
			Calculated: new(true),
		},
		"empty or beside it": {Or: []channel.Where{}, Calculated: new(true)},
	}
	for name, w := range shapes {
		run := func(ctx context.Context, q channel.Retrieve) []error {
			var (
				res []channel.Channel
				ch  channel.Channel
			)
			_, countErr := q.Count(ctx, db)
			_, existsErr := q.Exists(ctx, db)
			return []error{
				q.Entries(&res).Exec(ctx, db),
				q.Entry(&ch).Exec(ctx, db),
				q.Limit(1).Entries(&res).Exec(ctx, db),
				q.OrderBy(channel.OrderByName(gorp.DirectionAsc)).
					Entries(&res).
					Exec(ctx, db),
				q.Search("x").Entries(&res).Exec(ctx, db),
				countErr,
				existsErr,
			}
		}
		It("Should fail every read on an empty table, "+name, func(ctx SpecContext) {
			f := MustSucceed(channel.MatchWhere(w))
			for i, err := range run(ctx, svc.NewRetrieve().Where(f)) {
				Expect(err).To(MatchError(ContainSubstring(unresolved)), "read %d", i)
			}
		})
	}
})
