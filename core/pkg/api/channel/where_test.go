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
	"fmt"
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/testutil/vectors"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

type vectorRecord struct {
	Key         channel.Key    `json:"key"`
	Name        string         `json:"name"`
	Leaseholder node.Key       `json:"leaseholder"`
	DataType    telem.DataType `json:"data_type"`
	IsIndex     bool           `json:"is_index"`
	Index       channel.Key    `json:"index"`
	Virtual     bool           `json:"virtual"`
	Internal    bool           `json:"internal"`
	Expression  string         `json:"expression"`
	Calculated  bool           `json:"calculated"`
}

func usesVirtual(w channel.Where) bool {
	return w.Virtual != nil ||
		slices.ContainsFunc(w.And, usesVirtual) ||
		slices.ContainsFunc(w.Or, usesVirtual) ||
		(w.Not != nil && usesVirtual(*w.Not))
}

// The channel service counts every virtual channel, so the filter vectors whose cases
// hold the API's meaning of virtual run here.
var _ = Describe("Filter vectors", Ordered, func() {
	var (
		v       = MustSucceed(vectors.Read[vectorRecord, channel.Where]("channels"))
		u       user.User
		keys    channel.Keys
		aliases = make(map[channel.Key]string)
	)

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		u = MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			rbacSvc,
			userSvc,
			user.User{Name: "vectors-" + uuid.New().String()},
		))
		grantOn(
			ctx,
			u.ResourceID(),
			access.AllActions,
			resource.ID{Type: resource.TypeChannel},
			resource.ID{Type: resource.TypeCalculation},
			resource.ID{Type: resource.TypeFramer},
		)
		create := func(alias string, ch apichannel.Channel) apichannel.Channel {
			GinkgoHelper()
			created := MustSucceed(createChannels(
				AuthedCtx(ctx, u),
				apichannel.CreateRequest{Channels: []apichannel.Channel{ch}},
			)).Channels[0]
			aliases[created.Key] = alias
			keys = append(keys, created.Key)
			return created
		}
		// The index, then the data channels, so a calculation can read one of them.
		var index, data apichannel.Channel
		for alias, r := range v.Records {
			if r.IsIndex {
				index = create(alias, apichannel.Channel{
					Name:     UniqueChannelName(),
					DataType: r.DataType,
					IsIndex:  true,
				})
			}
		}
		for alias, r := range v.Records {
			if r.IsIndex || r.Calculated || r.Internal {
				continue
			}
			ch := apichannel.Channel{
				Name:     UniqueChannelName(),
				DataType: r.DataType,
				Virtual:  r.Virtual,
			}
			if !r.Virtual {
				ch.Index = index.Key
			}
			if created := create(alias, ch); !r.Virtual {
				data = created
			}
		}
		for alias, r := range v.Records {
			switch {
			case r.Calculated:
				create(alias, apichannel.Channel{
					Name:       UniqueChannelName(),
					DataType:   r.DataType,
					Virtual:    true,
					Expression: fmt.Sprintf("return %s * 2", data.Name),
				})
			case r.Internal:
				ch := channel.Channel{
					Name:       UniqueChannelName(),
					DataType:   r.DataType,
					Virtual:    r.Virtual,
					LocalIndex: index.Key.LocalKey(),
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).Declare(ctx, &ch)
				})).To(Succeed())
				aliases[ch.Key()] = alias
				keys = append(keys, ch.Key())
			}
		}
	})

	DescribeTable(
		"Should match the case",
		func(ctx SpecContext, c vectors.Case[channel.Where]) {
			where := channel.Where{And: []channel.Where{
				c.Where,
				{Key: &filter.Set[channel.Key]{In: keys}},
			}}
			res := MustSucceed(retrieveChannels(
				AuthedCtx(ctx, u),
				apichannel.RetrieveRequest{Where: &where},
			))
			Expect(lo.Map(res.Channels, func(ch apichannel.Channel, _ int) string {
				return aliases[ch.Key]
			})).To(ConsistOf(c.Expected))
		},
		lo.FilterMap(
			v.Cases,
			func(c vectors.Case[channel.Where], _ int) (TableEntry, bool) {
				return Entry(c.Name, c), usesVirtual(c.Where)
			},
		),
	)
})
