// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package metrics_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	builder   *mock.Cluster
	dist      mock.Node
	svcFramer *framer.Service
)

func TestMetrics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metrics Suite")
}

var _ = BeforeSuite(func() {
	builder = mock.NewCluster()
	ctx := context.Background()
	dist = builder.Provision(ctx)
	svcFramer = MustSucceed(framer.OpenService(ctx, framer.Config{
		Framer:  dist.Framer,
		Channel: dist.Channel,
	}))
})

var _ = AfterSuite(func() {
	Expect(svcFramer.Close()).To(Succeed())
	Expect(builder.Close()).To(Succeed())
})
