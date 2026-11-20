// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
)

var _ = Describe("ErrorProvider", func() {
	codec := errors.NewCodec(control.ErrorProvider())
	It("Should round-trip an unauthorized error", func() {
		pld := codec.Encode(errors.Wrap(control.ErrUnauthorized, "ch 5"), false)
		Expect(pld.Type).To(Equal("sy.control.unauthorized"))
		Expect(codec.Decode(pld)).To(SatisfyAll(
			MatchError(control.ErrUnauthorized),
			MatchError(ContainSubstring("ch 5")),
		))
	})
	It("Should decode another control subtype as a control error", func() {
		err := codec.Decode(errors.Payload{Type: "sy.control.other", Data: "x"})
		Expect(err).To(MatchError(control.ErrControl))
		Expect(err).ToNot(MatchError(control.ErrUnauthorized))
	})
	It("Should leave other errors to other providers", func() {
		Expect(codec.Encode(errors.New("cat"), false).Type).
			To(Equal(errors.TypeUnknown))
	})
})
