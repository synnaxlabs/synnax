// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot_test

import (
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot/versions"
	v7 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v7"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// openChannels opens a node with a channel service on its DB, and creates count
// channels. It returns the DB and the channel keys.
func openChannels(ctx SpecContext, count int) (*gorp.DB, channel.Keys) {
	GinkgoHelper()
	node := testutil.NewNode(ctx)
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: node.DB,
	}))
	channelSvc := MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel: node.Channel,
		DB:      node.DB,
		HostKey: node.Identity.HostKey,
		Status:  statusSvc,
	}))
	keys := MustSucceed(channeltestutil.CreateVirtual(ctx, node.DB, channelSvc, count))
	return node.DB, keys
}

var _ = Describe("Stored v7 line plots", func() {
	It(
		"Should keep key references to ranges the Core does not hold",
		func(ctx SpecContext) {
			db, keys := openChannels(ctx, 1)
			ch := keys[0]
			// Mark the chain through v7 as applied, so only the later steps run on
			// open.
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "LinePlot",
				Migrations: versions.Migrations[:len(versions.Migrations)-7],
			})).To(Succeed())
			saved := ranger.Range{
				Key:       uuid.New(),
				Name:      "saved",
				TimeRange: telem.TimeRange{Start: 1, End: 2},
			}
			Expect(gorp.NewCreate[ranger.Key, ranger.Range]().
				Entry(&saved).Exec(ctx, db)).To(Succeed())
			local := uuid.New().String()
			old := v7.LinePlot{
				Key:      uuid.New(),
				Name:     "plot",
				Channels: v7.Channels{Y1: []channel.Key{ch}},
				Ranges: v7.Ranges{X1: []string{
					saved.Key.String(), local, "not-a-uuid", "rolling1m",
				}},
				Lines: []v7.Line{
					{
						Key: fmt.Sprintf(
							"y1---x1---%s---0---%d",
							saved.Key,
							ch,
						),
						StrokeWidth: 3,
					},
					{
						Key:         fmt.Sprintf("y1---x1---%s---0---%d", local, ch),
						StrokeWidth: 4,
					},
					{
						Key:         fmt.Sprintf("y1---x1---rolling1m---0---%d", ch),
						StrokeWidth: 5,
					},
				},
			}
			Expect(gorp.NewCreate[v7.Key, v7.LinePlot]().
				Entry(&old).Exec(ctx, db)).To(Succeed())
			projects := MustOpen(project.OpenService(ctx, project.ServiceConfig{
				DB: db,
			}))
			proj := project.Project{Name: "Project"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return projects.NewWriter(tx).Create(ctx, &proj)
			})).To(Succeed())
			owner := ontology.Relationship{
				From: resource.ID{Type: resource.TypeProject, Key: proj.Key.String()},
				Type: ontology.RelationshipTypeParentOf,
				To:   resource.ID{Type: resource.TypeLineplot, Key: old.Key.String()},
			}
			Expect(gorp.NewCreate[string, ontology.Relationship]().
				Entry(&owner).
				Exec(ctx, db)).To(Succeed())
			MustOpen(lineplot.OpenService(ctx, lineplot.ServiceConfig{
				DB:   db,
				ImEx: imex.NewService(),
			}))
			var got lineplot.LinePlot
			Expect(gorp.NewRetrieve[lineplot.Key, lineplot.LinePlot]().
				Where(gorp.MatchKeys[lineplot.Key, lineplot.LinePlot](old.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.Ranges).To(Equal([]ranger.Ref{
				keyRef(saved.Key.String()),
				keyRef(local),
				{Variant: ranger.WindowRef{
					Window: telem.Window{
						Variant: telem.DynamicWindow{TimeSpan: telem.Minute},
					},
				}},
			}))
			// The three v7 lines style one channel, so they merge into one line.
			Expect(got.Lines).To(HaveLen(1))
		},
	)

	It("Should drop channels the Core does not hold", func(ctx SpecContext) {
		db, keys := openChannels(ctx, 2)
		y1, y2 := keys[0], keys[1]
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "LinePlot",
			Migrations: versions.Migrations,
		})).To(Succeed())
		stored := lineplot.LinePlot{
			Key:  uuid.New(),
			Name: "plot",
			Channels: lineplot.Channels{
				X1: 6,
				Y1: []channel.Key{y1, 8},
				Y2: []channel.Key{y2},
			},
			Lines: []lineplot.Line{
				{YAxis: lineplot.YAxisKeyY1, YChannel: y1, XChannel: 6, StrokeWidth: 3},
				{YAxis: lineplot.YAxisKeyY1, YChannel: 8, XChannel: 6, StrokeWidth: 4},
			},
		}
		Expect(gorp.NewCreate[lineplot.Key, lineplot.LinePlot]().
			Entry(&stored).
			Exec(ctx, db)).To(Succeed())
		MustOpen(lineplot.OpenService(ctx, lineplot.ServiceConfig{
			DB:   db,
			ImEx: imex.NewService(),
		}))
		var got lineplot.LinePlot
		Expect(gorp.NewRetrieve[lineplot.Key, lineplot.LinePlot]().
			Where(gorp.MatchKeys[lineplot.Key, lineplot.LinePlot](stored.Key)).
			Entry(&got).
			Exec(ctx, db)).To(Succeed())
		Expect(got.Channels).To(Equal(lineplot.Channels{
			Y1: []channel.Key{y1},
			Y2: []channel.Key{y2},
		}))
		Expect(got.Lines).To(HaveLen(2))
		Expect(got.Lines[0]).To(SatisfyAll(
			HaveField("YChannel", y1),
			HaveField("XChannel", channel.Key(0)),
			HaveField("StrokeWidth", 3.0),
		))
		Expect(got.Lines[1].YChannel).To(Equal(y2))
	})
})
