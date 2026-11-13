// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("WriteEndpoint", func() {
	It("Should reject a null write field", func() {
		var endpoint v4.WriteEndpoint
		Expect(json.Unmarshal([]byte(`{"fields": [null]}`), &endpoint)).To(Succeed())
		Expect(endpoint.Validate()).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("fields.0.type: required")),
		))
	})

	It("Should reject a write field without a type", func() {
		var endpoint v4.WriteEndpoint
		Expect(json.Unmarshal([]byte(`{"fields": [{}]}`), &endpoint)).
			To(MatchError(validate.ErrValidation))
	})
})
