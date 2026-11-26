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
	"slices"
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
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// markAppliedBefore records each step of chain before the step with key next as applied
// in namespace, as a v0.62 Core left it.
func markAppliedBefore(
	ctx context.Context,
	db *gorp.DB,
	namespace string,
	chain []gorp.Migration,
	next string,
) {
	GinkgoHelper()
	end := slices.IndexFunc(
		chain,
		func(m gorp.Migration) bool { return m.Key() == next },
	)
	Expect(end).To(BeNumerically(">", 0))
	keys := make([]string, end)
	for i, m := range chain[:end] {
		keys[i] = m.Key()
	}
	markApplied(ctx, db, namespace, keys...)
}

// storeV062 makes n hold a store as a v0.62 Core left it: the chain steps before the
// name passes applied, a channel v1 record for each name, and the folders. edit changes
// the records before they are stored. It returns the records.
func storeV062(
	ctx context.Context,
	n testutil.Node,
	names []string,
	edit func([]channelv1.Channel),
	folders ...groupv1.Group,
) []channelv1.Channel {
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
	alloc := make([]distchannel.Channel, len(names))
	for i, name := range names {
		alloc[i] = distchannel.Channel{
			Name:        name,
			Leaseholder: node.KeyFree,
			Virtual:     true,
		}
	}
	keys := MustSucceed(n.Channel.Allocate(ctx, alloc))
	stored := make([]channelv1.Channel, len(keys))
	for i, k := range keys {
		stored[i] = channelv1.Channel{
			Name:        k.Name,
			Leaseholder: node.KeyFree,
			LocalKey:    k.LocalKey,
			DataType:    telem.Float32T,
			Virtual:     true,
		}
	}
	edit(stored)
	Expect(gorp.WrapWriter[channelv1.Key, channelv1.Channel](n.DB).
		Set(ctx, stored...)).To(Succeed())
	Expect(gorp.WrapWriter[groupv1.Key, groupv1.Group](n.DB).Set(ctx, folders...)).
		To(Succeed())
	return stored
}

// folder returns a top-level channel folder named name.
func folder(name string) groupv1.Group {
	return groupv1.Group{Key: uuid.New(), Name: name, Scope: resource.TypeChannel}
}

// namesOf returns the name of each stored channel by key.
func namesOf(
	ctx context.Context,
	n testutil.Node,
	stored []channelv1.Channel,
) map[channel.Key]string {
	GinkgoHelper()
	keys := make([]channel.Key, len(stored))
	for i, c := range stored {
		keys[i] = c.Key()
	}
	var chs []channel.Channel
	Expect(gorp.NewRetrieve[channel.Key, channel.Channel]().
		Where(gorp.MatchKeys[channel.Key, channel.Channel](keys...)).
		Entries(&chs).
		Exec(ctx, n.DB)).To(Succeed())
	names := make(map[channel.Key]string, len(chs))
	for _, c := range chs {
		names[c.Key()] = c.Name
	}
	return names
}

// renameStatuses returns each rename status by the name it reports.
func renameStatuses(
	ctx context.Context,
	n testutil.Node,
) map[string]status.Status[any] {
	GinkgoHelper()
	isRename := func(_ gorp.Context, s *status.Status[any]) (bool, error) {
		return s.Message == "Renamed channel" || s.Message == "Renamed folder", nil
	}
	var statuses []status.Status[any]
	Expect(gorp.NewRetrieve[status.Key, status.Status[any]]().
		Where(gorp.Match(isRename)).
		Entries(&statuses).
		Exec(ctx, n.DB)).To(Succeed())
	byName := make(map[string]status.Status[any], len(statuses))
	for _, s := range statuses {
		Expect(s.Variant).To(Equal(status.VariantWarning))
		byName[s.Name] = s
	}
	return byName
}

var _ = Describe("Name upgrade", func() {
	It("Should rename and report each name of a v0.62 store that a path cannot hold",
		func(ctx SpecContext) {
			n := testutil.NewNode(ctx)
			rigs, rigs2, xy := folder("Rigs"), folder("Rigs"), folder("x.y")
			stored := storeV062(ctx, n, []string{
				"pt-01",
				"Stand A",
				"stand.pt",
				"pressure",
				"pressure",
				"a.b",
				"pt\xff",
			}, func(cs []channelv1.Channel) {
				cs[5].Group = &rigs.Key
			}, rigs, rigs2, xy)

			MustOpen(svctestutil.OpenLayer(ctx, n))

			Expect(namesOf(ctx, n, stored)).To(Equal(map[channel.Key]string{
				stored[0].Key(): "pt-01",
				stored[1].Key(): "Stand A",
				stored[2].Key(): "stand_pt",
				stored[3].Key(): "pressure",
				stored[4].Key(): "pressure_2",
				stored[5].Key(): "a_b",
				stored[6].Key(): "pt�",
			}))
			var folders []group.Group
			Expect(gorp.NewRetrieve[group.Key, group.Group]().
				Where(gorp.MatchKeys[group.Key, group.Group](
					rigs.Key,
					rigs2.Key,
					xy.Key,
				)).
				Entries(&folders).
				Exec(ctx, n.DB)).To(Succeed())
			Expect(folders).To(ConsistOf(
				HaveField("Name", "Rigs"),
				HaveField("Name", "Rigs_2"),
				HaveField("Name", "x_y"),
			))
			statuses := renameStatuses(ctx, n)
			reported := make(map[string]string, len(statuses))
			for name, s := range statuses {
				reported[name] = s.Message
			}
			Expect(reported).To(Equal(map[string]string{
				"stand_pt":   "Renamed channel",
				"pressure_2": "Renamed channel",
				"a_b":        "Renamed channel",
				"pt�":        "Renamed channel",
				"Rigs_2":     "Renamed folder",
				"x_y":        "Renamed folder",
			}))
			Expect(statuses["stand_pt"].Description).To(Equal(
				`The upgrade renamed channel "stand.pt" to "stand_pt", because a '.' ` +
					"now separates the folders of a path, so a name cannot hold one. " +
					"Update each reference to the old name outside Synnax.",
			))
			Expect(statuses["pressure_2"].Description).To(Equal(
				`The upgrade renamed channel "pressure" to "pressure_2", because ` +
					"another channel in the same folder held the same name. Update each " +
					"reference to the old name outside Synnax.",
			))
			Expect(statuses["pt�"].Description).To(ContainSubstring(
				"because the old name held bytes that are not valid text.",
			))
			Expect(statuses["Rigs_2"].Description).To(ContainSubstring(
				"because another folder beside it held the same name.",
			))
			Expect(MustSucceed(resourcev0.StagedRenames(n.DB))).To(BeEmpty())
		})

	It("Should rename the storage of a renamed channel", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		storeV062(ctx, n, nil, func([]channelv1.Channel) {})
		idx := MustSucceed(n.Channel.Allocate(ctx, []distchannel.Channel{{
			Name:     "stand.time",
			DataType: telem.TimestampT,
			IsIndex:  true,
		}}))[0]
		Expect(n.Storage.TS.CreateChannel(
			ctx,
			distchannel.ToStorage([]distchannel.Channel{idx})...,
		)).To(Succeed())
		Expect(gorp.WrapWriter[channelv1.Key, channelv1.Channel](n.DB).Set(
			ctx,
			channelv1.Channel{
				Name:        idx.Name,
				Leaseholder: idx.Leaseholder,
				LocalKey:    idx.LocalKey,
				LocalIndex:  idx.LocalKey,
				IsIndex:     true,
				DataType:    telem.TimestampT,
			},
		)).To(Succeed())
		MustOpen(svctestutil.OpenLayer(ctx, n))
		stored := MustSucceed(
			n.Storage.TS.RetrieveChannel(ctx, cesium.ChannelKey(idx.Key())),
		)
		Expect(stored.Name).To(Equal("stand_time"))
	})
})
