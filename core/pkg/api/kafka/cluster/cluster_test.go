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
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
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

var (
	clusterType = resource.ID{Type: resource.TypeKafkaCluster}
	statusType  = resource.ID{Type: resource.TypeStatus}
	channelType = resource.ID{Type: resource.TypeChannel}
	allWrites   = []access.Action{
		access.ActionCreate,
		access.ActionUpdate,
		access.ActionDelete,
	}
)

var _ = Describe("Service", func() {
	var u user.User
	BeforeEach(func(ctx SpecContext) { u = createUser(ctx) })

	newCluster := func(overrides ...func(*cluster.Cluster)) cluster.Cluster {
		var s cluster.Cluster
		s.SetDefaults()
		s.Key = uuid.New()
		s.Rack = svc.Rack.EmbeddedKey
		s.Name = "API-" + uuid.New().String()
		s.Brokers = []cluster.Broker{{Host: "kafka.example.com"}}
		for _, override := range overrides {
			override(&s)
		}
		return s
	}
	grantRetrieve := func(ctx SpecContext) {
		GinkgoHelper()
		grant(ctx, u, []access.Action{access.ActionRetrieve}, clusterType)
	}
	// grantWrites lets u write any Kafka cluster, its status, and the rack it sits on.
	grantWrites := func(ctx SpecContext, r rack.Key) {
		GinkgoHelper()
		grant(ctx, u, allWrites, clusterType, statusType)
		grant(ctx, u, []access.Action{access.ActionUpdate}, r.ResourceID())
	}
	retrieveCluster := func(ctx SpecContext, key cluster.Key) cluster.Cluster {
		GinkgoHelper()
		var s cluster.Cluster
		Expect(svc.Kafka.Clusters.NewRetrieve().
			Where(cluster.MatchKeys(key)).
			Entry(&s).
			Exec(ctx, db)).To(Succeed())
		return s
	}
	createCluster := func(
		ctx SpecContext,
		overrides ...func(*cluster.Cluster),
	) cluster.Cluster {
		GinkgoHelper()
		s := newCluster(overrides...)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Kafka.Clusters.NewWriter(tx).Create(ctx, &s)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Kafka.Clusters.NewWriter(tx).Delete(ctx, s.Key)
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

	Describe("Create", func() {
		It("Should mint a key when writes are granted", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			s := newCluster()
			s.Key = uuid.Nil()
			res := MustSucceed(create(
				testutil.AuthedCtx(ctx, u),
				apicluster.CreateRequest{Clusters: []cluster.Cluster{s}},
			))
			key := res.Clusters[0].Key
			Expect(key).ToNot(Equal(uuid.Nil()))
			Expect(retrieveCluster(ctx, key).Name).To(Equal(s.Name))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Kafka.Clusters.NewWriter(tx).Delete(ctx, key)
			})).To(Succeed())
		})

		It("Should not echo a status the client sent", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			s := newCluster()
			s.Status = &cluster.Status{Message: "Cluster connected"}
			res := MustSucceed(create(
				testutil.AuthedCtx(ctx, u),
				apicluster.CreateRequest{Clusters: []cluster.Cluster{s}},
			))
			Expect(res.Clusters[0].Status).To(BeNil())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Kafka.Clusters.NewWriter(tx).Delete(ctx, s.Key)
			})).To(Succeed())
		})

		It("Should reject the request without update access on the rack", func(
			ctx SpecContext,
		) {
			grant(ctx, u, allWrites, clusterType, statusType)
			s := newCluster()
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apicluster.CreateRequest{Clusters: []cluster.Cluster{s}},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should reject the request when create is not granted", func(
			ctx SpecContext,
		) {
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apicluster.CreateRequest{Clusters: []cluster.Cluster{newCluster()}},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should reject an invalid cluster", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apicluster.CreateRequest{Clusters: []cluster.Cluster{
					newCluster(
						func(s *cluster.Cluster) { s.Brokers = nil },
					),
				}},
			)).Error().To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Update", func() {
		rename := func(s cluster.Cluster) apicluster.UpdateRequest {
			return apicluster.UpdateRequest{
				Key: s.Key,
				Update: cluster.Update{Variant: cluster.IdentityUpdate{
					Identity: cluster.Identity{
						Rack:      s.Rack,
						Name:      "Renamed",
						Brokers:   s.Brokers,
						Transport: s.Transport,
						Auth:      s.Auth,
					},
				}},
			}
		}

		It("Should write only the identity group", func(ctx SpecContext) {
			chs := MustSucceed(channeltestutil.CreateVirtual(ctx, db, svc.Channel, 2))
			s := createCluster(ctx, func(s *cluster.Cluster) {
				s.Reads = []cluster.Read{{
					Topic:   "plant",
					Message: uuid.New(),
					Index:   chs[0],
					Channel: chs[1],
				}}
			})
			grantWrites(ctx, s.Rack)
			grant(ctx, u, []access.Action{access.ActionRetrieve}, channelType)
			MustSucceed(update(testutil.AuthedCtx(ctx, u), rename(s)))
			got := retrieveCluster(ctx, s.Key)
			Expect(got.Name).To(Equal("Renamed"))
			Expect(got.Reads).To(Equal(s.Reads))
		})

		// readsOf returns one read of the channels chs.
		readsOf := func(chs []channel.Key) []cluster.Read {
			return []cluster.Read{{
				Topic:     "plant",
				RecordKey: new("pump"),
				Message:   uuid.UUID{1},
				Index:     chs[0],
				Channel:   chs[1],
			}}
		}
		// bindReads returns an update that replaces the reads of s with reads.
		bindReads := func(s cluster.Cluster, reads []cluster.Read) apicluster.UpdateRequest {
			return apicluster.UpdateRequest{
				Key: s.Key,
				Update: cluster.Update{Variant: cluster.ConfigurationUpdate{
					Configuration: cluster.Configuration{Reads: reads},
				}},
			}
		}

		It("Should write only the configuration group", func(ctx SpecContext) {
			chs := MustSucceed(channeltestutil.CreateVirtual(ctx, db, svc.Channel, 2))
			s := createCluster(ctx)
			grantWrites(ctx, s.Rack)
			grant(ctx, u, []access.Action{access.ActionRetrieve}, channelType)
			reads := readsOf(chs)
			MustSucceed(update(testutil.AuthedCtx(ctx, u), bindReads(s, reads)))
			got := retrieveCluster(ctx, s.Key)
			Expect(got.Reads).To(Equal(reads))
			Expect(got.Name).To(Equal(s.Name))
		})

		It("Should reject a read of a channel the user cannot retrieve", func(
			ctx SpecContext,
		) {
			chs := MustSucceed(channeltestutil.CreateVirtual(ctx, db, svc.Channel, 2))
			s := createCluster(ctx)
			grantWrites(ctx, s.Rack)
			Expect(update(testutil.AuthedCtx(ctx, u), bindReads(s, readsOf(chs)))).
				Error().To(MatchError(access.ErrDenied))
			Expect(retrieveCluster(ctx, s.Key).Reads).To(BeEmpty())
		})

		It("Should move a cluster only with update access on the new rack", func(
			ctx SpecContext,
		) {
			other := rack.Rack{
				Name:         "Other",
				Integrations: []string{cluster.Integration},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Rack.NewWriter(tx).Create(ctx, &other)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.Rack.NewWriter(tx).Delete(ctx, nil, other.Key)
				})).To(Succeed())
			})
			s := createCluster(ctx)
			move := apicluster.UpdateRequest{
				Key: s.Key,
				Update: cluster.Update{Variant: cluster.IdentityUpdate{
					Identity: cluster.Identity{
						Rack:      other.Key,
						Name:      s.Name,
						Brokers:   s.Brokers,
						Transport: s.Transport,
						Auth:      s.Auth,
					},
				}},
			}
			grantWrites(ctx, s.Rack)
			Expect(update(testutil.AuthedCtx(ctx, u), move)).
				Error().To(MatchError(access.ErrDenied))
			grant(ctx, u, []access.Action{access.ActionUpdate}, other.ResourceID())
			MustSucceed(update(testutil.AuthedCtx(ctx, u), move))
			Expect(retrieveCluster(ctx, s.Key).Rack).To(Equal(other.Key))
		})

		It("Should reject a request with no group", func(ctx SpecContext) {
			s := createCluster(ctx)
			grantWrites(ctx, s.Rack)
			Expect(update(
				testutil.AuthedCtx(ctx, u),
				apicluster.UpdateRequest{Key: s.Key},
			)).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should reject the request when update is not granted", func(
			ctx SpecContext,
		) {
			s := createCluster(ctx)
			Expect(update(testutil.AuthedCtx(ctx, u), rename(s))).
				Error().To(MatchError(access.ErrDenied))
			Expect(retrieveCluster(ctx, s.Key).Name).To(Equal(s.Name))
		})
	})

	Describe("Delete", func() {
		It("Should remove the cluster when delete is granted", func(ctx SpecContext) {
			s := createCluster(ctx)
			grantWrites(ctx, s.Rack)
			MustSucceed(del(
				testutil.AuthedCtx(ctx, u),
				apicluster.DeleteRequest{Keys: []cluster.Key{s.Key}},
			))
			Expect(svc.Kafka.Clusters.NewRetrieve().
				Where(cluster.MatchKeys(s.Key)).
				Exists(ctx, db)).To(BeFalse())
		})

		It("Should reject the request when delete is not granted", func(
			ctx SpecContext,
		) {
			s := createCluster(ctx)
			Expect(del(
				testutil.AuthedCtx(ctx, u),
				apicluster.DeleteRequest{Keys: []cluster.Key{s.Key}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveCluster(ctx, s.Key).Key).To(Equal(s.Key))
		})
	})
})
