// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bind_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

func TestBind(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Bind Suite", SuiteConfig())
}

var (
	db     *gorp.DB
	svc    *service.Layer
	binder *bind.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	binder = MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   db,
		RBAC: svc.RBAC,
	}))
})

var _ = ShouldNotLeakGoroutinesPerSpec()
