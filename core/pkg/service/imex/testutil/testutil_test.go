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
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

type resource struct {
	Name  string `json:"name"`
	Field string `json:"field"`
}

var _ = Describe("WireRoundTrip", func() {
	It("Should preserve headers and bind a codec so the envelope decodes", func(
		ctx SpecContext,
	) {
		env := imex.Envelope{Version: 1, Type: "test", Name: "roundtrip"}
		r := resource{Name: "roundtrip", Field: "value"}
		Expect(imex.Encode(&env, r)).To(Succeed())
		out := WireRoundTrip(env)
		Expect(out.Version).To(Equal(imex.Version(1)))
		Expect(out.Type).To(Equal("test"))
		Expect(out.Name).To(Equal("roundtrip"))
		Expect(MustSucceed(imex.Decode[resource](ctx, out))).To(Equal(r))
	})
})
