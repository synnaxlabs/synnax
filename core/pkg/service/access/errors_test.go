// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package access_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var _ = Describe("Denied", func() {
	It("Should wrap ErrDenied and name the subject, action, and object", func() {
		err := access.Denied(
			resource.ID{Type: resource.TypeUser, Key: "u1"},
			access.ActionUpdate,
			resource.ID{Type: resource.TypeRange, Key: "r1"},
		)
		Expect(err).To(MatchError(access.ErrDenied))
		Expect(err).To(MatchError(
			"user:u1 cannot update range:r1: access denied: auth error",
		))
	})
})
