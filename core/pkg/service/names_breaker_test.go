// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channelversions "github.com/synnaxlabs/synnax/pkg/service/channel/versions"
	channelv1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	groupversions "github.com/synnaxlabs/synnax/pkg/service/group/versions"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// storedChannel is a channel that a v0.62 Core stored, in group when group is set.
type storedChannel struct {
	name     string
	group    *uuid.UUID
	internal bool
}

// createNamedV062Store writes folders and channels as a v0.62 Core left them, and
// returns the key of each channel in order.
func createNamedV062Store(
	ctx context.Context,
	n testutil.Node,
	folders []groupv1.Group,
	channels ...storedChannel,
) []channel.Key {
	GinkgoHelper()
	markAppliedBefore(
		ctx,
		n.DB,
		"Channel",
		channelversions.Migrations,
		"v63_channel_name_rule",
	)
	markAppliedBefore(
		ctx,
		n.DB,
		"Group",
		groupversions.Migrations,
		"v63_folder_name_rule",
	)
	alloc := make([]distchannel.Channel, len(channels))
	for i, c := range channels {
		alloc[i] = distchannel.Channel{
			Name:        c.name,
			Leaseholder: node.KeyFree,
			Virtual:     true,
		}
	}
	allocated := MustSucceed(n.Channel.Allocate(ctx, alloc))
	stored := make([]channelv1.Channel, len(allocated))
	keys := make([]channel.Key, len(allocated))
	for i, k := range allocated {
		stored[i] = channelv1.Channel{
			Name:        channels[i].name,
			Leaseholder: node.KeyFree,
			LocalKey:    k.LocalKey,
			DataType:    telem.Float32T,
			Virtual:     true,
			Group:       channels[i].group,
			Internal:    channels[i].internal,
		}
		keys[i] = k.Key()
	}
	Expect(gorp.WrapWriter[channelv1.Key, channelv1.Channel](n.DB).
		Set(ctx, stored...)).To(Succeed())
	Expect(gorp.WrapWriter[groupv1.Key, groupv1.Group](n.DB).
		Set(ctx, folders...)).To(Succeed())
	return keys
}

func retrieveChannel(
	ctx context.Context,
	db *gorp.DB,
	key channel.Key,
) channel.Channel {
	GinkgoHelper()
	var c channel.Channel
	Expect(gorp.NewRetrieve[channel.Key, channel.Channel]().
		Where(gorp.MatchKeys[channel.Key, channel.Channel](key)).
		Entry(&c).
		Exec(ctx, db)).To(Succeed())
	return c
}

func retrieveFolder(ctx context.Context, db *gorp.DB, key group.Key) group.Group {
	GinkgoHelper()
	var g group.Group
	Expect(gorp.NewRetrieve[group.Key, group.Group]().
		Where(gorp.MatchKeys[group.Key, group.Group](key)).
		Entry(&g).
		Exec(ctx, db)).To(Succeed())
	return g
}

var _ = Describe("Name upgrade breaker", func() {
	It("Should keep a channel that the name rule renames into a retired name", func(
		ctx SpecContext,
	) {
		n := testutil.NewNode(ctx)
		keys := createNamedV062Store(
			ctx,
			n,
			nil,
			storedChannel{name: "sy_range_alias.set"},
		)
		MustOpen(svctestutil.OpenLayer(ctx, n))
		var s status.Status[any]
		Expect(gorp.NewRetrieve[status.Key, status.Status[any]]().
			Where(gorp.Match(func(_ gorp.Context, s *status.Status[any]) (bool, error) {
				return s.Message == "Renamed channel", nil
			})).
			Entry(&s).
			Exec(ctx, n.DB)).To(Succeed())
		Expect(s.Name).To(Equal("sy_range_alias_set"))
		var c channel.Channel
		Expect(gorp.NewRetrieve[channel.Key, channel.Channel]().
			Where(gorp.MatchKeys[channel.Key, channel.Channel](keys[0])).
			Entry(&c).
			Exec(ctx, n.DB)).To(Succeed())
	})

	// D-283: the top-level "Metrics" folder is built in and keeps its name; the user's
	// channel takes the suffix, and no clash is left at the top.
	It("Should keep the Metrics folder and leave no folder and channel with one name "+
		"at the top", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		metrics := groupv1.Group{
			Key:   uuid.New(),
			Name:  "Metrics",
			Scope: resource.TypeChannel,
		}
		keys := createNamedV062Store(
			ctx,
			n,
			[]groupv1.Group{metrics},
			storedChannel{name: "Metrics"},
		)
		MustOpen(svctestutil.OpenLayer(ctx, n))
		Expect(retrieveFolder(ctx, n.DB, metrics.Key).Name).To(Equal("Metrics"))
		Expect(retrieveChannel(ctx, n.DB, keys[0]).Name).To(Equal("Metrics_2"))
		var folders []group.Group
		Expect(gorp.NewRetrieve[group.Key, group.Group]().
			Where(gorp.Match(func(_ gorp.Context, g *group.Group) (bool, error) {
				return g.Scope == resource.TypeChannel && g.Parent == nil, nil
			})).
			Entries(&folders).
			Exec(ctx, n.DB)).To(Succeed())
		var top []channel.Channel
		Expect(gorp.NewRetrieve[channel.Key, channel.Channel]().
			Where(gorp.Match(func(_ gorp.Context, c *channel.Channel) (bool, error) {
				return c.Group == nil, nil
			})).
			Entries(&top).
			Exec(ctx, n.DB)).To(Succeed())
		held := make(set.Set[string], len(top))
		for _, c := range top {
			held.Add(c.Name)
		}
		var clashes []string
		for _, f := range folders {
			if held.Contains(f.Name) {
				clashes = append(clashes, f.Name)
			}
		}
		Expect(clashes).To(BeEmpty())
	})

	It("Should give a rewritten channel the first free suffix of its rewritten name "+
		"when it yields to a folder", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		folder := groupv1.Group{
			Key:   uuid.New(),
			Name:  "a_b_2",
			Scope: resource.TypeChannel,
		}
		keys := createNamedV062Store(
			ctx,
			n,
			[]groupv1.Group{folder},
			storedChannel{name: "a_b"},
			storedChannel{name: "a.b"},
		)
		MustOpen(svctestutil.OpenLayer(ctx, n))
		Expect(retrieveChannel(ctx, n.DB, keys[0]).Name).To(Equal("a_b"))
		Expect(retrieveFolder(ctx, n.DB, folder.Key).Name).To(Equal("a_b_2"))
		Expect(retrieveChannel(ctx, n.DB, keys[1]).Name).To(Equal("a_b_3"))
	})

	It("Should give a rewritten folder the first free suffix of its rewritten name "+
		"when it yields to a channel", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		first := groupv1.Group{
			Key:   uuid.MustParse("00000000-0000-4000-8000-000000000001"),
			Name:  "pt",
			Scope: resource.TypeChannel,
		}
		second := groupv1.Group{
			Key:   uuid.MustParse("00000000-0000-4000-8000-000000000002"),
			Name:  "pt",
			Scope: resource.TypeChannel,
		}
		keys := createNamedV062Store(
			ctx,
			n,
			[]groupv1.Group{first, second},
			storedChannel{name: "pt_2"},
		)
		MustOpen(svctestutil.OpenLayer(ctx, n))
		Expect(retrieveChannel(ctx, n.DB, keys[0]).Name).To(Equal("pt_2"))
		Expect(retrieveFolder(ctx, n.DB, first.Key).Name).To(Equal("pt"))
		Expect(retrieveFolder(ctx, n.DB, second.Key).Name).To(Equal("pt_3"))
	})

	It("Should say why it renamed a channel whose name was not in NFC", func(
		ctx SpecContext,
	) {
		n := testutil.NewNode(ctx)
		keys := createNamedV062Store(ctx, n, nil, storedChannel{name: "cafe\u0301"})
		MustOpen(svctestutil.OpenLayer(ctx, n))
		Expect(retrieveChannel(ctx, n.DB, keys[0]).Name).To(Equal("caf\u00e9"))
		var s status.Status[any]
		Expect(gorp.NewRetrieve[status.Key, status.Status[any]]().
			Where(gorp.Match(func(_ gorp.Context, s *status.Status[any]) (bool, error) {
				return s.Message == "Renamed channel", nil
			})).
			Entry(&s).
			Exec(ctx, n.DB)).To(Succeed())
		Expect(s.Description).To(Or(
			ContainSubstring("NFC"),
			ContainSubstring("normal form"),
		))
	})
})
