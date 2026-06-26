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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	dist mock.Node
	svc  *channel.Service
)

func TestChannel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Channel Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	cluster := DeferClose(mock.NewCluster())
	dist = DeferClose(cluster.Provision(ctx))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Search:   dist.Search,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       dist.DB,
		Group:    dist.Group,
		Ontology: dist.Ontology,
		Label:    labelSvc,
		Search:   dist.Search,
	}))
	svc = MustSucceed(channel.NewService(ctx, channel.ServiceConfig{
		DB:           dist.DB,
		Distribution: dist.Channel,
		Status:       statusSvc,
	}))
})
