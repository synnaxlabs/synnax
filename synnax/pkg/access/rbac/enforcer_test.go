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
	"github.com/synnaxlabs/synnax/pkg/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
)

var _ = Describe("enforcer", func() {
	var (
		db         *gorp.DB
		legislator *rbac.Legislator
		enforcer   access.Enforcer
	)
	BeforeEach(func() {
		db = gorp.Wrap(memkv.New())
		legislator = &rbac.Legislator{DB: db}
		enforcer = &rbac.Enforcer{
			DefaultEffect: access.Deny,
			Legislator:    legislator,
		}
		txn := db.OpenTx()
		Expect(legislator.Create(txn, changePasswordPolicy)).To(Succeed())
		Expect(txn.Commit(ctx)).To(Succeed())
	})
	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})
	It("Should allow access when a valid policy exists", func() {
		Expect(enforcer.Enforce(ctx, access.Request{
			Subject: userID,
			Objects: userID,
			Action:  "changePassword",
		})).To(BeNil())
	})
	It("Should return the default effect when a policy can't be found", func() {
		Expect(enforcer.Enforce(ctx, access.Request{
			Subject: user.OntologyID(uuid.New()),
			Objects: userID,
			Action:  "changePassword",
		})).To(Equal(access.Denied))
	})
	It("Should return the default effect when no policy applies to the request", func() {
		Expect(enforcer.Enforce(ctx, access.Request{
			Subject: userID,
			Objects: userID,
			Action:  "retrieve",
		})).To(Equal(access.Denied))
	})
})
