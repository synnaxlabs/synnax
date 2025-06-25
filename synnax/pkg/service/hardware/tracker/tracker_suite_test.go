// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package tracker_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/x/config"
)

var (
	ctx         = context.Background()
	mockCluster *mock.Cluster
	dist        mock.Node
)

var _ = BeforeSuite(func() {
	mockCluster = mock.NewCluster(distribution.Config{EnableSearch: config.False()})
	dist = mockCluster.Provision(ctx)
})

var _ = AfterSuite(func() {
	Expect(mockCluster.Close()).To(Succeed())
})

func TestTracker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tracker Suite")
}
