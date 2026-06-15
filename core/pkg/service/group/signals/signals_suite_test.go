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
	groupsignals "github.com/synnaxlabs/synnax/pkg/service/group/signals"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	. "github.com/synnaxlabs/x/testutil"
)

func TestSignals(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Group Signals Suite")
}

var dist mock.Node

var _ = BeforeSuite(func(ctx SpecContext) {
	builder := DeferClose(mock.NewCluster())
	dist = DeferClose(builder.Provision(ctx))
	sigs := MustSucceed(signals.New(signals.Config{Channel: dist.Channel, Framer: dist.Framer}))
	MustOpen(groupsignals.Publish(ctx, sigs, dist.Group.Observe()))
})
