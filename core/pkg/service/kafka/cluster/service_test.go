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
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/kafka/cluster"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Service", func() {
	var (
		rackSvc *rack.Service
		svc     *cluster.Service
		tx      gorp.Tx
	)
	BeforeEach(func(ctx SpecContext) {
		MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
		rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:         db,
			HostKey:    1,
			Status:     stat,
			DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roleSvc)),
		}))
		svc = MustOpen(cluster.OpenService(ctx, cluster.ServiceConfig{DB: db}))
		tx = DeferClose(db.OpenTx())
	})

	create := func(ctx context.Context, name string) cluster.Cluster {
		GinkgoHelper()
		s := cluster.Cluster{
			Key:       uuid.New(),
			Rack:      rackSvc.EmbeddedKey,
			Name:      name,
			Transport: cluster.Transport{Variant: cluster.PlaintextTransport{}},
			Brokers:   []cluster.Broker{{Host: "plc.example.com"}},
			Auth:      cluster.Auth{Variant: cluster.NoneAuth{}},
		}
		Expect(gorp.NewCreate[cluster.Key, cluster.Cluster]().
			Entry(&s).
			Exec(ctx, tx)).To(Succeed())
		return s
	}

	Describe("OpenService", func() {
		It("Should reject a config without a database", func(ctx SpecContext) {
			Expect(cluster.OpenService(ctx, cluster.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})

		It("Should let a later config override an earlier one", func() {
			cfg := cluster.ServiceConfig{}.Override(cluster.ServiceConfig{DB: db})
			Expect(cfg.DB).To(BeIdenticalTo(db))
			Expect(cfg.Validate()).To(Succeed())
		})
	})

	Describe("NewRetrieve", func() {
		It("Should retrieve a stored cluster by its key", func(ctx SpecContext) {
			s := create(ctx, "Boiler Cluster")
			var res cluster.Cluster
			Expect(svc.NewRetrieve().
				Where(cluster.MatchKeys(s.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal(s))
		})

		It("Should return not found for a missing key", func(ctx SpecContext) {
			var res cluster.Cluster
			Expect(svc.NewRetrieve().
				Where(cluster.MatchKeys(uuid.New())).
				Entry(&res).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})
	})

	It("Should refuse to delete a rack that holds a cluster", func(ctx SpecContext) {
		other := rack.Rack{Name: "Other"}
		Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
		s := cluster.Cluster{
			Key:       uuid.New(),
			Rack:      other.Key,
			Name:      "Gateway",
			Transport: cluster.Transport{Variant: cluster.PlaintextTransport{}},
			Brokers:   []cluster.Broker{{Host: "gateway.example.com"}},
			Auth:      cluster.Auth{Variant: cluster.NoneAuth{}},
		}
		Expect(gorp.NewCreate[cluster.Key, cluster.Cluster]().
			Entry(&s).
			Exec(ctx, tx)).To(Succeed())
		Expect(rackSvc.NewWriter(tx).Delete(ctx, nil, other.Key)).To(MatchError(
			ContainSubstring("because kafka_cluster %s references it", s.Key),
		))
		Expect(svc.NewRetrieve().
			Where(cluster.MatchKeys(s.Key)).
			Exists(ctx, tx)).To(BeTrue())
	})

	Describe("Resources", func() {
		It("Should name a cluster by its resource ID", func(ctx SpecContext) {
			s := create(ctx, "Boiler Cluster")
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{s.ResourceID()})).
				To(Equal([]resource.Resource{{ID: s.ResourceID(), Name: s.Name}}))
		})

		It("Should find a cluster by its name", func(ctx SpecContext) {
			s := create(ctx, "Boiler Cluster")
			res := MustSucceed(svc.Resources().Search(
				ctx,
				tx,
				search.Query[resource.ID]{Term: "boiler", Limit: 10},
			))
			Expect(res.Keys).To(ContainElement(s.ResourceID()))
		})

		It("Should reject a resource ID with a malformed key", func(ctx SpecContext) {
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{{
				Type: resource.TypeKafkaCluster,
				Key:  "plc",
			}})).Error().To(MatchError(validate.ErrValidation))
		})
	})
})
