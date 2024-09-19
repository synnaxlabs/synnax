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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbac2 "github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer", func() {
	var (
		db        *gorp.DB
		writer    rbac2.Writer
		retriever rbac2.Retriever
		svc       *rbac2.Service
	)
	BeforeEach(func() {
		db = gorp.Wrap(memkv.New())
		svc = MustSucceed(rbac2.NewService(rbac2.Config{DB: db}))
		writer = svc.NewWriter(nil)
		retriever = svc.NewRetriever()
	})
	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})
	It("Should create a new policy", func() {
		Expect(writer.Create(ctx, &changePasswordPolicy)).To(Succeed())
	})
	It("Should delete a policy", func() {
		Expect(writer.Create(ctx, &changePasswordPolicy)).To(Succeed())
		Expect(writer.Delete(ctx, changePasswordPolicy.Key)).To(Succeed())
		var p rbac2.Policy
		Expect(retriever.Entry(&p).WhereSubject(userID).Exec(ctx, nil)).To(MatchError(query.NotFound))
	})
})
