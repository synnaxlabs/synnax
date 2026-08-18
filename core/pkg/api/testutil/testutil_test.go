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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
)

var _ = Describe("AuthedCtx", func() {
	It("Should install the user as the request subject", func(ctx SpecContext) {
		u := user.User{Key: uuid.New(), Username: "author"}
		Expect(auth.GetSubject(AuthedCtx(ctx, u))).To(Equal(u.OntologyID()))
	})
})
