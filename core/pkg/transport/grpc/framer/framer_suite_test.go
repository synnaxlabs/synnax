// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	svcchannel "github.com/synnaxlabs/synnax/pkg/service/channel"
	svcmock "github.com/synnaxlabs/synnax/pkg/service/mock"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	channelSvc    *svcchannel.Service
	apiChannelSvc *channel.Service
)

func TestFramer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Transport gRPC Framer Suite", SuiteConfig())
}

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	svc := MustOpen(svcmock.OpenLayer(ctx, mock.NewNode(ctx)))
	channelSvc = svc.Channel
	apiChannelSvc = MustSucceed(channel.NewService(channel.ServiceConfig{
		RBAC:    svc.RBAC,
		Channel: svc.Channel,
		Status:  svc.Status,
		Alias:   svc.Alias,
	}))
})

var _ = ShouldNotLeakGoroutinesPerSpec()
