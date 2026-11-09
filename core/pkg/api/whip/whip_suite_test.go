// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package whip_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	db  *gorp.DB
	svc *service.Layer
	// rackKey is a rack that no Driver runs, so a spec configures its tasks itself.
	rackKey rack.Key
)

func TestWHIP(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API WHIP Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	r := rack.Rack{Name: "whip-rack"}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Rack.NewWriter(tx).Create(ctx, &r)
	})).To(Succeed())
	rackKey = r.Key
})
