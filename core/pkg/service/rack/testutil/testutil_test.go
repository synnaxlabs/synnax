// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("CreateDriverRole", func() {
	It("Should create the role", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		roles := MustOpen(role.OpenService(ctx, role.ServiceConfig{
			DB: db,
		}))
		k := MustSucceed(racktestutil.CreateDriverRole(ctx, roles))
		var r role.Role
		Expect(roles.NewRetrieve().
			Where(role.MatchKeys(k)).
			Entry(&r).
			Exec(ctx, nil)).To(Succeed())
		Expect(r.Name).To(Equal("Driver"))
	})
})
