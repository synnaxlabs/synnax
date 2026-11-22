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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// linked holds channel keys the way exported files do: alone, in a list, and in a
// nested value.
type linked struct {
	Channel  channel.Key   `json:"channel"`
	Channels []channel.Key `json:"channels"`
	Inner    struct {
		Channel channel.Key `json:"channel"`
	} `json:"inner"`
	Rate int `json:"rate"`
}

var _ = Describe("Links", Ordered, func() {
	var (
		db                   *gorp.DB
		svc                  *channel.Service
		stand, sensors       string
		rootName, nestedName string
		root, nested         channel.Key
	)

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		cfg, groupSvc := serviceConfig(ctx, node, node.DB)
		svc = MustOpen(channel.OpenService(ctx, cfg))
		stand, sensors = UniqueChannelName(), UniqueChannelName()
		rootName, nestedName = UniqueChannelName(), UniqueChannelName()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			standFolder := group.Group{Name: stand, Scope: resource.TypeChannel}
			if err := groupSvc.NewWriter(tx).Create(ctx, &standFolder); err != nil {
				return err
			}
			sensorsFolder := group.Group{
				Name:   sensors,
				Scope:  resource.TypeChannel,
				Parent: new(group.ResourceID(standFolder.Key)),
			}
			if err := groupSvc.NewWriter(tx).Create(ctx, &sensorsFolder); err != nil {
				return err
			}
			chs := []channel.Channel{
				{Name: rootName, DataType: telem.Float32T, Virtual: true},
				{
					Name:     nestedName,
					DataType: telem.Float32T,
					Virtual:  true,
					Group:    new(sensorsFolder.Key),
				},
			}
			if err := svc.NewWriter(tx).CreateMany(ctx, &chs); err != nil {
				return err
			}
			root, nested = chs[0].Key(), chs[1].Key()
			return nil
		})).To(Succeed())
	})

	Describe("LinkWriter", func() {
		It(
			"Should write each key as the exact path of its channel",
			func(ctx SpecContext) {
				v := linked{
					Channel:  root,
					Channels: []channel.Key{nested, root},
					Rate:     3,
				}
				v.Inner.Channel = nested
				nestedPath := stand + "." + sensors + "." + nestedName
				Expect(json.Marshal(v, svc.LinkWriter(ctx, db))).To(MatchJSON(`{
				"channel": "` + rootName + `",
				"channels": ["` + nestedPath + `", "` + rootName + `"],
				"inner": {"channel": "` + nestedPath + `"},
				"rate": 3
			}`))
			},
		)

		It(
			"Should write 0 for key 0 and for a key of no channel",
			func(ctx SpecContext) {
				v := linked{Channels: []channel.Key{nested + 1000}}
				Expect(json.Marshal(v, svc.LinkWriter(ctx, db))).To(MatchJSON(
					`{"channel": 0, "channels": [0], "inner": {"channel": 0}, "rate": 0}`,
				))
			},
		)
	})

	Describe("LinkReader", func() {
		It("Should resolve each path to the key of its channel", func(ctx SpecContext) {
			r := svc.NewLinkReader(ctx, db)
			var v linked
			Expect(json.Unmarshal([]byte(`{
				"channel": "`+rootName+`",
				"channels": ["`+stand+`.`+sensors+`.`+nestedName+`"],
				"rate": 3
			}`), &v, r.Options())).To(Succeed())
			Expect(r.Err()).To(Succeed())
			Expect(v.Channel).To(Equal(root))
			Expect(v.Channels).To(Equal([]channel.Key{nested}))
			Expect(v.Rate).To(Equal(3))
		})

		It("Should decode a number as a raw key", func(ctx SpecContext) {
			r := svc.NewLinkReader(ctx, db)
			var v linked
			Expect(json.Unmarshal(
				[]byte(`{"channel": 12345, "channels": [0]}`),
				&v,
				r.Options(),
			)).To(Succeed())
			Expect(r.Err()).To(Succeed())
			Expect(v.Channel).To(Equal(channel.Key(12345)))
			Expect(v.Channels).To(Equal([]channel.Key{0}))
		})

		It(
			"Should list every path that matches no channel, sorted",
			func(ctx SpecContext) {
				r := svc.NewLinkReader(ctx, db)
				var v linked
				Expect(json.Unmarshal([]byte(`{
				"channel": "plant.zeta",
				"channels": ["plant.alpha", "`+rootName+`", "plant.zeta"],
				"inner": {"channel": "`+nestedName+`"}
			}`), &v, r.Options())).To(Succeed())
				Expect(r.Err()).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						"no channel is at plant.alpha, plant.zeta, %s", nestedName,
					)),
				))
				Expect(v.Channel).To(BeZero())
				Expect(v.Channels).To(Equal([]channel.Key{0, root, 0}))
			},
		)

		It("Should read back the keys that a LinkWriter wrote", func(ctx SpecContext) {
			v := linked{Channel: nested, Channels: []channel.Key{root, nested}}
			b := MustSucceed(json.Marshal(v, svc.LinkWriter(ctx, db)))
			r := svc.NewLinkReader(ctx, db)
			var read linked
			Expect(json.Unmarshal(b, &read, r.Options())).To(Succeed())
			Expect(r.Err()).To(Succeed())
			Expect(read).To(Equal(v))
		})
	})
})
