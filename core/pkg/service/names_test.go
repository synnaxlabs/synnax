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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
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

// markApplied records each step of chain before the step with key next as applied in
// namespace, as a v0.62 Core left it.
func markApplied(
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
	applied := make([]gorp.Migration, end)
	for i, m := range chain[:end] {
		applied[i] = migrate.New(
			m.Key(),
			func(context.Context, gorp.Tx, alamos.Instrumentation) error { return nil },
		)
	}
	Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
		DB:         db,
		Namespace:  namespace,
		Migrations: applied,
	})).To(Succeed())
}

var _ = Describe("Name upgrade", func() {
	It("Should rename, quote, and report the names of a v0.62 store", func(
		ctx SpecContext,
	) {
		n := testutil.NewNode(ctx)
		markApplied(
			ctx,
			n.DB,
			"Channel",
			channelversions.Migrations,
			"v63_channel_name_rule",
		)
		markApplied(
			ctx,
			n.DB,
			"Group",
			groupversions.Migrations,
			"v63_folder_name_rule",
		)
		names := []string{
			"pt-01",
			"stand.pt",
			"pressure",
			"pressure",
			"Rigs",
			"a.b",
			"doubled",
			"x.y",
			"a_b_2",
		}
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
		stored[6].Expression = "return pt-01 * 2"
		rigs := groupv1.Group{
			Key:   uuid.New(),
			Name:  "Rigs",
			Scope: resource.TypeChannel,
		}
		// A channel in another folder holds the first suffix of a moved channel.
		stored[8].Group = &rigs.Key
		Expect(gorp.WrapWriter[channelv1.Key, channelv1.Channel](n.DB).
			Set(ctx, stored...)).To(Succeed())
		ab := groupv1.Group{Key: uuid.New(), Name: "a_b", Scope: resource.TypeChannel}
		xy := groupv1.Group{Key: uuid.New(), Name: "x.y", Scope: resource.TypeChannel}
		Expect(
			gorp.WrapWriter[groupv1.Key, groupv1.Group](n.DB).Set(ctx, rigs, ab, xy),
		).
			To(Succeed())
		Expect(MustSucceed(arc.OpenTable(ctx, arc.TableConfig{DB: n.DB})).Close()).
			To(Succeed())
		isRoot := func(_ gorp.Context, d *directory.Directory) (bool, error) {
			return d.Parent == nil, nil
		}
		var root directory.Directory
		Expect(gorp.NewRetrieve[directory.Key, directory.Directory]().
			Where(gorp.Match(isRoot)).
			Entry(&root).
			Exec(ctx, n.DB)).To(Succeed())
		textArc := arc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
			Key:       uuid.New(),
			Name:      "text",
			Directory: &root.Key,
			Source: arc.Source{Variant: arc.TextSource{
				Text: text.Text{Doc: text.Create("x := pt-01 * 2 // pt-01")},
			}},
		}}}
		graphArc := arc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
			Key:       uuid.New(),
			Name:      "graph",
			Directory: &root.Key,
			Source: arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
				Functions: ir.Functions{
					{Key: "f", Body: ir.Body{Raw: "{ return pt-01 }"}},
				},
			}}},
		}}}
		Expect(gorp.WrapWriter[arc.Key, arc.Arc](n.DB).Set(ctx, textArc, graphArc)).
			To(Succeed())

		MustOpen(svctestutil.OpenLayer(ctx, n))

		chKeys := make([]channel.Key, len(keys))
		for i, k := range keys {
			chKeys[i] = k.Key()
		}
		var chs []channel.Channel
		Expect(gorp.NewRetrieve[channel.Key, channel.Channel]().
			Where(gorp.MatchKeys[channel.Key, channel.Channel](chKeys...)).
			Entries(&chs).
			Exec(ctx, n.DB)).To(Succeed())
		byKey := make(map[channel.Key]string, len(chs))
		for _, c := range chs {
			byKey[c.Key()] = c.Name
		}
		Expect(byKey).To(Equal(map[channel.Key]string{
			keys[0].Key(): "pt-01",
			keys[1].Key(): "stand_pt",
			keys[2].Key(): "pressure",
			keys[3].Key(): "pressure_2",
			keys[4].Key(): "Rigs",
			keys[5].Key(): "a_b_3",
			keys[6].Key(): "doubled",
			keys[7].Key(): "x_y",
			keys[8].Key(): "a_b_2",
		}))
		var folders []group.Group
		Expect(gorp.NewRetrieve[group.Key, group.Group]().
			Where(gorp.MatchKeys[group.Key, group.Group](rigs.Key, ab.Key, xy.Key)).
			Entries(&folders).
			Exec(ctx, n.DB)).To(Succeed())
		Expect(folders).To(ConsistOf(
			HaveField("Name", "Rigs_2"),
			HaveField("Name", "a_b"),
			HaveField("Name", "x_y_2"),
		))

		ofDoubled := func(_ gorp.Context, c *calculation.Calculation) (bool, error) {
			return c.Output == keys[6].Key(), nil
		}
		var calc calculation.Calculation
		Expect(gorp.NewRetrieve[calculation.Key, calculation.Calculation]().
			Where(gorp.Match(ofDoubled)).
			Entry(&calc).
			Exec(ctx, n.DB)).To(Succeed())
		Expect(calc.Expression).To(Equal("return 'pt-01' * 2"))
		Expect(calc.Reads).To(Equal([]channel.Key{keys[0].Key()}))

		var arcs []arc.Arc
		Expect(gorp.NewRetrieve[arc.Key, arc.Arc]().
			Where(gorp.MatchKeys[arc.Key, arc.Arc](
				textArc.Base().Key,
				graphArc.Base().Key,
			)).
			Entries(&arcs).
			Exec(ctx, n.DB)).To(Succeed())
		Expect(arcs).To(HaveLen(2))
		for _, a := range arcs {
			switch src := a.Base().Source.Variant.(type) {
			case arc.TextSource:
				Expect(
					src.Text.Materialize().Raw,
				).To(Equal("x := 'pt-01' * 2 // pt-01"))
			case arc.GraphSource:
				Expect(src.Graph.Functions[0].Body.Raw).To(Equal("{ return 'pt-01' }"))
			default:
				Fail("unexpected source")
			}
		}

		var statuses []status.Status[any]
		Expect(gorp.NewRetrieve[status.Key, status.Status[any]]().
			Where(gorp.Match(func(_ gorp.Context, s *status.Status[any]) (bool, error) {
				return s.Message == "Renamed channel" ||
					s.Message == "Renamed folder", nil
			})).
			Entries(&statuses).
			Exec(ctx, n.DB)).To(Succeed())
		reported := make(map[string]string, len(statuses))
		for _, s := range statuses {
			Expect(s.Variant).To(Equal(status.VariantWarning))
			reported[s.Name] = s.Message
			if s.Name == "a_b_3" {
				Expect(s.Description).To(ContainSubstring(`"a.b" to "a_b_3"`))
			}
		}
		Expect(reported).To(Equal(map[string]string{
			"stand_pt":   "Renamed channel",
			"pressure_2": "Renamed channel",
			"a_b_3":      "Renamed channel",
			"x_y":        "Renamed channel",
			"Rigs_2":     "Renamed folder",
			"x_y_2":      "Renamed folder",
		}))
		Expect(MustSucceed(resourcev0.StagedRenames(n.DB))).To(BeEmpty())
	})
})
