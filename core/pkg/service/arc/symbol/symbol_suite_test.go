// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
)

func TestSymbol(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Symbol Suite")
}

var (
	ctx  context.Context
	dist mock.Node
)

var _ = BeforeEach(func() { ctx = context.Background() })

var _ = BeforeSuite(func() {
	ctx = context.Background()
	distB := mock.NewCluster()
	dist = distB.Provision(ctx)
})

var _ = AfterSuite(func() {
	Expect(dist.Close()).To(Succeed())
})
