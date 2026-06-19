// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	. "github.com/synnaxlabs/x/testutil"
)

func TestSignals(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Signals Suite")
}

var (
	dist  mock.Node
	chSvc *channel.Service
	sigs  *signals.Provider
)

var _ = BeforeSuite(func(ctx SpecContext) {
	dist = mock.NewNode(ctx)
	chSvc = MustSucceed(channel.OpenService(ctx, channel.ServiceConfig{Channel: dist.Channel, DB: dist.DB, HostResolver: dist.Cluster, Ontology: dist.Ontology, Group: dist.Group, Search: dist.Search}))
	sigs = MustSucceed(signals.New(signals.Config{
		Channel: chSvc,
		Framer:  framer.Wrap(dist.Framer, chSvc),
	}))
})
