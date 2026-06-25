// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	distmock "github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/security"
	secmock "github.com/synnaxlabs/synnax/pkg/security/mock"
	"github.com/synnaxlabs/synnax/pkg/service"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	apiLayer *api.Layer
	dist     *distribution.Layer
)

func TestHTTP(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Transport HTTP Suite")
}

var _ = BeforeSuite(func(ctx SpecContext) {
	node := distmock.MustOpenNode(ctx)
	dist = DeferClose(node.Layer)
	insecure := true
	sec := MustSucceed(security.NewProvider(security.ProviderConfig{
		Insecure: &insecure,
		KeySize:  secmock.SmallKeySize,
	}))
	svc := MustOpen(service.OpenLayer(ctx, service.LayerConfig{
		Distribution: dist,
		Security:     sec,
		Storage:      node.Storage,
	}))
	apiLayer = MustSucceed(api.NewLayer(api.LayerConfig{
		Service:      svc,
		Distribution: dist,
	}))
})
