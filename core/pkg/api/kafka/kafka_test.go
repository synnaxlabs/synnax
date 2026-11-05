// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apikafka "github.com/synnaxlabs/synnax/pkg/api/kafka"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/twmb/franz-go/pkg/kfake"
)

var (
	apiSvc  *apikafka.Service
	cluster *kfake.Cluster
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	apiSvc = MustSucceed(apikafka.NewService(apikafka.ServiceConfig{
		Device: svc.Device,
		RBAC:   svc.RBAC,
	}))
	cluster = MustSucceed(kfake.NewCluster(
		kfake.NumBrokers(1),
		kfake.SeedTopics(1, "readings"),
	))
	DeferCleanup(cluster.Close)
})

var _ = Describe("Service", func() {
	var (
		u     user.User
		dev   device.Device
		props kafka.Properties
	)

	BeforeEach(func(ctx SpecContext) {
		u = createUser(ctx)
		props = kafka.Properties{Brokers: cluster.ListenAddrs()}
		dev = device.Device{
			Key:        uuid.New().String(),
			Rack:       svc.Rack.EmbeddedKey,
			Location:   "kafka",
			Make:       kafka.Make,
			Model:      "cluster",
			Name:       "cluster",
			Configured: true,
			Properties: msgpack.EncodedJSON{"brokers": props.Brokers},
		}
		Expect(svc.Device.NewWriter(nil).Create(ctx, &dev)).To(Succeed())
	})

	Describe("Test", func() {
		It("Should refuse a new cluster without create access on devices",
			func(ctx SpecContext) {
				Expect(apiSvc.Test(AuthedCtx(ctx, u), apikafka.TestRequest{
					Properties: props,
				})).Error().To(MatchError(access.ErrDenied))
			},
		)

		It("Should test a new cluster with create access on devices",
			func(ctx SpecContext) {
				grant(
					ctx,
					access.ActionCreate,
					u.OntologyID(),
					ontology.ID{Type: ontology.ResourceTypeDevice},
				)
				res := MustSucceed(apiSvc.Test(AuthedCtx(ctx, u), apikafka.TestRequest{
					Properties: props,
				}))
				Expect(res.Steps).ToNot(BeEmpty())
				for _, step := range res.Steps {
					Expect(step.Variant).To(Equal(status.VariantSuccess))
				}
			},
		)

		It("Should refuse an edit of a device without update access",
			func(ctx SpecContext) {
				grant(
					ctx,
					access.ActionRetrieve,
					u.OntologyID(),
					device.OntologyID(dev.Key),
				)
				Expect(apiSvc.Test(AuthedCtx(ctx, u), apikafka.TestRequest{
					Device:     dev.Key,
					Properties: props,
				})).Error().To(MatchError(access.ErrDenied))
			},
		)
	})

	Describe("Browse", func() {
		It("Should refuse a user without retrieve access", func(ctx SpecContext) {
			Expect(apiSvc.Browse(AuthedCtx(ctx, u), apikafka.BrowseRequest{
				Device: dev.Key,
			})).Error().To(MatchError(access.ErrDenied))
		})

		It("Should list the topics of a cluster device", func(ctx SpecContext) {
			grant(
				ctx,
				access.ActionRetrieve,
				u.OntologyID(),
				device.OntologyID(dev.Key),
			)
			res := MustSucceed(apiSvc.Browse(AuthedCtx(ctx, u), apikafka.BrowseRequest{
				Device: dev.Key,
			}))
			Expect(res.Topics).To(HaveLen(1))
			Expect(res.Topics[0].Name).To(Equal("readings"))
		})
	})

	Describe("Sample", func() {
		It("Should refuse a user without retrieve access", func(ctx SpecContext) {
			Expect(apiSvc.Sample(AuthedCtx(ctx, u), apikafka.SampleRequest{
				Device: dev.Key,
				Topics: []string{"readings"},
			})).Error().To(MatchError(access.ErrDenied))
		})

		It("Should sample the topics of a cluster device", func(ctx SpecContext) {
			grant(
				ctx,
				access.ActionRetrieve,
				u.OntologyID(),
				device.OntologyID(dev.Key),
			)
			res := MustSucceed(apiSvc.Sample(AuthedCtx(ctx, u), apikafka.SampleRequest{
				Device: dev.Key,
				Topics: []string{"readings"},
			}))
			Expect(res.Topics).To(HaveLen(1))
			Expect(res.Topics[0].Topic).To(Equal("readings"))
			Expect(res.Topics[0].Error).To(BeEmpty())
			Expect(res.Topics[0].Records).To(BeEmpty())
		})
	})
})
