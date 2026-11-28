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
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("KeysByPaths", func() {
	var (
		stand, hidden group.Group
		name          string
		top, inStand  channel.Channel
		inHidden      channel.Channel
	)
	folder := func(ctx SpecContext) group.Group {
		GinkgoHelper()
		g := group.Group{
			Key:   uuid.New(),
			Name:  UniqueChannelName(),
			Scope: resource.TypeChannel,
		}
		Expect(gorp.WrapWriter[group.Key, group.Group](db).Set(ctx, g)).To(Succeed())
		return g
	}
	create := func(ctx SpecContext, name string, g *group.Key) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
			Group:    g,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		// The name index holds one channel of a name, so the channels of one name are
		// stored as a v0.62 Core left them.
		ch.Name = name
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, ch)).
			To(Succeed())
		return ch
	}
	BeforeEach(func(ctx SpecContext) {
		stand, hidden = folder(ctx), folder(ctx)
		name = UniqueChannelName()
		top = create(ctx, name, nil)
		inStand = create(ctx, name, &stand.Key)
		inHidden = create(ctx, name, &hidden.Key)
	})
	allowed := func(id resource.ID) bool { return id != hidden.ResourceID() }

	It("Should look up a key, a name at the top level, and a path, in order",
		func(ctx SpecContext) {
			Expect(svc.KeysByPaths(ctx, db, allowed, []string{
				inStand.Key().String(),
				name,
				stand.Name + "." + name,
			})).To(Equal(channel.Keys{inStand.Key(), top.Key(), inStand.Key()}))
		})

	It("Should compare each name in NFC", func(ctx SpecContext) {
		cafe := create(ctx, "café_"+name, nil)
		Expect(svc.KeysByPaths(ctx, db, nil, []string{"café_" + name})).
			To(Equal(channel.Keys{cafe.Key()}))
	})

	It("Should name the paths of the name that the subject can see when a path is "+
		"missing", func(ctx SpecContext) {
		missing := "rig." + name
		shown := []string{name, stand.Name + "." + name}
		slices.Sort(shown)
		Expect(
			svc.KeysByPaths(ctx, db, allowed, []string{missing}),
		).Error().
			To(SatisfyAll(
				MatchError(query.ErrNotFound),
				MatchError(fmt.Sprintf(
					"%s does not exist. Did you mean %s?",
					missing,
					strings.Join(shown, ", "),
				)),
			))
	})

	It("Should treat a channel behind a folder that the subject cannot see as missing",
		func(ctx SpecContext) {
			path := hidden.Name + "." + name
			Expect(svc.KeysByPaths(ctx, db, allowed, []string{path})).Error().
				To(MatchError(query.ErrNotFound))
			Expect(svc.KeysByPaths(ctx, db, nil, []string{path})).
				To(Equal(channel.Keys{inHidden.Key()}))
		})

	It("Should look up by key a channel that the subject can see behind a folder that "+
		"it cannot", func(ctx SpecContext) {
		Expect(svc.KeysByPaths(ctx, db, allowed, []string{inHidden.Key().String()})).
			To(Equal(channel.Keys{inHidden.Key()}))
	})

	It("Should name no path when no channel of the name exists", func(ctx SpecContext) {
		missing := UniqueChannelName()
		Expect(svc.KeysByPaths(ctx, db, nil, []string{missing})).Error().
			To(MatchError(missing + " does not exist"))
	})

	It("Should refuse a key that no channel holds", func(ctx SpecContext) {
		Expect(svc.KeysByPaths(ctx, db, nil, []string{"4294967"})).Error().
			To(MatchError(query.ErrNotFound))
	})
})
