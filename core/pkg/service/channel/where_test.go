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
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

// record is a channel as the client sees it, since the TypeScript matches reads the
// same records.
type record struct {
	Key         channel.Key    `json:"key"`
	Name        string         `json:"name"`
	Leaseholder node.Key       `json:"leaseholder"`
	DataType    telem.DataType `json:"data_type"`
	IsIndex     bool           `json:"is_index"`
	Index       channel.Key    `json:"index"`
	Virtual     bool           `json:"virtual"`
	Internal    bool           `json:"internal"`
	Expression  string         `json:"expression"`
}

var _ = Describe("Where", Ordered, func() {
	var (
		v      svctestutil.Vectors[record, channel.Where]
		chSvc  *channel.Service
		byName = make(map[string]string)
	)
	v = MustSucceed(svctestutil.ReadVectors[record, channel.Where]("channels"))

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		// An Aspen database leases each record to its leaseholder, and the records sit
		// on nodes that the mock cluster does not have.
		cfg := serviceConfig(
			ctx,
			testutil.NewNode(ctx),
			DeferClose(gorp.Wrap(memkv.New())),
		)
		chSvc = MustOpen(channel.OpenService(ctx, cfg))
		chs := make([]channel.Channel, 0, len(v.Records))
		for alias, r := range v.Records {
			byName[r.Name] = alias
			chs = append(chs, channel.Channel{
				Name:        r.Name,
				Leaseholder: r.Leaseholder,
				DataType:    r.DataType,
				IsIndex:     r.IsIndex,
				LocalKey:    r.Key.LocalKey(),
				LocalIndex:  r.Index.LocalKey(),
				Virtual:     r.Virtual,
				Internal:    r.Internal,
				Expression:  r.Expression,
			})
		}
		// The writer assigns its own keys, so the records go straight to the table.
		Expect(gorp.NewCreate[channel.Key, channel.Channel]().
			Entries(&chs).
			Exec(ctx, cfg.DB)).To(Succeed())
	})

	DescribeTable(
		"Should match the case",
		func(ctx SpecContext, c svctestutil.Case[channel.Where]) {
			f := MustSucceed(channel.MatchWhere(c.Where))
			var res []channel.Channel
			Expect(chSvc.NewRetrieve().
				SkipMissingKeys().
				Where(f).
				Entries(&res).
				Exec(ctx, nil)).To(Succeed())
			Expect(lo.Map(res, func(ch channel.Channel, _ int) string {
				return byName[ch.Name]
			})).To(ConsistOf(c.Expected))
		},
		lo.Map(v.Cases, func(c svctestutil.Case[channel.Where], _ int) TableEntry {
			return Entry(c.Name, c)
		}),
	)

	It("Should return a path error for an invalid regex", func() {
		var w channel.Where
		Expect(json.Unmarshal(
			[]byte(`{"or":[{"key":{"in":[]}},{"name":{"regex":"(a"}}]}`),
			&w,
		)).To(Succeed())
		Expect(channel.MatchWhere(w)).Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError(
				"or.1.name.regex: error parsing regexp: missing closing ): `(a`: validation error",
			),
		))
	})
})
