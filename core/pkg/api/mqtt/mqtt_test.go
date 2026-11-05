// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt_test

import (
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apimqtt "github.com/synnaxlabs/synnax/pkg/api/mqtt"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var apiSvc *apimqtt.Service

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	apiSvc = MustSucceed(apimqtt.NewService(apimqtt.ServiceConfig{
		MQTT: svc.MQTTFactory,
		RBAC: svc.RBAC,
	}))
})

// closedPort returns a local port that no server listens on.
func closedPort() int {
	GinkgoHelper()
	l := MustSucceed(net.Listen("tcp", "127.0.0.1:0"))
	port := l.Addr().(*net.TCPAddr).Port
	Expect(l.Close()).To(Succeed())
	return port
}

var _ = Describe("Service", func() {
	var (
		u   user.User
		req apimqtt.TestRequest
	)

	BeforeEach(func(ctx SpecContext) {
		u = createUser(ctx)
		req = apimqtt.TestRequest{
			Location:   "127.0.0.1",
			Properties: msgpack.EncodedJSON{"port": closedPort()},
		}
	})

	Describe("Test", func() {
		It("Should refuse a new broker without create access on devices",
			func(ctx SpecContext) {
				Expect(apiSvc.Test(AuthedCtx(ctx, u), req)).Error().
					To(MatchError(access.ErrDenied))
			},
		)

		It("Should test a new broker with create access on devices",
			func(ctx SpecContext) {
				grant(
					ctx,
					access.ActionCreate,
					u.OntologyID(),
					ontology.ID{Type: ontology.ResourceTypeDevice},
				)
				res := MustSucceed(apiSvc.Test(AuthedCtx(ctx, u), req))
				Expect(res.Steps).To(HaveLen(1))
				Expect(res.Steps[0].Stage).To(Equal("dial"))
				Expect(res.Steps[0].Variant).To(Equal(status.VariantError))
			},
		)

		It("Should refuse an edit of a device without update access",
			func(ctx SpecContext) {
				req.Device = "broker"
				grant(
					ctx,
					access.ActionRetrieve,
					u.OntologyID(),
					device.OntologyID("broker"),
				)
				Expect(apiSvc.Test(AuthedCtx(ctx, u), req)).Error().
					To(MatchError(access.ErrDenied))
			},
		)
	})

	It("Should refuse a browse without retrieve access", func(ctx SpecContext) {
		Expect(apiSvc.Browse(AuthedCtx(ctx, u), apimqtt.BrowseRequest{
			Device: "broker",
			Filter: "#",
		})).Error().To(MatchError(access.ErrDenied))
	})

	It("Should refuse a sample without retrieve access", func(ctx SpecContext) {
		Expect(apiSvc.Sample(AuthedCtx(ctx, u), apimqtt.SampleRequest{
			Device: "broker",
			Topics: []string{"plant/a"},
		})).Error().To(MatchError(access.ErrDenied))
	})
})
