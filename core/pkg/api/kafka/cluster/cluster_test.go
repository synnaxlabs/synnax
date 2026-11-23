// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cluster_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apicluster "github.com/synnaxlabs/synnax/pkg/api/kafka/cluster"
	"github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/kafka/cluster"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var clusterType = resource.ID{Type: resource.TypeKafkaCluster}

var _ = Describe("Service", func() {
	var u user.User
	BeforeEach(func(ctx SpecContext) { u = createUser(ctx) })

	newCluster := func(overrides ...func(*cluster.Cluster)) cluster.Cluster {
		s := cluster.Cluster{
			Key:       uuid.New(),
			Rack:      svc.Rack.EmbeddedKey,
			Name:      "API-" + uuid.New().String(),
			Transport: cluster.Transport{Variant: cluster.PlaintextTransport{}},
			Brokers:   []cluster.Broker{{Host: "plc.example.com"}},
			Auth:      cluster.Auth{Variant: cluster.NoneAuth{}},
		}
		for _, override := range overrides {
			override(&s)
		}
		return s
	}
	grantRetrieve := func(ctx SpecContext) {
		GinkgoHelper()
		grant(ctx, u, []access.Action{access.ActionRetrieve}, clusterType)
	}
	createCluster := func(ctx SpecContext, overrides ...func(*cluster.Cluster)) cluster.Cluster {
		GinkgoHelper()
		s := newCluster(overrides...)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return gorp.NewCreate[cluster.Key, cluster.Cluster]().Entry(&s).
				Exec(ctx, tx)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return gorp.NewDelete[cluster.Key, cluster.Cluster]().
					Where(gorp.MatchKeys[cluster.Key, cluster.Cluster](s.Key)).
					Exec(ctx, tx)
			})).To(Succeed())
		})
		return s
	}

	Describe("NewService", func() {
		DescribeTable("Should reject a config missing a required field",
			func(field string, unset func(*apicluster.ServiceConfig)) {
				cfg := apicluster.ServiceConfig{
					Cluster: svc.Kafka.Clusters,
					Status:  svc.Status,
				}
				unset(&cfg)
				Expect(apicluster.NewService(cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("cluster", "cluster", func(c *apicluster.ServiceConfig) {
				c.Cluster = nil
			}),
			Entry("status", "status", func(c *apicluster.ServiceConfig) {
				c.Status = nil
			}),
		)

		It("Should let a later config override an earlier one", func() {
			base := apicluster.ServiceConfig{Cluster: svc.Kafka.Clusters}
			Expect(apicluster.NewService(base, apicluster.ServiceConfig{
				Status: svc.Status,
			})).ToNot(BeNil())
		})
	})

	Describe("Retrieve", func() {
		It("Should reject a named cluster when retrieve is not granted", func(
			ctx SpecContext,
		) {
			key := createCluster(ctx).Key
			Expect(retrieve(
				testutil.AuthedCtx(ctx, u),
				apicluster.RetrieveRequest{
					Where: &cluster.Where{Key: &filter.Set[cluster.Key]{Eq: &key}},
				},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should filter by rack and attach the status", func(ctx SpecContext) {
			s := createCluster(ctx)
			createCluster(ctx)
			st := cluster.Status{
				Owner:   new(s.ResourceID()),
				Name:    s.Name,
				Time:    telem.Now(),
				Variant: status.VariantSuccess,
				Message: "Cluster connected",
				Details: cluster.StatusDetails{Cluster: s.Key, Connected: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Status.NewWriter(tx).Set(ctx, &st)
			})).To(Succeed())
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apicluster.RetrieveRequest{
					Where: &cluster.Where{
						Name: &filter.String{Eq: &s.Name},
						Rack: &filter.Set[rack.Key]{Eq: &s.Rack},
					},
					IncludeStatus: true,
				},
			))
			Expect(res.Clusters).To(HaveLen(1))
			Expect(res.Clusters[0].Key).To(Equal(s.Key))
			Expect(res.Clusters[0].Status.Message).To(Equal("Cluster connected"))
			Expect(res.Clusters[0].Status.Details.Connected).To(BeTrue())
		})

		It("Should leave out the clusters the subject cannot retrieve", func(
			ctx SpecContext,
		) {
			name := createCluster(ctx).Name
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apicluster.RetrieveRequest{
					Where: &cluster.Where{Name: &filter.String{Eq: &name}},
				},
			))
			Expect(res.Clusters).To(BeEmpty())
		})

		It("Should find a cluster by its name", func(ctx SpecContext) {
			d := createCluster(
				ctx,
				func(s *cluster.Cluster) { s.Name = "Boiler Cluster" },
			)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apicluster.RetrieveRequest{SearchTerm: "boiler plc"},
			))
			Expect(lo.Map(res.Clusters, func(s cluster.Cluster, _ int) cluster.Key {
				return s.Key
			})).To(ContainElement(d.Key))
		})

		It("Should cap the results at the limit", func(ctx SpecContext) {
			a := createCluster(ctx)
			b := createCluster(ctx)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apicluster.RetrieveRequest{
					Where: &cluster.Where{Key: &filter.Set[cluster.Key]{
						In: []cluster.Key{a.Key, b.Key},
					}},
					Limit: new(uint32(1)),
				},
			))
			Expect(res.Clusters).To(HaveLen(1))
		})

		It("Should page past the after key", func(ctx SpecContext) {
			a := createCluster(ctx)
			b := createCluster(ctx)
			grantRetrieve(ctx)
			authed := testutil.AuthedCtx(ctx, u)
			req := apicluster.RetrieveRequest{
				Where: &cluster.Where{Key: &filter.Set[cluster.Key]{
					In: []cluster.Key{a.Key, b.Key},
				}},
				Limit: new(uint32(1)),
			}
			first := MustSucceed(retrieve(authed, req))
			Expect(first.Clusters).To(HaveLen(1))
			req.After = &apicluster.Cursor{Key: first.Clusters[0].Key}
			second := MustSucceed(retrieve(authed, req))
			Expect(second.Clusters).To(HaveLen(1))
			Expect(second.Clusters[0].Key).
				ToNot(Equal(first.Clusters[0].Key))
		})

		It("Should reject a search that resumes after a key", func(ctx SpecContext) {
			d := createCluster(ctx)
			grantRetrieve(ctx)
			Expect(retrieve(
				testutil.AuthedCtx(ctx, u),
				apicluster.RetrieveRequest{
					SearchTerm: d.Name,
					After:      &apicluster.Cursor{Key: d.Key},
				},
			)).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should return a path error for a malformed where", func(ctx SpecContext) {
			grantRetrieve(ctx)
			Expect(retrieve(testutil.AuthedCtx(ctx, u), apicluster.RetrieveRequest{
				Where: &cluster.Where{Name: &filter.String{Regex: new("(a")}},
			})).Error().To(SatisfyAll(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError(ContainSubstring("where.name.regex: error parsing regexp")),
				MatchError(validate.ErrValidation),
			))
		})

		It("Should return no clusters for a zero limit", func(ctx SpecContext) {
			createCluster(ctx)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apicluster.RetrieveRequest{Limit: new(uint32(0))},
			))
			Expect(res.Clusters).To(BeEmpty())
		})

		It("Should return no clusters for a missing key", func(ctx SpecContext) {
			grantRetrieve(ctx)
			missing := uuid.New()
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apicluster.RetrieveRequest{
					Where: &cluster.Where{Key: &filter.Set[cluster.Key]{Eq: &missing}},
				},
			))
			Expect(res.Clusters).To(BeEmpty())
		})
	})
})
