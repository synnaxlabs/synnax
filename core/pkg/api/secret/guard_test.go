// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apisecret "github.com/synnaxlabs/synnax/pkg/api/secret"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
)

var _ = Describe("Guard", func() {
	var (
		readable resource.ID
		guard    func([]resource.ID) error
	)
	BeforeEach(func(ctx SpecContext) {
		u := createUser(ctx)
		readable = secret.ResourceID(uuid.New())
		allow(ctx, u.ResourceID(), []access.Action{access.ActionRetrieve}, readable)
		guard = apisecret.Guard(AuthedCtx(ctx, u), svc.RBAC.NewEnforcer(nil))
	})

	It("Should allow a list with no resources", func() {
		Expect(guard(nil)).To(Succeed())
	})

	It("Should not check resources that are not secrets", func() {
		Expect(guard([]resource.ID{{Type: resource.TypeChannel, Key: "1"}})).
			To(Succeed())
	})

	It("Should allow secrets that the subject can retrieve", func() {
		Expect(guard([]resource.ID{readable})).To(Succeed())
	})

	It("Should deny a list with a secret that the subject cannot retrieve", func() {
		hidden := secret.ResourceID(uuid.New())
		Expect(guard([]resource.ID{readable, hidden})).To(MatchError(access.ErrDenied))
	})
})
