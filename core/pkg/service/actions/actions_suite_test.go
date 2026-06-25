// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package actions_test

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

func TestActions(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Actions Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	dist mock.Node
	sigs *signals.Provider
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	cluster := DeferClose(mock.NewCluster())
	dist = DeferClose(cluster.Provision(ctx))
	sigs = MustSucceed(signals.New(signals.Config{
		Channel: channel.Wrap(dist.Channel),
		Framer:  framer.Wrap(dist.Framer),
	}))
})

// testAction is a small concrete action type used to instantiate the generic
// types under test. Its shape mirrors the per-service Action union (a
// discriminator plus a payload) without depending on any concrete service.
type testAction struct {
	Type    string `json:"type" msgpack:"type"`
	Payload string `json:"payload" msgpack:"payload"`
}
