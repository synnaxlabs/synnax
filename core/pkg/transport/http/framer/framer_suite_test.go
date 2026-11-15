// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	svcchannel "github.com/synnaxlabs/synnax/pkg/service/channel"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	db            *gorp.DB
	channelSvc    *svcchannel.Service
	apiChannelSvc *channel.Service
	framerSvc     *svcframer.Service
)

func TestFramer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Transport HTTP Framer Suite", SuiteConfig())
}

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	svc := MustOpen(svctestutil.OpenLayer(ctx, node))
	channelSvc, framerSvc = svc.Channel, svc.Framer
	apiChannelSvc = MustSucceed(channel.NewService(channel.ServiceConfig{
		DB:      db,
		RBAC:    svc.RBAC,
		Channel: svc.Channel,
		Status:  svc.Status,
	}))
})

var _ = ShouldNotLeakGoroutinesPerSpec()
