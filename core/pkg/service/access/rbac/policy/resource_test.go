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
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	It("Should name each policy by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		p := policy.Policy{
			Role:    owner.Key,
			Name:    "resource-policy",
			Objects: []resource.ID{{Type: resource.TypeChannel, Key: "1"}},
			Actions: []access.Action{access.ActionRetrieve},
		}
		Expect(svc.NewWriter(tx, false).Create(ctx, &p)).To(Succeed())
		Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{p.ResourceID()})).
			To(Equal([]resource.Resource{{ID: p.ResourceID(), Name: p.Name}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			nil,
			[]resource.ID{{Type: resource.TypePolicy, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid policy key "pump"`)),
		))
	})
})
