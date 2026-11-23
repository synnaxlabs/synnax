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
	"fmt"
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
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
	resourcev0 "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// topClashes returns each name that a top-level channel folder and a top-level channel
// both hold.
func topClashes(ctx context.Context, n testutil.Node) []string {
	GinkgoHelper()
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
	return clashes
}

// channelsWithKey returns the stored channels that hold key.
func channelsWithKey(
	ctx context.Context,
	n testutil.Node,
	key channel.Key,
) []channel.Channel {
	GinkgoHelper()
	var found []channel.Channel
	Expect(gorp.NewRetrieve[channel.Key, channel.Channel]().
		Where(gorp.Match(func(_ gorp.Context, c *channel.Channel) (bool, error) {
			return c.Key() == key, nil
		})).
		Entries(&found).
		Exec(ctx, n.DB)).To(Succeed())
	return found
}

// storeV062Metrics stores the metrics channels of the Core of n, with their storage, in
// the folder core, and the folders, as a v0.62 Core left them.
func storeV062Metrics(
	ctx context.Context,
	n testutil.Node,
	core group.Key,
	folders ...groupv1.Group,
) {
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
	prefix := fmt.Sprintf("sy_node_%s_metrics_", n.Identity.HostKey)
	idx := MustSucceed(n.Channel.Allocate(ctx, []distchannel.Channel{{
		Name:     prefix + "time",
		DataType: telem.TimestampT,
		IsIndex:  true,
	}}))[0]
	names := []string{"cpu_percentage", "mem_percentage", "ts_size_gb", "kv_size_gb"}
	data := make([]distchannel.Channel, len(names))
	for i, name := range names {
		data[i] = distchannel.Channel{
			Name:       prefix + name,
			DataType:   telem.Float32T,
			LocalIndex: idx.LocalKey,
		}
	}
	stored := append(
		[]distchannel.Channel{idx},
		MustSucceed(n.Channel.Allocate(ctx, data))...,
	)
	Expect(n.Storage.TS.CreateChannel(ctx, distchannel.ToStorage(stored)...)).
		To(Succeed())
	records := make([]channelv1.Channel, len(stored))
	for i, ch := range stored {
		records[i] = channelv1.Channel{
			Name:        ch.Name,
			Leaseholder: ch.Leaseholder,
			DataType:    ch.DataType,
			IsIndex:     ch.IsIndex,
			LocalKey:    ch.LocalKey,
			LocalIndex:  ch.LocalIndex,
			Group:       &core,
		}
	}
	Expect(gorp.WrapWriter[channelv1.Key, channelv1.Channel](n.DB).
		Set(ctx, records...)).To(Succeed())
	Expect(gorp.WrapWriter[groupv1.Key, groupv1.Group](n.DB).
		Set(ctx, folders...)).To(Succeed())
}

var _ = Describe("Name upgrade breaker round 2", func() {
	It("Should name a channel in the reason of a folder that yields to a channel", func(
		ctx SpecContext,
	) {
		n := testutil.NewNode(ctx)
		rigs := folder("Rigs")
		storeV062(ctx, n, []string{"Rigs"}, func([]channelv1.Channel) {}, rigs)
		MustOpen(svctestutil.OpenLayer(ctx, n))
		Expect(retrieveFolder(ctx, n.DB, rigs.Key).Name).To(Equal("Rigs_2"))
		statuses := renameStatuses(ctx, n)
		Expect(statuses).To(HaveKey("Rigs_2"))
		Expect(statuses["Rigs_2"].Description).To(ContainSubstring(
			"another folder or a channel beside it held the same name",
		))
	})

	It("Should say that a name was not in NFC when NFC broke its long run of "+
		"combining marks", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		name := "a" + strings.Repeat("\u0301", 40)
		createNamedV062Store(ctx, n, nil, storedChannel{name: name})
		MustOpen(svctestutil.OpenLayer(ctx, n))
		var s status.Status[any]
		Expect(gorp.NewRetrieve[status.Key, status.Status[any]]().
			Where(gorp.Match(func(_ gorp.Context, s *status.Status[any]) (bool, error) {
				return s.Message == "Renamed channel", nil
			})).
			Entry(&s).
			Exec(ctx, n.DB)).To(Succeed())
		Expect(s.Description).ToNot(ContainSubstring("shows as nothing"))
		Expect(s.Description).To(ContainSubstring("NFC"))
	})

	It("Should keep a channel that the name rule renames into sy_calculation_status",
		func(ctx SpecContext) {
			n := testutil.NewNode(ctx)
			keys := createNamedV062Store(
				ctx,
				n,
				nil,
				storedChannel{name: "sy_calculation.status"},
			)
			MustOpen(svctestutil.OpenLayer(ctx, n))
			Expect(channelsWithKey(ctx, n, keys[0])).To(ConsistOf(
				HaveField("Name", "sy_calculation_status"),
			))
		})

	It("Should keep a user channel named sy_calculation_status across a restart",
		func(ctx SpecContext) {
			n := testutil.NewNode(ctx)
			l := MustSucceed(svctestutil.OpenLayer(ctx, n))
			user := channel.Channel{
				Name:        "sy_calculation_status",
				DataType:    telem.StringT,
				Virtual:     true,
				Leaseholder: node.KeyFree,
			}
			Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return l.Channel.NewWriter(tx).Create(ctx, &user)
			})).To(Succeed())
			Expect(channelsWithKey(ctx, n, user.Key())).To(HaveLen(1))
			Expect(l.Close()).To(Succeed())
			MustOpen(svctestutil.OpenLayer(ctx, n))
			Expect(channelsWithKey(ctx, n, user.Key())).To(HaveLen(1))
		})

	It("Should keep the name of the Metrics folder of the Core when a user folder at "+
		"the top holds the same name", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		core := groupv1.Group{
			Key:   uuid.MustParse("ffffffff-ffff-4fff-bfff-ffffffffffff"),
			Name:  groupv1.MetricsFolder,
			Scope: resource.TypeChannel,
		}
		user := groupv1.Group{
			Key:   uuid.MustParse("00000000-0000-4000-8000-000000000001"),
			Name:  groupv1.MetricsFolder,
			Scope: resource.TypeChannel,
		}
		storeV062Metrics(ctx, n, core.Key, core, user)
		MustOpen(svctestutil.OpenLayer(ctx, n))
		prefix := fmt.Sprintf("sy_node_%s_metrics_", n.Identity.HostKey)
		folderOf := func(name string) string {
			GinkgoHelper()
			var c channel.Channel
			Expect(gorp.NewRetrieve[channel.Key, channel.Channel]().
				Where(gorp.Match(func(_ gorp.Context, c *channel.Channel) (bool, error) {
					return c.Name == name, nil
				})).
				Entry(&c).
				Exec(ctx, n.DB)).To(Succeed())
			Expect(c.Group).ToNot(BeNil(), name)
			return retrieveFolder(ctx, n.DB, *c.Group).Name
		}
		Expect(map[string]string{
			"core folder":   retrieveFolder(ctx, n.DB, core.Key).Name,
			"time":          folderOf(prefix + "time"),
			"total_size_gb": folderOf(prefix + "total_size_gb"),
		}).To(Equal(map[string]string{
			"core folder":   groupv1.MetricsFolder,
			"time":          groupv1.MetricsFolder,
			"total_size_gb": groupv1.MetricsFolder,
		}))
	})

	It("Should leave no folder and channel with one name at the top when the store "+
		"holds no Metrics folder", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		host := groupv1.Group{
			Key:   uuid.New(),
			Name:  "Host",
			Scope: resource.TypeChannel,
		}
		createNamedV062Store(
			ctx,
			n,
			[]groupv1.Group{host},
			storedChannel{name: "Metrics"},
		)
		MustOpen(svctestutil.OpenLayer(ctx, n))
		Expect(topClashes(ctx, n)).To(BeEmpty())
	})

	It("Should write the new name of a user channel into text when the only other "+
		"holder of its old name is a retired channel", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		stored := storeV062(ctx, n, []string{
			"sy_range_alias_set",
			"sy_range_alias_set",
			"doubled",
		}, func(cs []channelv1.Channel) {
			cs[0].Internal = true
			cs[2].Expression = "return sy_range_alias_set * 2"
		})
		l := MustOpen(svctestutil.OpenLayer(ctx, n))
		Expect(channelsWithKey(ctx, n, stored[0].Key())).To(BeEmpty())
		Expect(namesOf(ctx, n, stored[1:2])).To(Equal(map[channel.Key]string{
			stored[1].Key(): "sy_range_alias_set_2",
		}))
		Expect(calculationOf(ctx, n, l.Calculation, stored[2].Key()).Expression).
			To(Equal("return 'sy_range_alias_set_2' * 2"))
	})

	It("Should leave text alone when several renamed channels held its name",
		func(ctx SpecContext) {
			n := testutil.NewNode(ctx)
			metrics := folder(groupv1.MetricsFolder)
			stored := storeV062(ctx, n, []string{"Metrics", "Metrics", "doubled"}, func(
				cs []channelv1.Channel,
			) {
				cs[2].Expression = "return Metrics * 2"
			}, metrics)
			l := MustOpen(svctestutil.OpenLayer(ctx, n))
			Expect(namesOf(ctx, n, stored[:2])).To(Equal(map[channel.Key]string{
				stored[0].Key(): "Metrics_3",
				stored[1].Key(): "Metrics_2",
			}))
			Expect(calculationOf(ctx, n, l.Calculation, stored[2].Key()).Expression).
				To(Equal("return Metrics * 2"))
		})

	It("Should leave text alone when a channel in a folder still holds the old name",
		func(ctx SpecContext) {
			n := testutil.NewNode(ctx)
			metrics, rig := folder(groupv1.MetricsFolder), folder("Rig")
			stored := storeV062(ctx, n, []string{"Metrics", "Metrics", "doubled"}, func(
				cs []channelv1.Channel,
			) {
				cs[1].Group = &rig.Key
				cs[2].Expression = "return Metrics * 2"
			}, metrics, rig)
			l := MustOpen(svctestutil.OpenLayer(ctx, n))
			Expect(namesOf(ctx, n, stored[:2])).To(Equal(map[channel.Key]string{
				stored[0].Key(): "Metrics_2",
				stored[1].Key(): "Metrics",
			}))
			Expect(calculationOf(ctx, n, l.Calculation, stored[2].Key()).Expression).
				To(Equal("return Metrics * 2"))
		})

	It("Should start again and keep the channel after the name rule renames it into "+
		"sy_calculation_status", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		keys := createNamedV062Store(
			ctx,
			n,
			nil,
			storedChannel{name: "sy_calculation.status"},
		)
		open := func() string {
			GinkgoHelper()
			l, err := svctestutil.OpenLayer(ctx, n)
			if err != nil {
				return err.Error()
			}
			Expect(l.Close()).To(Succeed())
			return ""
		}
		first := open()
		kept := len(channelsWithKey(ctx, n, keys[0]))
		second := open()
		Expect(map[string]any{
			"first start":   first,
			"channels kept": kept,
			"second start":  second,
		}).To(Equal(map[string]any{
			"first start":   "",
			"channels kept": 1,
			"second start":  "",
		}))
	})

	It("Should delete a stored retired channel and its storage", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
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
		idx := MustSucceed(n.Channel.Allocate(ctx, []distchannel.Channel{{
			Name:     "sy_range_alias_set_time",
			DataType: telem.TimestampT,
			IsIndex:  true,
		}}))[0]
		data := MustSucceed(n.Channel.Allocate(ctx, []distchannel.Channel{{
			Name:       "sy_range_alias_set",
			DataType:   telem.Float32T,
			LocalIndex: idx.LocalKey,
		}}))[0]
		stored := []distchannel.Channel{idx, data}
		Expect(n.Storage.TS.CreateChannel(ctx, distchannel.ToStorage(stored)...)).
			To(Succeed())
		records := make([]channelv1.Channel, len(stored))
		for i, ch := range stored {
			records[i] = channelv1.Channel{
				Name:        ch.Name,
				Leaseholder: ch.Leaseholder,
				DataType:    ch.DataType,
				IsIndex:     ch.IsIndex,
				LocalKey:    ch.LocalKey,
				LocalIndex:  ch.LocalIndex,
				Internal:    true,
			}
		}
		Expect(gorp.WrapWriter[channelv1.Key, channelv1.Channel](n.DB).
			Set(ctx, records...)).To(Succeed())
		MustOpen(svctestutil.OpenLayer(ctx, n))
		Expect(channelsWithKey(ctx, n, data.Key())).To(BeEmpty())
		Expect(n.Storage.TS.RetrieveChannels(ctx, uint32(data.Key()))).
			Error().To(MatchError(cesium.ErrChannelNotFound))
	})

	It("Should delete the retired channels after a start that fails between the "+
		"channel chain and the composition delete", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		stored := storeV062(
			ctx,
			n,
			[]string{"sy_ontology_resource_set"},
			func(cs []channelv1.Channel) { cs[0].Internal = true },
		)
		Expect(resourcev0.StageRename(
			ctx,
			n.DB,
			resource.ID{Type: resource.TypeChannel, Key: "999999"},
			"ghost",
		)).To(Succeed())
		Expect(svctestutil.OpenLayer(ctx, n)).Error().
			To(MatchError(ContainSubstring("names no record")))
		Expect(MustSucceed(channelv1.RetiredKeys(n.DB))).
			To(ConsistOf(stored[0].Key()))
		Expect(resourcev0.ClearRenames(ctx, n.DB)).To(Succeed())
		MustOpen(svctestutil.OpenLayer(ctx, n))
		Expect(channelsWithKey(ctx, n, stored[0].Key())).To(BeEmpty())
		Expect(MustSucceed(channelv1.RetiredKeys(n.DB))).To(BeEmpty())
	})

	It("Should keep a nested Metrics folder and rename it when a channel beside it "+
		"holds its name", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		rig := folder("Rig")
		parent := resource.ID{Type: resource.TypeGroup, Key: rig.Key.String()}
		nested := groupv1.Group{
			Key:    uuid.New(),
			Name:   groupv1.MetricsFolder,
			Scope:  resource.TypeChannel,
			Parent: &parent,
		}
		keys := createNamedV062Store(
			ctx,
			n,
			[]groupv1.Group{rig, nested},
			storedChannel{name: "Metrics", group: &rig.Key},
		)
		MustOpen(svctestutil.OpenLayer(ctx, n))
		Expect(retrieveChannel(ctx, n.DB, keys[0]).Name).To(Equal("Metrics"))
		Expect(retrieveFolder(ctx, n.DB, nested.Key).Name).To(Equal("Metrics_2"))
	})
})
