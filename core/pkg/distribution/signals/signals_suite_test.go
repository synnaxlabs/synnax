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
	"context"
	"testing"

	"github.com/synnaxlabs/synnax/pkg/distribution/mock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSignals(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Signals Suite")
}

var (
	mockCluster *mock.Cluster
	dist        mock.Node
	ctx         = context.Background()
)

var _ = BeforeSuite(func() {
	mockCluster = mock.NewCluster()
	dist = mockCluster.Provision(ctx)
})

var _ = AfterSuite(func() {
	Expect(mockCluster.Close()).To(Succeed())
})
