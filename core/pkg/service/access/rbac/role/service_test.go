// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	Describe("Config", func() {
		Describe("Validate", func() {
			It("Should validate properly shaped configs", func() {
				cfg := role.ServiceConfig{DB: db, Ontology: otg, Group: groupSvc}
				Expect(cfg.Validate()).To(Succeed())
			})
			It("Should fail to validate configs missing the DB field", func() {
				cfg := role.ServiceConfig{Ontology: otg, Group: groupSvc}
				Expect(cfg.Validate()).To(HaveOccurred())
			})
			It("Should fail to validate configs missing the Ontology field", func() {
				cfg := role.ServiceConfig{DB: db, Group: groupSvc}
				Expect(cfg.Validate()).To(HaveOccurred())
			})
			It("Should fail to validate configs missing the Group field", func() {
				cfg := role.ServiceConfig{DB: db, Ontology: otg}
				Expect(cfg.Validate()).To(HaveOccurred())
			})
		})
		Describe("Override", func() {
			It("Should override properly fields with non-nil values", func() {
				cfg := role.ServiceConfig{}
				other := role.ServiceConfig{DB: db, Ontology: otg, Group: groupSvc}
				Expect(cfg.Override(other)).To(Equal(role.ServiceConfig{DB: db, Ontology: otg, Group: groupSvc}))
			})
		})
	})
	Describe("Open", func() {
		It("Should open a service with valid configuration", func() {
			s := MustSucceed(role.OpenService(ctx, role.ServiceConfig{DB: db, Ontology: otg, Group: groupSvc}))
			Expect(s.Close()).To(Succeed())
		})
		It("Should fail to open a service with an invalid configuration", func() {
			Expect(role.OpenService(ctx)).Error().To(HaveOccurred())
		})
	})
	Describe("UsersGroup", func() {
		It("Should return the users group", func() {
			g := svc.UsersGroup()
			Expect(g.Name).To(Equal("Users"))
			var parent ontology.Resource
			Expect(otg.NewRetrieve().WhereIDs(g.OntologyID()).
				TraverseTo(ontology.Parents).
				Entry(&parent).
				Exec(ctx, db)).To(Succeed())
			Expect(parent.ID).To(Equal(ontology.RootID))
		})
	})

})
