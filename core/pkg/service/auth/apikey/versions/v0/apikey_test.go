// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/auth/apikey/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

var _ = Describe("APIKey", func() {
	It("Should use its key as the gorp key", func() {
		key := uuid.New()
		Expect(v0.APIKey{Key: key}.GorpKey()).To(Equal(key))
	})
	It("Should return no options", func() {
		Expect(v0.APIKey{}.SetOptions()).To(BeNil())
	})
	It("Should validate its subject", func() {
		Expect(v0.APIKey{}.Validate()).To(MatchError(ContainSubstring("subject")))
		Expect(v0.APIKey{Subject: ontology.ID{
			Type: ontology.ResourceTypeRack,
			Key:  "1",
		}}.Validate()).To(Succeed())
	})
})
