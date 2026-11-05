// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package video_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/video"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ServiceConfig", func() {
	It("Should require a DB", func() {
		Expect(video.ServiceConfig{Ontology: otg}.Validate()).
			To(MatchError(ContainSubstring("db: must be non-nil")))
	})

	It("Should require an ontology", func() {
		Expect(video.ServiceConfig{DB: db}.Validate()).
			To(MatchError(ContainSubstring("ontology: must be non-nil")))
	})

	It("Should prefer the fields of the other config when set", func() {
		merged := video.ServiceConfig{}.Override(video.ServiceConfig{
			DB:       db,
			Ontology: otg,
		})
		Expect(merged.DB).To(BeIdenticalTo(db))
		Expect(merged.Ontology).To(BeIdenticalTo(otg))
	})
})

var _ = Describe("OpenService", func() {
	It(
		"Should publish actions and deletes when a signals provider is set",
		func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: node.DB}))
			groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
				DB:       node.DB,
				Ontology: otg,
			}))
			labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
				DB:       node.DB,
				Ontology: otg,
				Group:    groupSvc,
			}))
			statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
				DB:       node.DB,
				Ontology: otg,
				Group:    groupSvc,
				Label:    labelSvc,
			}))
			channelSvc := MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
				Channel:      node.Channel,
				DB:           node.DB,
				HostProvider: node.Cluster,
				Ontology:     otg,
				Group:        groupSvc,
				Status:       statusSvc,
			}))
			channelGraph := MustOpen(calcgraph.Open(ctx, calcgraph.Config{
				DB:      node.DB,
				Channel: channelSvc,
				Status:  statusSvc,
			}))
			framerSvc := MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
				DB:           node.DB,
				Framer:       node.Framer,
				Channel:      channelSvc,
				ChannelGraph: channelGraph,
			}))
			sigs := MustSucceed(signals.New(signals.Config{
				DB:      node.DB,
				Channel: channelSvc,
				Framer:  framerSvc,
			}))
			MustOpen(video.OpenService(ctx, video.ServiceConfig{
				DB:       node.DB,
				Ontology: otg,
				Signals:  sigs,
			}))
			for _, name := range []string{"sy_video_set", "sy_video_delete"} {
				var ch channel.Channel
				Expect(channelSvc.NewRetrieve().Where(channel.MatchNames(name)).
					Entry(&ch).Exec(ctx, nil)).To(Succeed())
				Expect(ch.Virtual).To(BeTrue())
				Expect(ch.Internal).To(BeTrue())
			}
		},
	)
})
