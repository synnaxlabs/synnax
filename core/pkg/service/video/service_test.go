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
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/video"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ServiceConfig", func() {
	It("Should require a DB", func() {
		Expect(video.ServiceConfig{}.Validate()).
			To(MatchError(ContainSubstring("db: must be non-nil")))
	})

	It("Should prefer the fields of the other config when set", func() {
		merged := video.ServiceConfig{}.Override(video.ServiceConfig{DB: db})
		Expect(merged.DB).To(BeIdenticalTo(db))
	})
})

var _ = Describe("OpenService", func() {
	It(
		"Should publish actions and deletes when a signals provider is set",
		func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: node.DB}))
			statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
				DB: node.DB,
			}))
			channelSvc := MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
				Channel: node.Channel,
				DB:      node.DB,
				HostKey: node.Identity.HostKey,
				Status:  statusSvc,
			}))
			sigs := MustSucceed(signals.New(signals.Config{
				DB:      node.DB,
				Channel: channelSvc,
				Framer:  node.Framer,
			}))
			MustOpen(video.OpenService(ctx, video.ServiceConfig{
				DB:      node.DB,
				Signals: sigs,
			}))
			for _, name := range []string{"sy_video_set", "sy_video_delete"} {
				var ch channel.Channel
				Expect(channelSvc.NewRetrieve().Where(channel.MatchNames(name)).
					Entry(&ch).Exec(ctx, node.DB)).To(Succeed())
				Expect(ch.Virtual).To(BeTrue())
				Expect(ch.Internal).To(BeTrue())
			}
		},
	)
})
