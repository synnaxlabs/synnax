// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	. "github.com/synnaxlabs/x/testutil"
)

func TestWriter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Framer Writer Suite")
}

var (
	mockNode      mock.Node
	channelSvc    *channel.Service
	channelWriter channel.Writer
	writerSvc     *writer.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	mockNode = mock.NewNode(ctx)
	searchIdx := MustOpen(search.OpenIndex())

	otg := MustOpen(ontology.Open(ctx, ontology.Config{
		DB: mockNode.DB,
	}))
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       mockNode.DB,
		Ontology: otg,
		Search:   searchIdx,
	}))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       mockNode.DB,
		Ontology: otg,
		Group:    groupSvc,
		Search:   searchIdx,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       mockNode.DB,
		Label:    labelSvc,
		Ontology: otg,
		Group:    groupSvc,
		Search:   searchIdx,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      mockNode.Channel,
		DB:           mockNode.DB,
		HostResolver: mockNode.Cluster,
		Ontology:     otg,
		Group:        groupSvc,
		Search:       searchIdx,
		Status:       statusSvc,
	}))
	channelWriter = channelSvc.NewWriter(nil)
	writerSvc = MustSucceed(writer.NewService(writer.ServiceConfig{
		Framer:  mockNode.Framer,
		Channel: channelSvc,
	}))
})

var _ = ShouldNotLeakGoroutinesPerSpec()
