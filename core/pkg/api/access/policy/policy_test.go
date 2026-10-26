// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apipolicy "github.com/synnaxlabs/synnax/pkg/api/access/policy"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
)

var policyType = ontology.ID{Type: ontology.ResourceTypePolicy}

var _ = Describe("Retrieve", func() {
	It("Should return the policies of a subject when retrieve is granted", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		p := grantOn(ctx, u.OntologyID(), access.ActionRetrieve, policyType)
		res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u), apipolicy.RetrieveRequest{
			Where: &policy.Where{Subjects: []ontology.ID{u.OntologyID()}},
		}))
		Expect(res.Policies).To(HaveExactElements(HaveField("Key", p.Key)))
	})

	It("Should reject the request when retrieve is not granted on a match", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		p := grantOn(ctx, u.OntologyID(), access.ActionUpdate, u.OntologyID())
		Expect(apiSvc.Retrieve(AuthedCtx(ctx, u), apipolicy.RetrieveRequest{
			Where: &policy.Where{Key: &filter.Set[policy.Key]{Eq: &p.Key}},
		})).Error().To(MatchError(access.ErrDenied))
	})
})
