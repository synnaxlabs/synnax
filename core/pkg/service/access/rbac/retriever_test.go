// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rbac_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer", func() {
	var (
		db        *gorp.DB
		writer    rbac.Writer
		retriever rbac.Retriever
		svc       *rbac.Service
	)
	BeforeEach(func() {
		db = gorp.Wrap(memkv.New())
		svc = MustSucceed(rbac.OpenService(ctx, rbac.Config{DB: db}))
		writer = svc.NewWriter(nil)
		retriever = svc.NewRetrieve()
	})
	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})
	It("Should retrieve a policy", func() {
		// Giving a user the rights to change their own password
		Expect(writer.Create(ctx, &changePasswordPolicy)).To(Succeed())
		var policy rbac.Policy
		err := retriever.Entry(&policy).WhereSubjects(userID).Exec(ctx, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(policy).To(Equal(changePasswordPolicy))
	})
	It("Should retrieve a policy by key", func() {
		Expect(writer.Create(ctx, &changePasswordPolicy)).To(Succeed())
		var policy rbac.Policy
		err := retriever.Entry(&policy).WhereKeys(changePasswordPolicy.Key).Exec(ctx, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(policy).To(Equal(changePasswordPolicy))

	})
})
