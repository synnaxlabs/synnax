// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errors_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/errors"
)

var _ = Describe("PB", func() {
	It("Should translate a payload to its protobuf representation", func() {
		pld := errors.Payload{Type: "Cat", Data: "Orange"}
		pb := errors.TranslatePayloadForward(pld)
		Expect(pb.Type).To(Equal("Cat"))
		Expect(pb.Data).To(Equal("Orange"))
	})

	It("Should translate a protobuf representation to a payload", func() {
		pb := &errors.PBPayload{Type: "Cat", Data: "Orange"}
		pld := errors.TranslatePayloadBackward(pb)
		Expect(pld.Type).To(Equal("Cat"))
		Expect(pld.Data).To(Equal("Orange"))
	})
})
