// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v3 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v3"
)

var _ = Describe("Session", func() {
	It("Should use its key as the gorp key", func() {
		key := uuid.New()
		Expect(v3.Session{Key: key}.GorpKey()).To(Equal(key))
	})
	It("Should return no options", func() {
		Expect(v3.Session{}.SetOptions()).To(BeNil())
	})
})
