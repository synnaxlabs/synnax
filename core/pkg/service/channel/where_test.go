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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// The case table is shared with the TypeScript matches, so that the Core and the
// client cache agree on every filter.
const vectorsPath = "../../../../x/testdata/filter/vectors.json"

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

type whereCase struct {
	Name     string        `json:"name"`
	Where    channel.Where `json:"where"`
	Expected []string      `json:"expected"`
}

type vectors struct {
	Channels struct {
		Records map[string]record `json:"records"`
		Cases   []whereCase       `json:"cases"`
	} `json:"channels"`
}

var _ = Describe("Where", Ordered, func() {
	var (
		v      vectors
		chSvc  *channel.Service
		byName = make(map[string]string)
	)
	Expect(json.Unmarshal(MustSucceed(os.ReadFile(vectorsPath)), &v)).To(Succeed())

	BeforeAll(func(ctx SpecContext) {
		cfg := serviceConfig(ctx, mock.NewNode(ctx))
		// An Aspen database leases each record to its leaseholder, and the records sit
		// on nodes that the mock cluster does not have.
		cfg.DB = DeferClose(gorp.Wrap(memkv.New()))
		chSvc = DeferClose(MustOpen(channel.OpenService(ctx, cfg)))
		chs := make([]channel.Channel, 0, len(v.Channels.Records))
		for alias, r := range v.Channels.Records {
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

	for _, c := range v.Channels.Cases {
		It("Should match the case: "+c.Name, func(ctx SpecContext) {
			f := MustSucceed(channel.MatchWhere(c.Where))
			var res []channel.Channel
			// The API skips not-found, since a missing key is a non-match.
			Expect(errors.Skip(
				chSvc.NewRetrieve().Where(f).Entries(&res).Exec(ctx, nil),
				query.ErrNotFound,
			)).To(Succeed())
			Expect(lo.Map(res, func(ch channel.Channel, _ int) string {
				return byName[ch.Name]
			})).To(ConsistOf(c.Expected))
		})
	}

	It("Should return a path error for an invalid regex", func() {
		var w channel.Where
		Expect(json.Unmarshal(
			[]byte(`{"or":[{"key":{"in":[]}},{"name":{"regex":"(a"}}]}`),
			&w,
		)).To(Succeed())
		_, err := channel.MatchWhere(w)
		var pathErr validate.PathError
		Expect(err).To(BeAssignableToTypeOf(pathErr))
		Expect(err).To(MatchError(ContainSubstring(
			"or.1.name.regex: error parsing regexp: missing closing ): `(a`",
		)))
	})
})
