// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rbac_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/access"
	"github.com/synnaxlabs/synnax/pkg/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("enforcer", func() {
	var (
		db     *gorp.DB
		writer rbac.Writer
		svc    *rbac.Service
	)
	BeforeEach(func() {
		db = gorp.Wrap(memkv.New())
		svc = MustSucceed(rbac.NewService(rbac.Config{DB: db}))
		writer = svc.NewWriter(nil)
		Expect(writer.Create(ctx, &changePasswordPolicy)).To(Succeed())
	})
	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})
	It("Should allow access when a valid policy exists", func() {
		Expect(svc.Enforce(ctx, access.Request{
			Subject: userID,
			Objects: []ontology.ID{userID},
			Action:  "changePassword",
		})).To(Succeed())
	})
	It("Should deny when a policy can't be found", func() {
		Expect(svc.Enforce(ctx, access.Request{
			Subject: user.OntologyID(uuid.New()),
			Objects: []ontology.ID{userID},
			Action:  "changePassword",
		})).To(Equal(access.Denied))
	})
	It("Should deny when no policy applies to the request", func() {
		Expect(svc.Enforce(ctx, access.Request{
			Subject: userID,
			Objects: []ontology.ID{userID},
			Action:  "retrieve",
		})).To(Equal(access.Denied))
	})
	It("Should deny when the policy is removed", func() {
		Expect(writer.Delete(ctx, changePasswordPolicy.Key)).To(Succeed())
		Expect(svc.Enforce(ctx, access.Request{
			Subject: userID,
			Objects: []ontology.ID{userID},
			Action:  "changePassword",
		})).To(Equal(access.Denied))
	})
	It("Should deny in the case of mix-matching", func() {
		var (
			userID1, userID2 = user.OntologyID(uuid.New()), user.OntologyID(uuid.New())
			user1Change2     = rbac.Policy{
				Subjects: []ontology.ID{userID1},
				Objects:  []ontology.ID{userID2},
				Actions:  []access.Action{"changePassword"},
			}
			user2Change1 = rbac.Policy{
				Subjects: []ontology.ID{userID2},
				Objects:  []ontology.ID{userID1},
				Actions:  []access.Action{"changePassword", "erasePassword"},
			}
		)
		Expect(writer.Create(ctx, &user1Change2)).To(Succeed())
		Expect(writer.Create(ctx, &user2Change1)).To(Succeed())

		Expect(svc.Enforce(ctx, access.Request{
			Subject: userID1,
			Objects: []ontology.ID{userID1},
			Action:  "changePassword",
		})).To(Equal(access.Denied))

		Expect(svc.Enforce(ctx, access.Request{
			Subject: userID2,
			Objects: []ontology.ID{userID2},
			Action:  "changePassword",
		})).To(Equal(access.Denied))

		Expect(svc.Enforce(ctx, access.Request{
			Subject: userID1,
			Objects: []ontology.ID{userID2},
			Action:  "erasePassword",
		})).To(Equal(access.Denied))

		Expect(svc.Enforce(ctx, access.Request{
			Subject: userID2,
			Objects: []ontology.ID{userID1},
			Action:  "erasePassword",
		})).To(Succeed())
	})
})
