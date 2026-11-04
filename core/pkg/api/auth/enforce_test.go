// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("EnforceOthers", func() {
	var nobody user.User
	BeforeEach(func(ctx SpecContext) {
		nobody = MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
			Name: "nobody",
		}))
	})
	It("Should let a caller act on itself without a policy", func(ctx SpecContext) {
		Expect(apiauth.EnforceOthers(
			ctx,
			rbacSvc,
			nobody.OntologyID(),
			access.ActionUpdate,
			[]ontology.ID{nobody.OntologyID()},
		)).To(Succeed())
	})
	It("Should let a caller with a policy act on another subject", func(
		ctx SpecContext,
	) {
		Expect(apiauth.EnforceOthers(
			ctx,
			rbacSvc,
			root.OntologyID(),
			access.ActionUpdate,
			[]ontology.ID{nobody.OntologyID()},
		)).To(Succeed())
	})
	It("Should deny a caller without a policy on another subject", func(
		ctx SpecContext,
	) {
		Expect(apiauth.EnforceOthers(
			ctx,
			rbacSvc,
			nobody.OntologyID(),
			access.ActionUpdate,
			[]ontology.ID{nobody.OntologyID(), root.OntologyID()},
		)).To(MatchError(access.ErrDenied))
	})
})
