// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package union_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/union"
)

func TestUnion(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Union Suite")
}

var _ = Describe("MissingPayload", func() {
	It("Should wrap ErrMissingPayload so errors.Is matches the sentinel", func() {
		err := union.MissingPayload("rename")
		Expect(errors.Is(err, union.ErrMissingPayload)).To(BeTrue())
	})

	It("Should include the variant name in the error message", func() {
		err := union.MissingPayload("set_node_position")
		Expect(err).To(MatchError(ContainSubstring(`variant "set_node_position"`)))
	})
})
